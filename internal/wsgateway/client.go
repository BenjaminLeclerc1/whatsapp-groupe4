package wsgateway

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/whatsapp-groupe4/internal/logger"
	"github.com/whatsapp-groupe4/internal/middleware"
)

const (
	maxMessageSize = 4096
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10

	sendBufSize = 64

	msgRateLimit  = 30
	msgRateWindow = 10 * time.Second
	maxContentLen = 2000
)

// Client represents a single authenticated WebSocket connection.
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	UserID string
	Send   chan []byte

	mu    sync.RWMutex
	rooms map[string]struct{}

	rateMu     sync.Mutex
	rateTokens int
	rateReset  time.Time

	closeOnce sync.Once

	// LastActivity is updated on every received message or pong.
	// The Hub sweeper uses it to detect zombie connections.
	LastActivity atomic.Int64 // unix nanoseconds
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	c := &Client{
		Hub:        hub,
		Conn:       conn,
		UserID:     userID,
		Send:       make(chan []byte, sendBufSize),
		rooms:      make(map[string]struct{}, 8),
		rateTokens: msgRateLimit,
		rateReset:  time.Now().Add(msgRateWindow),
	}
	c.touch()
	return c
}

func (c *Client) touch() {
	c.LastActivity.Store(time.Now().UnixNano())
}

// IdleDuration returns how long since last activity.
func (c *Client) IdleDuration() time.Duration {
	last := c.LastActivity.Load()
	return time.Since(time.Unix(0, last))
}

func (c *Client) allowMessage() bool {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	now := time.Now()
	if now.After(c.rateReset) {
		c.rateTokens = msgRateLimit
		c.rateReset = now.Add(msgRateWindow)
	}
	if c.rateTokens <= 0 {
		return false
	}
	c.rateTokens--
	return true
}

// SendWelcome sends connection metadata so the client knows the heartbeat
// contract and any restored room subscriptions.
func (c *Client) SendWelcome(restoredRooms []string) {
	c.sendJSON(Envelope{
		Type:              EventWelcome,
		UserID:            c.UserID,
		HeartbeatInterval: int(pingPeriod.Seconds()),
		HeartbeatTimeout:  int(pongWait.Seconds()),
		RestoredRooms:     restoredRooms,
	})
}

// ReadPump reads messages from the WebSocket and dispatches them to the hub.
// Must run in its own goroutine — one per Client.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.touch()
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Error("ws read user=%s: %v", c.UserID, err)
			}
			return
		}

		c.touch()

		if !c.allowMessage() {
			c.sendError("rate limit exceeded")
			continue
		}

		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			c.sendError("invalid json")
			continue
		}

		c.dispatch(env)
	}
}

// WritePump pumps messages from Send channel to the WebSocket.
// Must run in its own goroutine — one per Client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(msg)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte("\n"))
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) dispatch(env Envelope) {
	switch env.Type {
	case EventPing:
		c.sendJSON(Envelope{Type: EventPong})

	case EventSubscribe:
		chatID := strings.TrimSpace(env.ChatID)
		if !middleware.IsValidUUID(chatID) {
			c.sendError("invalid chat_id")
			return
		}
		c.mu.Lock()
		c.rooms[chatID] = struct{}{}
		c.mu.Unlock()
		c.Hub.Subscribe(c, chatID)
		c.Hub.SaveSession(c.UserID, chatID)
		c.sendJSON(Envelope{Type: EventSubscribed, ChatID: chatID})

	case EventUnsubscribe:
		chatID := strings.TrimSpace(env.ChatID)
		if !middleware.IsValidUUID(chatID) {
			c.sendError("invalid chat_id")
			return
		}
		c.mu.Lock()
		delete(c.rooms, chatID)
		c.mu.Unlock()
		c.Hub.UnsubscribeRoom(c, chatID)
		c.Hub.RemoveSession(c.UserID, chatID)
		c.sendJSON(Envelope{Type: EventUnsubscribed, ChatID: chatID})

	case EventMessage:
		chatID := strings.TrimSpace(env.ChatID)
		content := strings.TrimSpace(env.Content)
		if !middleware.IsValidUUID(chatID) {
			c.sendError("invalid chat_id")
			return
		}
		if content == "" || len(content) > maxContentLen {
			c.sendError("content empty or too long")
			return
		}
		now := time.Now()
		out := Envelope{
			Type:      EventNewMessage,
			ChatID:    chatID,
			SenderID:  c.UserID,
			Content:   content,
			CreatedAt: &now,
		}
		c.Hub.BroadcastToRoom(chatID, out, c.UserID)

	case EventTyping:
		chatID := strings.TrimSpace(env.ChatID)
		if !middleware.IsValidUUID(chatID) || env.Typing == nil {
			c.sendError("invalid chat_id or missing typing")
			return
		}
		out := Envelope{
			Type:   EventTypingUpdate,
			ChatID: chatID,
			UserID: c.UserID,
			Typing: env.Typing,
		}
		c.Hub.BroadcastToRoom(chatID, out, c.UserID)

	default:
		c.sendError("unknown event type")
	}
}

func (c *Client) sendJSON(env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	select {
	case c.Send <- data:
	default:
		c.Hub.Unregister <- c
	}
}

func (c *Client) sendError(msg string) {
	c.sendJSON(Envelope{Type: EventError, Message: msg})
}

// CloseSend safely closes the Send channel exactly once.
func (c *Client) CloseSend() {
	c.closeOnce.Do(func() { close(c.Send) })
}

// Rooms returns a snapshot of chat IDs the client is subscribed to.
func (c *Client) Rooms() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, 0, len(c.rooms))
	for r := range c.rooms {
		out = append(out, r)
	}
	return out
}
