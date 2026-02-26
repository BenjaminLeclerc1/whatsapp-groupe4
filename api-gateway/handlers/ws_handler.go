package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	// "whatsapp-groupe4/ws"
	// "whatsapp-groupe4/auth"
	"whatsapp-groupe4/api-gateway/ws"
	"whatsapp-groupe4/user-service/auth" // Assume you moved auth validation here
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func WsHandler(hub *ws.Hub) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Validate JWT
        token := c.Query("token")
        claims, err := auth.ValidateToken(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }

        // CORRECTED: Dereference the pointer and access the map key
        // Then perform a type assertion to string
        email, ok := (*claims)["email"].(string)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
            return
        }

        // 2. Upgrade HTTP to WebSocket
        conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil {
            return // Should handle upgrade error properly in production
        }
        
        hub.Register(email, conn)
        
        // 3. Keep connection alive
        for {
            _, _, err := conn.ReadMessage()
            if err != nil { break }
        }
    }
}