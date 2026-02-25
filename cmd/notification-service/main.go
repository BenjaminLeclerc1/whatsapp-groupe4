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

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	MessageID string    `json:"message_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	ChatID    string    `json:"chat_id"`
	Type      string    `json:"type"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationCount struct {
	UserID      string `json:"user_id"`
	UnreadCount int    `json:"unread_count"`
}

var (
	notifications = make(map[string]Notification)
	userNotifications = make(map[string][]string) // userID -> []notificationIDs
	mu            sync.RWMutex
)

func main() {
	logger.Init("notification-service")
	defer logger.Close()

	router := gin.Default()

	port := getEnv("PORT", "8083")

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "notification-service",
		})
	})

	// Routes notifications
	api := router.Group("/api/v1/notification")
	{
		// Obtenir le nombre de notifications non lues pour un utilisateur (pour l'icône avec badge)
		api.GET("/count/:userId", getUnreadCount)
		
		// Obtenir toutes les notifications d'un utilisateur
		api.GET("/user/:userId", getNotificationsByUser)
		
		// Obtenir les notifications non lues d'un utilisateur
		api.GET("/user/:userId/unread", getUnreadNotifications)
		
		// Obtenir une notification par ID
		api.GET("/:id", getNotificationByID)
		
		// Créer une notification (appelé quand un message est reçu)
		api.POST("", createNotification)
		
		// Marquer une notification comme lue
		api.PUT("/:id/read", markAsRead)
		
		// Marquer toutes les notifications d'un utilisateur comme lues
		api.PUT("/user/:userId/read-all", markAllAsRead)
		
		// Supprimer une notification
		api.DELETE("/:id", deleteNotification)
	}

	logger.Info("Notification Service démarré sur le port %s", port)

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

// getUnreadCount retourne le nombre de notifications non lues pour un utilisateur
// C'est l'endpoint principal pour l'icône avec badge qui s'auto-incrémente
func getUnreadCount(c *gin.Context) {
	userID := c.Param("userId")

	mu.RLock()
	defer mu.RUnlock()

	count := 0
	if notificationIDs, exists := userNotifications[userID]; exists {
		for _, notifID := range notificationIDs {
			if notif, ok := notifications[notifID]; ok && !notif.Read {
				count++
			}
		}
	}

	c.JSON(http.StatusOK, NotificationCount{
		UserID:      userID,
		UnreadCount: count,
	})
}

// getNotificationsByUser retourne toutes les notifications d'un utilisateur
func getNotificationsByUser(c *gin.Context) {
	userID := c.Param("userId")

	mu.RLock()
	defer mu.RUnlock()

	notificationList := make([]Notification, 0)
	if notificationIDs, exists := userNotifications[userID]; exists {
		for _, notifID := range notificationIDs {
			if notif, ok := notifications[notifID]; ok {
				notificationList = append(notificationList, notif)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notificationList,
		"user_id":      userID,
		"count":        len(notificationList),
	})
}

// getUnreadNotifications retourne uniquement les notifications non lues d'un utilisateur
func getUnreadNotifications(c *gin.Context) {
	userID := c.Param("userId")

	mu.RLock()
	defer mu.RUnlock()

	unreadList := make([]Notification, 0)
	if notificationIDs, exists := userNotifications[userID]; exists {
		for _, notifID := range notificationIDs {
			if notif, ok := notifications[notifID]; ok && !notif.Read {
				unreadList = append(unreadList, notif)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": unreadList,
		"user_id":      userID,
		"count":        len(unreadList),
	})
}

// getNotificationByID retourne une notification par son ID
func getNotificationByID(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	notif, exists := notifications[id]
	mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification non trouvée"})
		return
	}

	c.JSON(http.StatusOK, notif)
}

// createNotification crée une nouvelle notification quand un utilisateur reçoit un message
// Le compteur s'auto-incrémente automatiquement car la notification est créée avec Read=false
func createNotification(c *gin.Context) {
	var input struct {
		UserID    string `json:"user_id" binding:"required"`
		MessageID string `json:"message_id"`
		SenderID  string `json:"sender_id"`
		Content   string `json:"content" binding:"required"`
		ChatID    string `json:"chat_id"`
		Type      string `json:"type"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notifType := input.Type
	if notifType == "" {
		notifType = "message"
	}

	notif := Notification{
		ID:        uuid.New().String(),
		UserID:    input.UserID,
		MessageID: input.MessageID,
		SenderID:  input.SenderID,
		Content:   input.Content,
		ChatID:    input.ChatID,
		Type:      notifType,
		Read:      false, // Par défaut non lue, ce qui incrémente le compteur
		CreatedAt: time.Now(),
	}

	mu.Lock()
	notifications[notif.ID] = notif
	// Ajouter la notification à la liste de l'utilisateur
	userNotifications[input.UserID] = append(userNotifications[input.UserID], notif.ID)
	mu.Unlock()

	logger.Info("Notification créée pour l'utilisateur %s: %s", input.UserID, notif.ID)

	c.JSON(http.StatusCreated, notif)
}

// markAsRead marque une notification comme lue (décrémente le compteur)
func markAsRead(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	notif, exists := notifications[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification non trouvée"})
		return
	}

	notif.Read = true
	notifications[id] = notif

	c.JSON(http.StatusOK, notif)
}

// markAllAsRead marque toutes les notifications d'un utilisateur comme lues
func markAllAsRead(c *gin.Context) {
	userID := c.Param("userId")

	mu.Lock()
	defer mu.Unlock()

	count := 0
	if notificationIDs, exists := userNotifications[userID]; exists {
		for _, notifID := range notificationIDs {
			if notif, ok := notifications[notifID]; ok && !notif.Read {
				notif.Read = true
				notifications[notifID] = notif
				count++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Toutes les notifications ont été marquées comme lues",
		"user_id":      userID,
		"marked_count": count,
	})
}

// deleteNotification supprime une notification
func deleteNotification(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	notif, exists := notifications[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification non trouvée"})
		return
	}

	// Retirer la notification de la liste de l'utilisateur
	if notificationIDs, exists := userNotifications[notif.UserID]; exists {
		for i, notifID := range notificationIDs {
			if notifID == id {
				userNotifications[notif.UserID] = append(notificationIDs[:i], notificationIDs[i+1:]...)
				break
			}
		}
	}

	delete(notifications, id)
	c.JSON(http.StatusOK, gin.H{"message": "Notification supprimée"})
}
