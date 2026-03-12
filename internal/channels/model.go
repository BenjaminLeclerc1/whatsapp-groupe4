package channels

import "time"

type Channel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsGroup     bool      `json:"is_group"`
	OwnerID     string    `json:"owner_id"`
	MaxMembers  int       `json:"max_members"`
	CreatedAt   time.Time `json:"created_at"`
}

type Participant struct {
	ChatID   string    `json:"chat_id"`
	UserID   string    `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	ChatID    string    `json:"chat_id"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Request DTOs ---

type CreateChannelRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=500"`
	IsGroup     bool   `json:"is_group"`
	MaxMembers  int    `json:"max_members" binding:"omitempty,min=2,max=1000"`
}

type UpdateChannelRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	MaxMembers  *int    `json:"max_members" binding:"omitempty,min=2,max=1000"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// --- Response DTOs ---

type ChannelResponse struct {
	Channel      Channel `json:"channel"`
	MemberCount  int     `json:"member_count"`
}

type ChannelListResponse struct {
	Channels []ChannelResponse `json:"channels"`
	Count    int               `json:"count"`
}

type MemberListResponse struct {
	Members []Participant `json:"members"`
	Count   int           `json:"count"`
}

type MessageListResponse struct {
	Messages []Message `json:"messages"`
	ChatID   string    `json:"chat_id"`
	Count    int       `json:"count"`
	Cursor   string    `json:"cursor,omitempty"`
}
