package main

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/whatsapp-groupe4/internal/logger"
)

type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	ChatID    string    `json:"chat_id"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

var (
	messages = make(map[string]Message)
	mu       sync.RWMutex
)

func main() {
	logger.Init("message-service")
	defer logger.Close()

	router := gin.Default()

	port := getEnv("PORT", "8082")

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "message-service",
		})
	})

	// Routes messages
	api := router.Group("/api/v1/messages")
	{
		api.GET("", getAllMessages)
		api.GET("/:id", getMessageByID)
		api.GET("/chat/:chatId", getMessagesByChat)
		api.POST("", createMessage)
		api.DELETE("/:id", deleteMessage)
	}

	logger.Info("Message Service démarré sur le port %s", port)

	if err := router.Run(":" + port); err != nil {
		logger.Fatal("Erreur démarrage serveur: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getAllMessages(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

	messageList := make([]Message, 0, len(messages))
	for _, msg := range messages {
		messageList = append(messageList, msg)
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messageList,
		"count":    len(messageList),
	})
}

func getMessageByID(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	msg, exists := messages[id]
	mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message non trouvé"})
		return
	}

	c.JSON(http.StatusOK, msg)
}

func getMessagesByChat(c *gin.Context) {
	chatID := c.Param("chatId")

	mu.RLock()
	defer mu.RUnlock()

	chatMessages := make([]Message, 0)
	for _, msg := range messages {
		if msg.ChatID == chatID {
			chatMessages = append(chatMessages, msg)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": chatMessages,
		"chat_id":  chatID,
		"count":    len(chatMessages),
	})
}

func createMessage(c *gin.Context) {
	var input struct {
		SenderID string `json:"sender_id" binding:"required"`
		Content  string `json:"content" binding:"required"`
		ChatID   string `json:"chat_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := Message{
		ID:        uuid.New().String(),
		SenderID:  input.SenderID,
		Content:   input.Content,
		ChatID:    input.ChatID,
		CreatedAt: time.Now(),
		Status:    "sent",
	}

	mu.Lock()
	messages[msg.ID] = msg
	mu.Unlock()

	c.JSON(http.StatusCreated, msg)
}

func deleteMessage(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := messages[id]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message non trouvé"})
		return
	}

	delete(messages, id)
	c.JSON(http.StatusOK, gin.H{"message": "Message supprimé"})
}
