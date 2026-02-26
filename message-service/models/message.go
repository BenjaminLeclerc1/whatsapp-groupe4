
package models

import (
    "time" // <--- You must add this line
)


type Message struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    SenderID  string    `json:"sender_id" gorm:"index"`
    ChatID    string    `json:"chat_id" gorm:"index"`
    Content   string    `json:"content"`
    Status    string    `json:"status"` // e.g., "sent", "delivered", "read"
    CreatedAt time.Time `json:"created_at"`
}