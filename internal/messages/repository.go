package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateMessage(ctx context.Context, chatID, senderID, content string) (Message, error)
	GetMessageByID(ctx context.Context, id string) (Message, error)
	ListMessagesByChat(ctx context.Context, chatID, cursor string, limit int) ([]Message, error)
	DeleteMessage(ctx context.Context, id, senderID string) error
	MessageExists(ctx context.Context, id string) (bool, error)
	IsChatMember(ctx context.Context, chatID, userID string) (bool, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

// CreateMessage uses a CTE to verify membership and insert in a single round-trip.
// Returns pgx.ErrNoRows when the user is not a member of the chat.
// internal/messages/repository.go

// internal/messages/repository.go

func (r *pgRepository) CreateMessage(ctx context.Context, chatID, senderID, content string) (Message, error) {
    // 1. Check if the chat actually exists first
    var chatExists bool
    err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1::uuid)", chatID).Scan(&chatExists)
    if !chatExists {
        return Message{}, fmt.Errorf("chat_not_found: %s", chatID)
    }

    // 2. Check if the user is a member
    var isMember bool
    err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id = $1::uuid AND user_id = $2::uuid)", chatID, senderID).Scan(&isMember)
    if !isMember {
        return Message{}, fmt.Errorf("user_not_participant: %s", senderID)
    }

    // 3. Now perform the insert
    var m Message
    query := `
        INSERT INTO messages (sender_id, chat_id, content, status)
        VALUES ($1::uuid, $2::uuid, $3, 'sent')
        RETURNING id::text, sender_id::text, chat_id::text, content, status, created_at`

    err = r.pool.QueryRow(ctx, query, senderID, chatID, content).Scan(
        &m.ID, &m.SenderID, &m.ChatID, &m.Content, &m.Status, &m.CreatedAt,
    )

    return m, err
}

func (r *pgRepository) GetMessageByID(ctx context.Context, id string) (Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var m Message
	err := r.pool.QueryRow(ctx,
		`SELECT id, COALESCE(sender_id::text,''), chat_id, content, status, created_at
		 FROM messages WHERE id = $1`, id,
	).Scan(&m.ID, &m.SenderID, &m.ChatID, &m.Content, &m.Status, &m.CreatedAt)
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

	if cursor == "" {
		rows, err = r.pool.Query(ctx,
			`SELECT id, COALESCE(sender_id::text,''), chat_id, content, status, created_at
			 FROM messages
			 WHERE chat_id = $1
			 ORDER BY created_at DESC, id DESC
			 LIMIT $2`,
			chatID, limit,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, COALESCE(sender_id::text,''), chat_id, content, status, created_at
			 FROM messages
			 WHERE chat_id = $1
			   AND (created_at, id) < (
			       SELECT created_at, id FROM messages WHERE id = $2
			   )
			 ORDER BY created_at DESC, id DESC
			 LIMIT $3`,
			chatID, cursor, limit,
		)
	}
	if err != nil {
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

// DeleteMessage only deletes if the caller is the sender (WHERE sender_id = $2).
func (r *pgRepository) DeleteMessage(ctx context.Context, id, senderID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	tag, err := r.pool.Exec(ctx,
		`DELETE FROM messages WHERE id = $1 AND sender_id = $2`,
		id, senderID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete_no_rows")
	}
	return nil
}

func (r *pgRepository) MessageExists(ctx context.Context, id string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`, id,
	).Scan(&exists)
	return exists, err
}

func (r *pgRepository) IsChatMember(ctx context.Context, chatID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id = $1 AND user_id = $2)`,
		chatID, userID,
	).Scan(&exists)
	return exists, err
}
