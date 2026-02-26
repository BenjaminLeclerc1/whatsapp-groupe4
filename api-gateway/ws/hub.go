package ws

import (
	"sync"
	"github.com/gorilla/websocket"
)

type Hub struct {
	// Map UserID to their active WebSocket connection
	Clients    map[string]*websocket.Conn
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{Clients: make(map[string]*websocket.Conn)}
}

func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Clients[userID] = conn
}