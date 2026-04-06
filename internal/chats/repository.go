package chats

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, chat Chat) error
	FindByUser(ctx context.Context, userID string) ([]Chat, error)

	UpdateName(ctx context.Context, id string, name string) error // <--- NOUVEAU
    Delete(ctx context.Context, id string) error                 // <--- NOUVEAU
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


// Implémentation de UpdateName
func (r *repository) UpdateName(ctx context.Context, id string, name string) error {
    query := `UPDATE chats SET name = $1 WHERE id = $2`
    _, err := r.db.Exec(ctx, query, name, id)
    return err
}

// Implémentation de Delete
func (r *repository) Delete(ctx context.Context, id string) error {
    query := `DELETE FROM chats WHERE id = $1`
    _, err := r.db.Exec(ctx, query, id)
    return err
}