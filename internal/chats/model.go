package chats

import "time"

type Chat struct {
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	Type         string    `json:"type"` 
	Participants []string  `json:"participants"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateChatRequest struct {
	Participants []string `json:"participants" binding:"required,min=1"`
	Type         string   `json:"type" binding:"required,oneof=private group"`
	Name         string   `json:"name"`
}