package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

var (
	users = make(map[string]User)
	mu    sync.RWMutex
)

func main() {
	router := gin.Default()

	port := getEnv("PORT", "8081")

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "user-service",
		})
	})

	// Routes utilisateurs
	api := router.Group("/api/v1/users")
	{
		api.GET("", getAllUsers)
		api.GET("/:id", getUserByID)
		api.POST("", createUser)
		api.PUT("/:id", updateUser)
		api.DELETE("/:id", deleteUser)
	}

	log.Printf("User Service démarré sur le port %s", port)

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

func getAllUsers(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

	userList := make([]User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"users": userList,
		"count": len(userList),
	})
}

func getUserByID(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	user, exists := users[id]
	mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func createUser(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := User{
		ID:       uuid.New().String(),
		Username: input.Username,
		Email:    input.Email,
		Status:   "active",
	}

	mu.Lock()
	users[user.ID] = user
	mu.Unlock()

	c.JSON(http.StatusCreated, user)
}

func updateUser(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	user, exists := users[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Status   string `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	if input.Status != "" {
		user.Status = input.Status
	}

	users[id] = user
	c.JSON(http.StatusOK, user)
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := users[id]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	delete(users, id)
	c.JSON(http.StatusOK, gin.H{"message": "Utilisateur supprimé"})
}
