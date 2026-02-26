package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/whatsapp-groupe4/internal/cache"
	"github.com/whatsapp-groupe4/internal/logger"
	"github.com/whatsapp-groupe4/internal/sharding"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

// runMigrations is now correctly placed outside of main
func runMigrations(shardURLs []string) {
	for _, url := range shardURLs {
		// golang-migrate needs the 'postgres://' prefix
		m, err := migrate.New("file://migrations", url)
		if err != nil {
			logger.Fatal("Could not init migration for %s: %v", url, err)
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			logger.Fatal("Could not run migration for %s: %v", url, err)
		}
		logger.Info("Migrations successful for shard: %s", url)
	}
}

func main() {
	logger.Init("user-service")
	defer logger.Close()

	// 1. Get Shard URLs
	shardURLsEnv := os.Getenv("SHARD_URLS")
	if shardURLsEnv == "" {
		logger.Fatal("SHARD_URLS env variable is required")
	}
	shardURLs := strings.Split(shardURLsEnv, ",")

	// 2. Run Migrations on all shards BEFORE starting service
	runMigrations(shardURLs)

	// 3. Initialize Shard Manager
	shardMgr, err := sharding.NewShardManager(shardURLs)
	if err != nil {
		logger.Fatal("Failed to init shards: %v", err)
	}

	// 4. Initialize Redis
	rdb := cache.NewRedisClient(getEnv("REDIS_ADDR", "redis:6379"))

	app := &App{
		Shards: shardMgr,
		Redis:  rdb,
	}

	router := gin.Default()

	// 5. API Routes
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
	for _, pool := range app.Shards.Shards {
		rows, err := pool.Query(c.Request.Context(), "SELECT id, username, email, status FROM users")
		if err != nil {
			continue
		}
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}