package wsgateway

import "time"

// EventType identifies the kind of WebSocket frame.
type EventType string

const (
	EventSubscribe   EventType = "subscribe"
	EventUnsubscribe EventType = "unsubscribe"
	EventMessage     EventType = "message"
	EventTyping      EventType = "typing"
	EventPing        EventType = "ping"

	EventPong            EventType = "pong"
	EventNewMessage      EventType = "new_message"
	EventTypingUpdate    EventType = "typing_update"
	EventPresence        EventType = "presence"
	EventError           EventType = "error"
	EventSubscribed      EventType = "subscribed"
	EventUnsubscribed    EventType = "unsubscribed"
	EventWelcome         EventType = "welcome"
	EventSessionRestored EventType = "session_restored"
)

// Envelope is the wire format for every WebSocket frame (JSON).
type Envelope struct {
	Type      EventType `json:"type"`
	ChatID    string    `json:"chat_id,omitempty"`
	Content   string    `json:"content,omitempty"`
	MessageID string    `json:"message_id,omitempty"`
	SenderID  string    `json:"sender_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Status    string    `json:"status,omitempty"`
	Typing    *bool     `json:"typing,omitempty"`
	Message   string    `json:"message,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`

	// welcome payload
	HeartbeatInterval int      `json:"heartbeat_interval,omitempty"`
	HeartbeatTimeout  int      `json:"heartbeat_timeout,omitempty"`
	RestoredRooms     []string `json:"restored_rooms,omitempty"`
}

// RedisBroadcast wraps an Envelope with routing metadata for cross-instance
// delivery via Redis Pub/Sub.
type RedisBroadcast struct {
	ChatID      string   `json:"chat_id"`
	Envelope    Envelope `json:"envelope"`
	ExcludeUser string   `json:"exclude_user,omitempty"`
}
