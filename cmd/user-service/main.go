// package main

// import (
// 	"net/http"
// 	"os"
// 	"sync"

// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// 	"github.com/whatsapp-groupe4/internal/logger"
// )

// type User struct {
// 	ID       string `json:"id"`
// 	Username string `json:"username"`
// 	Email    string `json:"email"`
// 	Status   string `json:"status"`
// }

// var (
// 	users = make(map[string]User)
// 	mu    sync.RWMutex
// )

// func main() {
// 	logger.Init("user-service")
// 	defer logger.Close()

// 	router := gin.Default()

// 	port := getEnv("PORT", "8081")

// 	// Health check
// 	router.GET("/health", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, gin.H{
// 			"status":  "healthy",
// 			"service": "user-service",
// 		})
// 	})

// 	// Routes utilisateurs
// 	api := router.Group("/api/v1/users")
// 	{
// 		api.GET("", getAllUsers)
// 		api.GET("/:id", getUserByID)
// 		api.POST("", createUser)
// 		api.PUT("/:id", updateUser)
// 		api.DELETE("/:id", deleteUser)
// 	}

// 	logger.Info("User Service démarré sur le port %s", port)

// 	if err := router.Run(":" + port); err != nil {
// 		logger.Fatal("Erreur démarrage serveur: %v", err)
// 	}
// }

// func getEnv(key, defaultValue string) string {
// 	if value := os.Getenv(key); value != "" {
// 		return value
// 	}
// 	return defaultValue
// }

// func getAllUsers(c *gin.Context) {
// 	mu.RLock()
// 	defer mu.RUnlock()

// 	userList := make([]User, 0, len(users))
// 	for _, user := range users {
// 		userList = append(userList, user)
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"users": userList,
// 		"count": len(userList),
// 	})
// }

// func getUserByID(c *gin.Context) {
// 	id := c.Param("id")

// 	mu.RLock()
// 	user, exists := users[id]
// 	mu.RUnlock()

// 	if !exists {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, user)
// }

// func createUser(c *gin.Context) {
// 	var input struct {
// 		Username string `json:"username" binding:"required"`
// 		Email    string `json:"email" binding:"required"`
// 	}

// 	if err := c.ShouldBindJSON(&input); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	user := User{
// 		ID:       uuid.New().String(),
// 		Username: input.Username,
// 		Email:    input.Email,
// 		Status:   "active",
// 	}

// 	mu.Lock()
// 	users[user.ID] = user
// 	mu.Unlock()

// 	c.JSON(http.StatusCreated, user)
// }

// func updateUser(c *gin.Context) {
// 	id := c.Param("id")

// 	mu.Lock()
// 	defer mu.Unlock()

// 	user, exists := users[id]
// 	if !exists {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
// 		return
// 	}

// 	var input struct {
// 		Username string `json:"username"`
// 		Email    string `json:"email"`
// 		Status   string `json:"status"`
// 	}

// 	if err := c.ShouldBindJSON(&input); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if input.Username != "" {
// 		user.Username = input.Username
// 	}
// 	if input.Email != "" {
// 		user.Email = input.Email
// 	}
// 	if input.Status != "" {
// 		user.Status = input.Status
// 	}

// 	users[id] = user
// 	c.JSON(http.StatusOK, user)
// }

// func deleteUser(c *gin.Context) {
// 	id := c.Param("id")

// 	mu.Lock()
// 	defer mu.Unlock()

// 	if _, exists := users[id]; !exists {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
// 		return
// 	}

// 	delete(users, id)
// 	c.JSON(http.StatusOK, gin.H{"message": "Utilisateur supprimé"})
// }


package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/whatsapp-groupe4/internal/cache"
	"github.com/whatsapp-groupe4/internal/database"
	"github.com/whatsapp-groupe4/internal/logger"
)

// User struct represents the database and JSON model
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

// App holds our application dependencies (Jira: pgxpool & Redis)
type App struct {
	DB    *database.PostgresDB
	Redis *cache.RedisClient
}

func main() {
	// 1. Initialize Logger
	logger.Init("user-service")
	defer logger.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. Initialize pgxpool (Jira Ticket: Connection pooling)
	db := database.NewPostgresPool(ctx)
	defer db.Close()

	// 3. Initialize Redis (Jira Ticket: Setup Redis pour cache et sessions)
	redisAddr := getEnv("REDIS_ADDR", "redis:6379")
	rdb := cache.NewRedisClient(redisAddr)

	app := &App{
		DB:    db,
		Redis: rdb,
	}

	// 4. Setup Gin Router
	router := gin.Default()
	port := getEnv("PORT", "8081")

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "user-service"})
	})

	// User Routes
	api := router.Group("/api/v1/users")
	{
		api.GET("", app.getAllUsers)
		api.GET("/:id", app.getUserByID)
		api.POST("", app.createUser)
		api.PUT("/:id", app.updateUser)
		api.DELETE("/:id", app.deleteUser)
	}

	logger.Info("User Service démarré sur le port %s", port)

	if err := router.Run(":" + port); err != nil {
		logger.Fatal("Erreur démarrage serveur: %v", err)
	}
}

// --- HANDLERS ---

func (app *App) getAllUsers(c *gin.Context) {
	rows, err := app.DB.Pool.Query(c.Request.Context(), "SELECT id, username, email, status FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}
	defer rows.Close()

	var userList []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Status); err == nil {
			userList = append(userList, u)
		}
	}

	c.JSON(http.StatusOK, gin.H{"users": userList, "count": len(userList)})
}

func (app *App) getUserByID(c *gin.Context) {
	id := c.Param("id")
	var user User

	// 1. Try Cache First (Redis)
	err := app.Redis.GetSession(c.Request.Context(), id, &user)
	if err == nil {
		logger.Info("Cache Hit: User %s retrieved from Redis", id)
		c.JSON(http.StatusOK, user)
		return
	}

	// 2. Database Fallback (pgxpool)
	query := "SELECT id, username, email, status FROM users WHERE id = $1"
	err = app.DB.Pool.QueryRow(c.Request.Context(), query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Status)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	// 3. Set Cache for future requests (Expires in 10m)
	_ = app.Redis.SetSession(c.Request.Context(), id, user, 10*time.Minute)

	c.JSON(http.StatusOK, user)
}

func (app *App) createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.ID = uuid.New().String()
	user.Status = "active"

	query := "INSERT INTO users (id, username, email, status) VALUES ($1, $2, $3, $4)"
	_, err := app.DB.Pool.Exec(c.Request.Context(), query, user.ID, user.Username, user.Email, user.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur insertion DB"})
		return
	}

	// Pre-warm the cache
	_ = app.Redis.SetSession(c.Request.Context(), user.ID, user, 10*time.Minute)

	c.JSON(http.StatusCreated, user)
}

func (app *App) updateUser(c *gin.Context) {
	id := c.Param("id")
	var input User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := "UPDATE users SET username=$1, email=$2, status=$3 WHERE id=$4"
	_, err := app.DB.Pool.Exec(c.Request.Context(), query, input.Username, input.Email, input.Status, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur mise à jour DB"})
		return
	}

	// Invalidate Cache after update to prevent stale data
	app.Redis.Client.Del(c.Request.Context(), "session:"+id)

	c.JSON(http.StatusOK, gin.H{"message": "Utilisateur mis à jour"})
}

func (app *App) deleteUser(c *gin.Context) {
	id := c.Param("id")

	_, err := app.DB.Pool.Exec(c.Request.Context(), "DELETE FROM users WHERE id=$1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur suppression DB"})
		return
	}

	// Remove from Cache
	app.Redis.Client.Del(c.Request.Context(), "session:"+id)

	c.JSON(http.StatusOK, gin.H{"message": "Utilisateur supprimé"})
}

// --- HELPERS ---

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}