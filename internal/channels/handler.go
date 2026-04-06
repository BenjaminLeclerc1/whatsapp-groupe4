package channels

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/whatsapp-groupe4/internal/middleware"
)

const errInvalidChannelIDFormat = "invalid channel id format"

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	channels := rg.Group("/channels")
	{
		channels.POST("", h.CreateChannel)
		channels.GET("", h.ListMyChannels)
		channels.GET("/:id", h.GetChannel)
		channels.PUT("/:id", h.UpdateChannel)
		channels.DELETE("/:id", h.DeleteChannel)

		channels.POST("/:id/members", h.AddMember)
		channels.GET("/:id/members", h.ListMembers)
		channels.DELETE("/:id/members/:userId", h.RemoveMember)

		channels.GET("/:id/messages", h.ListMessages)
	}
}

func (h *Handler) CreateChannel(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.CreateChannel(c.Request.Context(), userID, req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) GetChannel(c *gin.Context) {
	userID := middleware.GetUserID(c)
	channelID := c.Param("id")

	if !middleware.IsValidUUID(channelID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidChannelIDFormat})
		return
	}

	resp, err := h.svc.GetChannel(c.Request.Context(), userID, channelID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateChannel(c *gin.Context) {
	userID := middleware.GetUserID(c)
	channelID := c.Param("id")

	if !middleware.IsValidUUID(channelID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidChannelIDFormat})
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.UpdateChannel(c.Request.Context(), userID, channelID, req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteChannel(c *gin.Context) {
	userID := middleware.GetUserID(c)
	channelID := c.Param("id")

	if !middleware.IsValidUUID(channelID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidChannelIDFormat})
		return
	}

	if err := h.svc.DeleteChannel(c.Request.Context(), userID, channelID); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "channel deleted"})
}

func (h *Handler) ListMyChannels(c *gin.Context) {
	userID := middleware.GetUserID(c)

	resp, err := h.svc.ListMyChannels(c.Request.Context(), userID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// --- Members ---

func (h *Handler) AddMember(c *gin.Context) {
	userID := middleware.GetUserID(c)
	channelID := c.Param("id")

	if !middleware.IsValidUUID(channelID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidChannelIDFormat})
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !middleware.IsValidUUID(req.UserID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
		return
	}

	p, err := h.svc.AddMember(c.Request.Context(), userID, channelID, req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (h *Handler) RemoveMember(c *gin.Context) {
	userID := middleware.GetUserID(c)
	channelID := c.Param("id")
	targetUserID := c.Param("userId")

	if !middleware.IsValidUUID(channelID) || !middleware.IsValidUUID(targetUserID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	if err := h.svc.RemoveMember(c.Request.Context(), userID, channelID, targetUserID); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}

func (h *Handler) ListMembers(c *gin.Context) {
	userID := middleware.GetUserID(c)
	channelID := c.Param("id")

	if !middleware.IsValidUUID(channelID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidChannelIDFormat})
		return
	}

	resp, err := h.svc.ListMembers(c.Request.Context(), userID, channelID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// --- Messages ---

func (h *Handler) ListMessages(c *gin.Context) {
	userID := middleware.GetUserID(c)
	channelID := c.Param("id")

	if !middleware.IsValidUUID(channelID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidChannelIDFormat})
		return
	}

	cursor, limit := middleware.ParsePagination(c)

	resp, err := h.svc.ListMessages(c.Request.Context(), userID, channelID, cursor, limit)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// writeServiceError maps domain error strings to appropriate HTTP status codes.
func writeServiceError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		c.JSON(http.StatusNotFound, gin.H{"error": msg})
	case strings.Contains(msg, "forbidden"):
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
	case strings.Contains(msg, "already a member"):
		c.JSON(http.StatusConflict, gin.H{"error": msg})
	case strings.Contains(msg, "maximum number of members"):
		c.JSON(http.StatusConflict, gin.H{"error": msg})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
