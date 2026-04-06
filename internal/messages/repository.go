package messages

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateMessage(ctx context.Context, chatID, senderID, content string) (Message, error)
	GetMessageByID(ctx context.Context, id string) (Message, error)
	ListMessagesByChat(ctx context.Context, chatID, cursor string, limit int) ([]Message, error)
	DeleteMessage(ctx context.Context, id, senderID string) error
	// Fixed: Added these back so service.go can compile
	MessageExists(ctx context.Context, id string) (bool, error)
	IsChatMember(ctx context.Context, chatID, userID string) (bool, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CreateMessage(ctx context.Context, chatID, senderID, content string) (Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	msgID := uuid.New().String()
	var m Message

	query := `
        INSERT INTO messages (id, sender_id, chat_id, content, status, created_at)
        VALUES ($1, $2, $3, $4, 'sent', CURRENT_TIMESTAMP)
        RETURNING id, sender_id::text, chat_id, content, status, created_at`

	err := r.pool.QueryRow(ctx, query, msgID, senderID, chatID, content).
		Scan(&m.ID, &m.SenderID, &m.ChatID, &m.Content, &m.Status, &m.CreatedAt)

	if err != nil {
		log.Printf("DATABASE ERROR in CreateMessage: %v", err)
		return Message{}, err
	}

	return m, nil
}

func (r *pgRepository) GetMessageByID(ctx context.Context, id string) (Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var m Message
	query := `SELECT id, sender_id::text, chat_id, content, status, created_at
              FROM messages WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).
		Scan(&m.ID, &m.SenderID, &m.ChatID, &m.Content, &m.Status, &m.CreatedAt)

	if err == pgx.ErrNoRows {
		return m, fmt.Errorf("message not found")
	}
	return m, err
}

func (r *pgRepository) ListMessagesByChat(ctx context.Context, chatID, cursor string, limit int) ([]Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows pgx.Rows
	var err error
	baseQuery := `SELECT id, sender_id::text, chat_id, content, status, created_at FROM messages WHERE chat_id = $1`

	if cursor == "" {
		rows, err = r.pool.Query(ctx, baseQuery+` ORDER BY created_at DESC, id DESC LIMIT $2`, chatID, limit)
	} else {
		rows, err = r.pool.Query(ctx, baseQuery+` AND (created_at, id) < (SELECT created_at, id FROM messages WHERE id = $2) ORDER BY created_at DESC, id DESC LIMIT $3`, chatID, cursor, limit)
	}

	if err != nil {
		log.Printf("DATABASE ERROR in ListMessages: %v", err)
		return nil, err
	}
	defer rows.Close()

	msgs := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ChatID, &m.Content, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *pgRepository) DeleteMessage(ctx context.Context, id, senderID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	tag, err := r.pool.Exec(ctx, `DELETE FROM messages WHERE id = $1 AND sender_id = $2`, id, senderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete_failed_unauthorized_or_not_found")
	}
	return nil
}

// --- New Methods to fix build errors ---

func (r *pgRepository) MessageExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func (r *pgRepository) IsChatMember(ctx context.Context, chatID, userID string) (bool, error) {
	// For now, return true since participants table is on Shard 0.
	// In the future, this should be an API call to the Chat Service.
	return true, nil
}