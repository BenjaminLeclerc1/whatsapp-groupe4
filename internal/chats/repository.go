package chats

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, chat Chat) error
	FindByUser(ctx context.Context, userID string) ([]Chat, error)
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, c Chat) error {
	query := `INSERT INTO chats (id, name, type, participants, created_at) 
              VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(ctx, query, c.ID, c.Name, c.Type, c.Participants, c.CreatedAt)
	return err
}

func (r *repository) FindByUser(ctx context.Context, userID string) ([]Chat, error) {
	query := `SELECT id, name, type, participants, created_at 
              FROM chats WHERE $1 = ANY(participants)`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Participants, &c.CreatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, nil
}