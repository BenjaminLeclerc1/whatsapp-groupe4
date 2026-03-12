package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// PresenceStatus représente les différents états possibles
type PresenceStatus string

const (
	StatusOnline  PresenceStatus = "online"
	StatusOffline PresenceStatus = "offline"
	StatusTyping  PresenceStatus = "typing"
)

// Presence contient les informations de présence d'un utilisateur
type Presence struct {
	UserID       string         `json:"user_id"`
	Status       PresenceStatus `json:"status"`
	LastSeen     time.Time      `json:"last_seen"`
	LastActivity time.Time      `json:"last_activity"`
	ChatID       string         `json:"chat_id,omitempty"` // Pour le statut "typing"
}

var (
	presences = make(map[string]Presence)
	mu        sync.RWMutex
)

// Timeout pour passer automatiquement en offline (5 minutes)
const activityTimeout = 5 * time.Minute

// Timeout pour le statut typing (10 secondes)
const typingTimeout = 10 * time.Second

func main() {
	router := gin.Default()

	port := getEnv("PORT", "8083")

	// Démarrer le worker qui vérifie les timeouts
	go presenceTimeoutWorker()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "presence-service",
		})
	})

	// Routes présence
	api := router.Group("/api/v1/presence")
	{
		api.GET("/:userId", getUserPresence)
		api.GET("/bulk", getBulkPresence)
		api.POST("/update", updatePresence)
		api.POST("/typing", setTypingStatus)
		api.POST("/online", setOnlineStatus)
		api.POST("/offline", setOfflineStatus)
	}

	log.Printf("Presence Service démarré sur le port %s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Erreur démarrage serveur: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getUserPresence récupère la présence d'un utilisateur spécifique
func getUserPresence(c *gin.Context) {
	userID := c.Param("userId")

	mu.RLock()
	presence, exists := presences[userID]
	mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Présence non trouvée",
			"user_id": userID,
			"status": StatusOffline,
		})
		return
	}

	// Vérifier si l'utilisateur est toujours actif
	if presence.Status == StatusOnline && time.Since(presence.LastActivity) > activityTimeout {
		presence.Status = StatusOffline
		mu.Lock()
		presences[userID] = presence
		mu.Unlock()
	}

	c.JSON(http.StatusOK, presence)
}

// getBulkPresence récupère la présence de plusieurs utilisateurs
func getBulkPresence(c *gin.Context) {
	var input struct {
		UserIDs []string `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string]Presence)
	for _, userID := range input.UserIDs {
		if presence, exists := presences[userID]; exists {
			// Vérifier le timeout
			if presence.Status == StatusOnline && time.Since(presence.LastActivity) > activityTimeout {
				presence.Status = StatusOffline
			}
			result[userID] = presence
		} else {
			// Utilisateur pas encore connu, retourner offline
			result[userID] = Presence{
				UserID:   userID,
				Status:   StatusOffline,
				LastSeen: time.Now(),
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"presences": result,
		"count":     len(result),
	})
}

// updatePresence met à jour la présence d'un utilisateur (endpoint générique)
func updatePresence(c *gin.Context) {
	var input struct {
		UserID string         `json:"user_id" binding:"required"`
		Status PresenceStatus `json:"status" binding:"required"`
		ChatID string         `json:"chat_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Valider le statut
	if input.Status != StatusOnline && input.Status != StatusOffline && input.Status != StatusTyping {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Statut invalide. Valeurs acceptées: online, offline, typing",
		})
		return
	}

	now := time.Now()
	presence := Presence{
		UserID:       input.UserID,
		Status:       input.Status,
		LastActivity: now,
		LastSeen:     now,
		ChatID:       input.ChatID,
	}

	mu.Lock()
	presences[input.UserID] = presence
	mu.Unlock()

	c.JSON(http.StatusOK, presence)
}

// setOnlineStatus marque un utilisateur comme en ligne
func setOnlineStatus(c *gin.Context) {
	var input struct {
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	presence := Presence{
		UserID:       input.UserID,
		Status:       StatusOnline,
		LastActivity: now,
		LastSeen:     now,
	}

	mu.Lock()
	presences[input.UserID] = presence
	mu.Unlock()

	log.Printf("Utilisateur %s est maintenant online", input.UserID)

	c.JSON(http.StatusOK, presence)
}

// setOfflineStatus marque un utilisateur comme hors ligne
func setOfflineStatus(c *gin.Context) {
	var input struct {
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()

	mu.Lock()
	presence, exists := presences[input.UserID]
	if exists {
		presence.Status = StatusOffline
		presence.LastSeen = now
	} else {
		presence = Presence{
			UserID:   input.UserID,
			Status:   StatusOffline,
			LastSeen: now,
		}
	}
	presences[input.UserID] = presence
	mu.Unlock()

	log.Printf("Utilisateur %s est maintenant offline", input.UserID)

	c.JSON(http.StatusOK, presence)
}

// setTypingStatus indique qu'un utilisateur est en train de taper
func setTypingStatus(c *gin.Context) {
	var input struct {
		UserID string `json:"user_id" binding:"required"`
		ChatID string `json:"chat_id" binding:"required"`
		Typing bool   `json:"typing" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()

	mu.Lock()
	presence, exists := presences[input.UserID]
	if !exists {
		presence = Presence{
			UserID:       input.UserID,
			LastActivity: now,
			LastSeen:     now,
		}
	}

	if input.Typing {
		presence.Status = StatusTyping
		presence.ChatID = input.ChatID
	} else {
		presence.Status = StatusOnline
		presence.ChatID = ""
	}
	presence.LastActivity = now

	presences[input.UserID] = presence
	mu.Unlock()

	log.Printf("Utilisateur %s - typing: %v dans chat %s", input.UserID, input.Typing, input.ChatID)

	c.JSON(http.StatusOK, presence)
}

// presenceTimeoutWorker vérifie périodiquement les timeouts
func presenceTimeoutWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		mu.Lock()
		for userID, presence := range presences {
			updated := false

			// Timeout pour le statut typing
			if presence.Status == StatusTyping && time.Since(presence.LastActivity) > typingTimeout {
				presence.Status = StatusOnline
				presence.ChatID = ""
				updated = true
				log.Printf("Timeout typing pour utilisateur %s", userID)
			}

			// Timeout pour le statut online
			if presence.Status == StatusOnline && time.Since(presence.LastActivity) > activityTimeout {
				presence.Status = StatusOffline
				presence.LastSeen = now
				updated = true
				log.Printf("Timeout online pour utilisateur %s", userID)
			}

			if updated {
				presences[userID] = presence
			}
		}
		mu.Unlock()
	}
}
