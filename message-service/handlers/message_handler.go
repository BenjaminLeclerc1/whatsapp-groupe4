package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"whatsapp-groupe4/message-service/models"
	"whatsapp-groupe4/message-service/repository"
)

type MessageHandler struct {
	Repo       *repository.MessageRepo
	NATSClient *nats.Conn
}

func (h *MessageHandler) CreateMessage(c *gin.Context) {
	var msg models.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// 1. Save to repo
	if err := h.Repo.Save(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save"})
		return
	}

	// 2. Publish to NATS
	data, _ := json.Marshal(msg)
	if err := h.NATSClient.Publish("messages.new", data); err != nil {
		log.Printf("NATS publish error: %v", err)
	}

	c.JSON(http.StatusCreated, msg)
}

func (h *MessageHandler) GetChatHistory(c *gin.Context) {
	chatID := c.Param("chat_id")
	messages, err := h.Repo.FindAllByChat(chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	c.JSON(http.StatusOK, messages)
}

func (h *MessageHandler) GetMessage(c *gin.Context) {
	msgID := c.Param("id")
	msg, err := h.Repo.FindByID(msgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	c.JSON(http.StatusOK, msg)
}

// DELETE (DELETE) - Now properly inside a function
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	msgID := c.Param("id")
	err := h.Repo.Delete(msgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}