package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgxPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository interface {
	CreateChannel(ctx context.Context, ch Channel) (Channel, error)
	GetChannelByID(ctx context.Context, id string) (Channel, error)
	UpdateChannel(ctx context.Context, id string, req UpdateChannelRequest) (Channel, error)
	DeleteChannel(ctx context.Context, id string) error
	ListChannelsByUser(ctx context.Context, userID string) ([]ChannelResponse, error)

	AddMember(ctx context.Context, chatID, userID string) (Participant, error)
	RemoveMember(ctx context.Context, chatID, userID string) error
	IsMember(ctx context.Context, chatID, userID string) (bool, error)
	ListMembers(ctx context.Context, chatID string) ([]Participant, error)
	CountMembers(ctx context.Context, chatID string) (int, error)

	ListMessages(ctx context.Context, chatID, cursor string, limit int) ([]Message, error)
}

const errChannelNotFound = "channel not found"

type pgRepository struct {
	pool pgxPool
}

func NewRepository(pool pgxPool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CreateChannel(ctx context.Context, ch Channel) (Channel, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var created Channel
	err := r.pool.QueryRow(ctx,
		`INSERT INTO chats (name, description, is_group, owner_id, max_members)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, description, is_group, owner_id, max_members, created_at`,
		ch.Name, ch.Description, ch.IsGroup, ch.OwnerID, ch.MaxMembers,
	).Scan(
		&created.ID, &created.Name, &created.Description,
		&created.IsGroup, &created.OwnerID, &created.MaxMembers, &created.CreatedAt,
	)
	return created, err
}

func (r *pgRepository) GetChannelByID(ctx context.Context, id string) (Channel, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ch Channel
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, COALESCE(description,''), is_group, owner_id, COALESCE(max_members,1000), created_at
		 FROM chats WHERE id = $1`, id,
	).Scan(&ch.ID, &ch.Name, &ch.Description, &ch.IsGroup, &ch.OwnerID, &ch.MaxMembers, &ch.CreatedAt)
	if err == pgx.ErrNoRows {
		return ch, fmt.Errorf(errChannelNotFound)
	}
	return ch, err
}

func (r *pgRepository) UpdateChannel(ctx context.Context, id string, req UpdateChannelRequest) (Channel, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ch Channel
	err := r.pool.QueryRow(ctx,
		`UPDATE chats
		 SET name        = COALESCE($1, name),
		     description = COALESCE($2, description),
		     max_members = COALESCE($3, max_members)
		 WHERE id = $4
		 RETURNING id, name, COALESCE(description,''), is_group, owner_id, COALESCE(max_members,1000), created_at`,
		req.Name, req.Description, req.MaxMembers, id,
	).Scan(&ch.ID, &ch.Name, &ch.Description, &ch.IsGroup, &ch.OwnerID, &ch.MaxMembers, &ch.CreatedAt)
	if err == pgx.ErrNoRows {
		return ch, fmt.Errorf(errChannelNotFound)
	}
	return ch, err
}

func (r *pgRepository) DeleteChannel(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tag, err := r.pool.Exec(ctx, `DELETE FROM chats WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf(errChannelNotFound)
	}
	return nil
}

func (r *pgRepository) ListChannelsByUser(ctx context.Context, userID string) ([]ChannelResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.name, COALESCE(c.description,''), c.is_group, c.owner_id,
		        COALESCE(c.max_members,1000), c.created_at,
		        (SELECT COUNT(*) FROM chat_participants cp2 WHERE cp2.chat_id = c.id) AS member_count
		 FROM chats c
		 INNER JOIN chat_participants cp ON cp.chat_id = c.id
		 WHERE cp.user_id = $1
		 ORDER BY c.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]ChannelResponse, 0, 32)
	for rows.Next() {
		var cr ChannelResponse
		if err := rows.Scan(
			&cr.Channel.ID, &cr.Channel.Name, &cr.Channel.Description,
			&cr.Channel.IsGroup, &cr.Channel.OwnerID, &cr.Channel.MaxMembers,
			&cr.Channel.CreatedAt, &cr.MemberCount,
		); err != nil {
			return nil, err
		}
		results = append(results, cr)
	}
	return results, rows.Err()
}

// --- Members ---

func (r *pgRepository) AddMember(ctx context.Context, chatID, userID string) (Participant, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var p Participant
	err := r.pool.QueryRow(ctx,
		`INSERT INTO chat_participants (chat_id, user_id)
		 VALUES ($1, $2)
		 ON CONFLICT (chat_id, user_id) DO NOTHING
		 RETURNING chat_id, user_id, joined_at`,
		chatID, userID,
	).Scan(&p.ChatID, &p.UserID, &p.JoinedAt)
	if err == pgx.ErrNoRows {
		return p, fmt.Errorf("user is already a member")
	}
	return p, err
}

func (r *pgRepository) RemoveMember(ctx context.Context, chatID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tag, err := r.pool.Exec(ctx,
		`DELETE FROM chat_participants WHERE chat_id = $1 AND user_id = $2`,
		chatID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *pgRepository) IsMember(ctx context.Context, chatID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id = $1 AND user_id = $2)`,
		chatID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *pgRepository) ListMembers(ctx context.Context, chatID string) ([]Participant, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pool.Query(ctx,
		`SELECT chat_id, user_id, joined_at
		 FROM chat_participants
		 WHERE chat_id = $1
		 ORDER BY joined_at ASC`, chatID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]Participant, 0, 64)
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ChatID, &p.UserID, &p.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, p)
	}
	return members, rows.Err()
}

func (r *pgRepository) CountMembers(ctx context.Context, chatID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM chat_participants WHERE chat_id = $1`, chatID,
	).Scan(&count)
	return count, err
}

// --- Messages (keyset pagination) ---

func (r *pgRepository) ListMessages(ctx context.Context, chatID, cursor string, limit int) ([]Message, error) {
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
