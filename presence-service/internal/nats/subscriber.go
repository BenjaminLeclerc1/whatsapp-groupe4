// internal/nats/subscriber.go
package nats_subscriber

import (
    "encoding/json"
    "github.com/nats-io/nats.go"
    "whatsapp-group4/presence-service/websocket" // Your socket manager
)

func StartMessageConsumer(nc *nats.Conn, wsManager *websocket.Manager) {
    // Subscribe to the subject published by message-service
    nc.Subscribe("messages.new", func(m *nats.Msg) {
        var msg struct {
            RecipientID string `json:"recipient_id"`
            Content     string `json:"content"`
        }
        
        json.Unmarshal(m.Data, &msg)

        // Find the user's active WebSocket connection and send the message
        if conn, ok := wsManager.GetConnection(msg.RecipientID); ok {
            conn.WriteJSON(msg)
        }
    })
}