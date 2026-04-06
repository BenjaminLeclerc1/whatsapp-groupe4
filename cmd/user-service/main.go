package main

import (
	// "bytes"
	// "encoding/json"
	// "fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
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

// --- MODELS & STRUCTS ---

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
	Shards    *sharding.ShardManager
	Redis     *cache.RedisClient
	JWTSecret string
}

var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

// --- MAIN ---

func main() {
	logger.Init("user-service")
	defer logger.Close()

	shardURLsEnv := os.Getenv("SHARD_URLS")
	if shardURLsEnv == "" {
		logger.Fatal("SHARD_URLS env variable is required")
	}
	shardURLs := strings.Split(shardURLsEnv, ",")

	runMigrations(shardURLs)

	shardMgr, err := sharding.NewShardManager(shardURLs)
	if err != nil {
		logger.Fatal("Failed to init shards: %v", err)
	}

	jwtSecret := requireEnv("JWT_SECRET")
	rdb := cache.NewRedisClient(getEnv("REDIS_ADDR", "redis:6379"))

	app := &App{
		Shards:    shardMgr,
		Redis:     rdb,
		JWTSecret: jwtSecret,
	}

	router := newUserRouter(app)

	port := getEnv("PORT", "8081")
	logger.Info("User Service started on port %s", port)
	router.Run(":" + port)
}

// --- HANDLERS ---

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
	query := `INSERT INTO users (id, username, telephone, email, password, role, status) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := shard.Exec(c.Request.Context(), query, user.ID, user.Username, user.Telephone, user.Email, string(hashedPassword), user.Role, user.Status)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}
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
	if !emailRegex.MatchString(input.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format d'email invalide"})
		return
	}
	var user User
	found := false
	for _, shard := range app.Shards.Shards {
		err := shard.QueryRow(c.Request.Context(), "SELECT id, password, role FROM users WHERE email = $1", input.Email).Scan(&user.ID, &user.Password, &user.Role)
		if err == nil {
			found = true
			break
		}
	}
	if !found || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	token, err := auth.GenerateToken(user.ID, user.Role, app.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": user.ID})
}

func (app *App) logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (app *App) searchUsers(c *gin.Context) {
	queryParam := c.Query("q")
	var results []map[string]string
	for _, shard := range app.Shards.Shards {
		rows, err := shard.Query(c.Request.Context(), "SELECT id, username FROM users WHERE username ILIKE $1", "%"+queryParam+"%")
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
	shard := app.Shards.GetShard(id)
	err := shard.QueryRow(c.Request.Context(), "SELECT id, username, telephone, email, role, status FROM users WHERE id = $1", id).Scan(&user.ID, &user.Username, &user.Telephone, &user.Email, &user.Role, &user.Status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (app *App) updateUser(c *gin.Context) {
	id := c.Param("id")
	var input User
	c.ShouldBindJSON(&input)
	shard := app.Shards.GetShard(id)
	shard.Exec(c.Request.Context(), "UPDATE users SET username=$1, telephone=$2, email=$3 WHERE id=$4", input.Username, input.Telephone, input.Email, id)
	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

func (app *App) deleteUser(c *gin.Context) {
	id := c.Param("id")
	shard := app.Shards.GetShard(id)
	shard.Exec(c.Request.Context(), "DELETE FROM users WHERE id=$1", id)
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func (app *App) getAllUsers(c *gin.Context) {
	var allUsers []User
	for _, shard := range app.Shards.Shards {
		rows, _ := shard.Query(c.Request.Context(), "SELECT id, username, email FROM users")
		for rows.Next() {
			var u User
			rows.Scan(&u.ID, &u.Username, &u.Email)
			allUsers = append(allUsers, u)
		}
		rows.Close()
	}
	c.JSON(http.StatusOK, allUsers)
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
			continue
		}
		m.Up()
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		logger.Fatal("%s environment variable is required", key)
	}
	return v
}

func newUserRouter(app *App) *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.OPTIONS("/*path", func(c *gin.Context) { c.AbortWithStatus(http.StatusOK) })

	api := router.Group("/api/v1/users")
	api.POST("/register", app.register)
	api.POST("/login", app.login)
	api.POST("/logout", app.logout)
	api.GET("/search", app.searchUsers)
	api.GET("/:id", app.getUserByID)
	api.PUT("/:id", app.updateUser)
	api.DELETE("/:id", app.deleteUser)
	api.GET("", app.getAllUsers)
	return router
}
