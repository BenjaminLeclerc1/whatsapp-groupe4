package messages

import (
	// "log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/whatsapp-groupe4/internal/middleware"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	msgs := rg.Group("/messages")
	{
		msgs.POST("", h.SendMessage)
		msgs.GET("/:id", h.GetMessage)
		msgs.GET("/chat/:chatId", h.GetMessageHistory)
		msgs.DELETE("/:id", h.DeleteMessage)
	}
}

// internal/messages/handler.go

func (h *Handler) SendMessage(c *gin.Context) {
    // 1. We define "userID" here
    userID := middleware.GetUserID(c)

    // 2. We define "req" here
    var req SendMessageRequest 
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if !middleware.IsValidUUID(req.ChatID) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat_id format"})
        return
    }

    // 3. Now "userID" and "req" exist and can be passed to the service
    msg, err := h.svc.SendMessage(c.Request.Context(), userID, req)
    if err != nil {
        writeServiceError(c, err)
        return
    }

    c.JSON(http.StatusCreated, MessageResponse{Message: msg})
}

func (h *Handler) GetMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	messageID := c.Param("id")

	if !middleware.IsValidUUID(messageID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id format"})
		return
	}

	msg, err := h.svc.GetMessage(c.Request.Context(), userID, messageID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: msg})
}

func (h *Handler) GetMessageHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	chatID := c.Param("chatId")

	if !middleware.IsValidUUID(chatID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat id format"})
		return
	}

	cursor, limit := middleware.ParsePagination(c)

	resp, err := h.svc.GetMessageHistory(c.Request.Context(), userID, chatID, cursor, limit)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	messageID := c.Param("id")

	if !middleware.IsValidUUID(messageID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id format"})
		return
	}

	if err := h.svc.DeleteMessage(c.Request.Context(), userID, messageID); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message deleted"})
}

func writeServiceError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		c.JSON(http.StatusNotFound, gin.H{"error": msg})
	case strings.Contains(msg, "forbidden"):
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
	case strings.Contains(msg, "validation"):
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
