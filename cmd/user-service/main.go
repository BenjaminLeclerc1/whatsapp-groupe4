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
	// "context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/whatsapp-groupe4/internal/cache"
	"github.com/whatsapp-groupe4/internal/logger"
	"github.com/whatsapp-groupe4/internal/sharding"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

type App struct {
	Shards *sharding.ShardManager
	Redis  *cache.RedisClient
}

func main() {
	logger.Init("user-service")
	defer logger.Close()

	// 1. Initialize Shards
	shardURLs := strings.Split(os.Getenv("SHARD_URLS"), ",")
	if len(shardURLs) == 0 || shardURLs[0] == "" {
		logger.Fatal("SHARD_URLS env variable is required")
	}

	shardMgr, err := sharding.NewShardManager(shardURLs)
	if err != nil {
		logger.Fatal("Failed to init shards: %v", err)
	}

	// 2. Initialize Redis
	rdb := cache.NewRedisClient(getEnv("REDIS_ADDR", "redis:6379"))

	app := &App{
		Shards: shardMgr,
		Redis:  rdb,
	}

	router := gin.Default()

	// API Routes
	api := router.Group("/api/v1/users")
	{
		api.POST("", app.createUser)
		api.GET("/:id", app.getUserByID)
		api.PUT("/:id", app.updateUser)
		api.DELETE("/:id", app.deleteUser)
		api.GET("", app.getAllUsers)
	}

	port := getEnv("PORT", "8081")
	logger.Info("User Service started on port %s", port)
	router.Run(":" + port)
}

// --- HANDLERS ---

func (app *App) createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.ID = uuid.New().String()
	user.Status = "active"

	shard := app.Shards.GetShard(user.ID)
	_, err := shard.Exec(c.Request.Context(), 
		"INSERT INTO users (id, username, email, status) VALUES ($1, $2, $3, $4)", 
		user.ID, user.Username, user.Email, user.Status)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
		return
	}

	_ = app.Redis.SetSession(c.Request.Context(), user.ID, user, 15*time.Minute)
	c.JSON(http.StatusCreated, user)
}

func (app *App) getUserByID(c *gin.Context) {
	id := c.Param("id")
	var user User

	if err := app.Redis.GetSession(c.Request.Context(), id, &user); err == nil {
		c.JSON(http.StatusOK, user)
		return
	}

	shard := app.Shards.GetShard(id)
	err := shard.QueryRow(c.Request.Context(), 
		"SELECT id, username, email, status FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Username, &user.Email, &user.Status)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	_ = app.Redis.SetSession(c.Request.Context(), id, user, 15*time.Minute)
	c.JSON(http.StatusOK, user)
}

func (app *App) getAllUsers(c *gin.Context) {
	var allUsers []User
	// Basic loop: queries each shard one by one
	for _, pool := range app.Shards.Shards {
		rows, _ := pool.Query(c.Request.Context(), "SELECT id, username, email, status FROM users")
		for rows.Next() {
			var u User
			rows.Scan(&u.ID, &u.Username, &u.Email, &u.Status)
			allUsers = append(allUsers, u)
		}
		rows.Close()
	}
	c.JSON(http.StatusOK, allUsers)
}

func (app *App) updateUser(c *gin.Context) {
	id := c.Param("id")
	var input User
	c.ShouldBindJSON(&input)

	shard := app.Shards.GetShard(id)
	shard.Exec(c.Request.Context(), "UPDATE users SET username=$1 WHERE id=$2", input.Username, id)
	
	app.Redis.Client.Del(c.Request.Context(), "session:"+id)
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (app *App) deleteUser(c *gin.Context) {
	id := c.Param("id")
	shard := app.Shards.GetShard(id)
	shard.Exec(c.Request.Context(), "DELETE FROM users WHERE id=$1", id)
	
	app.Redis.Client.Del(c.Request.Context(), "session:"+id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" { return v }
	return defaultValue
}