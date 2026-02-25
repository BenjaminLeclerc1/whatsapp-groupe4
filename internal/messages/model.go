package messages

import "time"

type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	ChatID    string    `json:"chat_id"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Request DTOs ---

type SendMessageRequest struct {
	ChatID  string `json:"chat_id" binding:"required"`
	Content string `json:"content" binding:"required,min=1,max=5000"`
}

// --- Response DTOs ---

type MessageResponse struct {
	Message Message `json:"message"`
}

type MessageListResponse struct {
	Messages []Message `json:"messages"`
	ChatID   string    `json:"chat_id"`
	Count    int       `json:"count"`
	Cursor   string    `json:"cursor,omitempty"`
}
