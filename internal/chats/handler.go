package chats

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateChat(c *gin.Context) {
	userID := c.GetString("user_id") // Extracted from JWT by middleware
	var req CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chat, err := h.svc.CreateChat(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}
	c.JSON(http.StatusCreated, chat)
}

func (h *Handler) GetMyChats(c *gin.Context) {
	userID := c.GetString("user_id")
	chats, err := h.svc.GetMyChats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}
	c.JSON(http.StatusOK, chats)
}


// Ajoute ces fonctions dans ton fichier handler.go

func (h *Handler) UpdateChat(c *gin.Context) {
    chatID := c.Param("id") // Récupère l'ID dans l'URL
    var req struct {
        Name string `json:"name"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }

    // Appelle le service (tu devras ajouter cette méthode au service plus tard)
    err := h.svc.UpdateChat(c.Request.Context(), chatID, req.Name)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chat"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Chat updated successfully"})
}

func (h *Handler) DeleteChat(c *gin.Context) {
    chatID := c.Param("id")

    // Appelle le service (tu devras ajouter cette méthode au service plus tard)
    err := h.svc.DeleteChat(c.Request.Context(), chatID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Chat deleted successfully"})
}