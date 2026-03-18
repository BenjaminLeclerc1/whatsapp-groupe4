package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/whatsapp-groupe4/internal/cache"
	"github.com/whatsapp-groupe4/internal/logger"
	"github.com/whatsapp-groupe4/internal/sharding"
	"github.com/whatsapp-groupe4/middleware/auth"
)

// User Model
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username" binding:"required"`
	Telephone string    `json:"telephone" binding:"required"`
	Email     string    `json:"email" binding:"required"`
	Password  string    `json:"password,omitempty"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type App struct {
	Shards *sharding.ShardManager
	Redis  *cache.RedisClient
}

func main() {
	logger.Init("user-service")
	defer logger.Close()

	shardURLsEnv := os.Getenv("SHARD_URLS")
	if shardURLsEnv == "" {
		logger.Fatal("SHARD_URLS env variable is required")
	}
	// Split by comma and trim whitespace
	shardURLs := strings.Split(shardURLsEnv, ",")

	// 1. Run Migrations on all shards
	runMigrations(shardURLs)

	// 2. Initialize Shard Manager
	shardMgr, err := sharding.NewShardManager(shardURLs)
	if err != nil {
		logger.Fatal("Failed to init shards: %v", err)
	}

	// 3. Initialize Redis
	rdb := cache.NewRedisClient(getEnv("REDIS_ADDR", "whatsapp-redis:6379"))

	app := &App{
		Shards: shardMgr,
		Redis:  rdb,
	}

	// 4. Setup Router
	router := gin.Default()

	api := router.Group("/api/v1/users")
	{
		api.POST("/register", app.register)
		api.POST("/login", app.login)
		api.POST("/logout", app.logout)
		api.GET("/search", app.searchUsers) // Moved to /search for clarity
		api.GET("/:id", app.getUserByID)
		api.PUT("/:id", app.updateUser)
		api.DELETE("/:id", app.deleteUser)
		api.GET("", app.getAllUsers)
	}

	port := getEnv("PORT", "8081")
	logger.Info("User Service started on port %s", port)
	router.Run(":" + port)
}

// --- CORE HANDLERS ---

func (app *App) register(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.Status = "active"
	user.Role = "user"

	shard := app.Shards.GetShard(user.ID)
	query := `INSERT INTO users (id, username, telephone, email, password, role, status) 
              VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := shard.Exec(c.Request.Context(), query,
		user.ID, user.Username, user.Telephone, user.Email, string(hashedPassword), user.Role, user.Status)

	if err != nil {
		logger.Error("DB Insert failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	// Async indexing to Search Service
	go func(u User) {
		searchURL := "http://search-service:8087/api/v1/search/index"
		payload := map[string]string{
			"id":      u.ID,
			"type":    "user",
			"content": u.Username,
		}
		jsonData, _ := json.Marshal(payload)
		http.Post(searchURL, "application/json", bytes.NewBuffer(jsonData))
	}(user)

	user.Password = ""
	c.JSON(http.StatusCreated, user)
}

func (app *App) login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Credentials required"})
		return
	}

	var user User
	found := false
	for _, shard := range app.Shards.Shards {
		err := shard.QueryRow(c.Request.Context(),
			"SELECT id, password, role FROM users WHERE email = $1", input.Email).
			Scan(&user.ID, &user.Password, &user.Role)
		if err == nil {
			found = true
			break
		}
	}

	if !found || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, _ := auth.GenerateToken(user.ID, user.Role)
	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": user.ID})
}

func (app *App) searchUsers(c *gin.Context) {
	queryParam := c.Query("q")
	if queryParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	var results []map[string]string
	// Fan-out: Query all shards
	for _, shard := range app.Shards.Shards {
		rows, err := shard.Query(c.Request.Context(),
			"SELECT id, username FROM users WHERE username ILIKE $1", "%"+queryParam+"%")
		if err != nil {
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var id, username string
			rows.Scan(&id, &username)
			results = append(results, map[string]string{"id": id, "username": username})
		}
	}
	c.JSON(http.StatusOK, results)
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
		"SELECT id, username, telephone, email, role, status FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Username, &user.Telephone, &user.Email, &user.Role, &user.Status)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	_ = app.Redis.SetSession(c.Request.Context(), id, user, 15*time.Minute)
	c.JSON(http.StatusOK, user)
}

func (app *App) updateUser(c *gin.Context) {
	id := c.Param("id")
	var input User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	shard := app.Shards.GetShard(id)
	query := "UPDATE users SET username=$1, telephone=$2, email=$3 WHERE id=$4::uuid"
	_, err := shard.Exec(c.Request.Context(), query, input.Username, input.Telephone, input.Email, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	app.Redis.Client.Del(c.Request.Context(), "session:"+id)
	c.JSON(http.StatusOK, gin.H{"message": "User profile updated"})
}

func (app *App) deleteUser(c *gin.Context) {
	id := c.Param("id")
	shard := app.Shards.GetShard(id)
	shard.Exec(c.Request.Context(), "DELETE FROM users WHERE id=$1::uuid", id)
	app.Redis.Client.Del(c.Request.Context(), "session:"+id)
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func (app *App) getAllUsers(c *gin.Context) {
	var allUsers []User
	for _, shard := range app.Shards.Shards {
		rows, err := shard.Query(c.Request.Context(), "SELECT id, username, email FROM users")
		if err != nil {
			continue
		}
		for rows.Next() {
			var u User
			rows.Scan(&u.ID, &u.Username, &u.Email)
			allUsers = append(allUsers, u)
		}
		rows.Close()
	}
	c.JSON(http.StatusOK, allUsers)
}

func (app *App) logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// --- HELPERS ---

func runMigrations(shardURLs []string) {
	for _, url := range shardURLs {
		targetURL := url
		if strings.Contains(url, "?") {
			targetURL += "&x-migrations-table=migrations_users"
		} else {
			targetURL += "?x-migrations-table=migrations_users"
		}

		m, err := migrate.New("file://migrations/user-service", targetURL)
		if err != nil {
			fmt.Printf("❌ Migration Init Error (%s): %v\n", url, err)
			continue
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			fmt.Printf("❌ Migration Run Error (%s): %v\n", url, err)
		} else {
			fmt.Printf("✅ Migration Success: %s\n", url)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}