package repository

import (
    "database/sql"
    "whatsapp-groupe4/message-service/models"
)

type MessageRepo struct {
    db *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
    return &MessageRepo{db: db}
}

// Ensure this method is defined exactly like this
func (r *MessageRepo) Save(msg models.Message) error {
    query := `INSERT INTO messages (id, sender_id, chat_id, content, status, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
    _, err := r.db.Exec(query, msg.ID, msg.SenderID, msg.ChatID, msg.Content, msg.Status, msg.CreatedAt)
    return err
}

// Ensure this method is defined exactly like this
func (r *MessageRepo) FindByID(id string) (models.Message, error) {
    var m models.Message
    query := `SELECT id, sender_id, chat_id, content, status, created_at FROM messages WHERE id = $1`
    err := r.db.QueryRow(query, id).Scan(&m.ID, &m.SenderID, &m.ChatID, &m.Content, &m.Status, &m.CreatedAt)
    return m, err
}

// ... include FindAllByChat and Delete as shown previously
func (r *MessageRepo) FindAllByChat(chatID string) ([]models.Message, error) {
    query := `SELECT id, sender_id, chat_id, content, status, created_at FROM messages WHERE chat_id = $1`
    rows, err := r.db.Query(query, chatID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var messages []models.Message
    for rows.Next() {
        var m models.Message
        err := rows.Scan(&m.ID, &m.SenderID, &m.ChatID, &m.Content, &m.Status, &m.CreatedAt)
        if err != nil {
            return nil, err
        }
        messages = append(messages, m)
    }
    return messages, nil
}

func (r *MessageRepo) Delete(msgID string) error {
    query := `DELETE FROM messages WHERE id = $1`
    _, err := r.db.Exec(query, msgID)
    return err
}