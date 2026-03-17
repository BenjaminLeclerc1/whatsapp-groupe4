package main

import (
	// "context"
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

	"github.com/whatsapp-groupe4/middleware/auth" // Ensure this package exists with GenerateToken
	"github.com/whatsapp-groupe4/internal/cache"
	"github.com/whatsapp-groupe4/internal/logger"
	"github.com/whatsapp-groupe4/internal/sharding"
)

// User Model matches the new migration schema
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
	shardURLs := strings.Split(shardURLsEnv, ",")

	runMigrations(shardURLs)

	shardMgr, err := sharding.NewShardManager(shardURLs)
	if err != nil {
		logger.Fatal("Failed to init shards: %v", err)
	}

	rdb := cache.NewRedisClient(getEnv("REDIS_ADDR", "redis:6379"))

	app := &App{
		Shards: shardMgr,
		Redis:  rdb,
	}

	router := gin.Default()

	// API Routes
	api := router.Group("/api/v1/users")
	{
		api.POST("/register", app.register)
		api.POST("/login", app.login)
		api.POST("/logout", app.logout)
		api.GET("/:id", app.getUserByID)
		api.PUT("/:id", app.updateUser)
		api.DELETE("/:id", app.deleteUser)
		api.GET("", app.getAllUsers)
	}

	port := getEnv("PORT", "8081")
	logger.Info("User Service started on port %s", port)
	router.Run(":" + port)
}


func runMigrations(shardURLs []string) {
    for _, url := range shardURLs {
        // We append a custom migration table name so this service 
        // doesn't conflict with others sharing the same shard.
        targetURL := url
        if strings.Contains(url, "?") {
            targetURL += "&x-migrations-table=migrations_users"
        } else {
            targetURL += "?x-migrations-table=migrations_users"
        }

        m, err := migrate.New("file://migrations/user-service", targetURL) 
        if err != nil {
            fmt.Printf("❌ Erreur init migration pour %s: %v\n", url, err)
            os.Exit(1)
        }

        if err := m.Up(); err != nil && err != migrate.ErrNoChange {
            fmt.Printf("❌ Erreur exécution migration pour %s: %v\n", url, err)
            os.Exit(1)
        }
        fmt.Printf("✅ Migration réussie pour : %s\n", url)
    }
}
// --- AUTH HANDLERS ---

func (app *App) register(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Security error"})
		return
	}

	// 2. Setup Metadata
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.Status = "active"
	user.Role = "user"

	// 3. Save to Shard
	shard := app.Shards.GetShard(user.ID)
	query := `INSERT INTO users (id, username, telephone, email, password, role, status) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err = shard.Exec(c.Request.Context(), query,
		user.ID, user.Username, user.Telephone, user.Email, string(hashedPassword), user.Role, user.Status)

	if err != nil {
		// THIS LINE IS NEW: It will show the real error in your terminal/docker logs
		logger.Error("CRITICAL: Database Insert failed: %v", err)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "User registration failed", 
			"details": err.Error(), // Temporarily send the real error to curl for debugging
		})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing email or password"})
		return
	}

	var user User
	found := false

	// Search all shards to find the user by email
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

	// Generate JWT Token
	token, err := auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"user_id": user.ID,
		"message": "Login successful",
	})
}

// Assure-toi que le nom correspond exactement (L minuscule si c'est interne)
func (app *App) logout(c *gin.Context) {
    // Ton code de déconnexion ici
    c.JSON(200, gin.H{"message": "Déconnecté avec succès"})
}
// --- USER MANAGEMENT HANDLERS ---

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

func (app *App) getAllUsers(c *gin.Context) {
	var allUsers []User
	for _, pool := range app.Shards.Shards {
		rows, err := pool.Query(c.Request.Context(), "SELECT id, username, telephone, email, role, status FROM users")
		if err != nil {
			continue
		}
		for rows.Next() {
			var u User
			rows.Scan(&u.ID, &u.Username, &u.Telephone, &u.Email, &u.Role, &u.Status)
			allUsers = append(allUsers, u)
		}
		rows.Close()
	}
	c.JSON(http.StatusOK, allUsers)
}




func (app *App) updateUser(c *gin.Context) {
    id := c.Param("id")
    var input User
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides", "details": err.Error()})
        return
    }

    shard := app.Shards.GetShard(id)
    
    // 1. On force Postgres à lire l'ID comme un UUID ($4::uuid)
    query := "UPDATE users SET username=$1, telephone=$2, email=$3 WHERE id=$4::uuid"
    
    // 2. Exécution avec log d'erreur
    _, err := shard.Exec(c.Request.Context(), query, 
        input.Username, 
        input.Telephone, 
        input.Email, 
        id,
    )

    if err != nil {
        fmt.Printf("❌ Erreur SQL Update: %v\n", err) 
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Update failed",
            "details": err.Error(),
        })
        return
    }

    app.Redis.Client.Del(c.Request.Context(), "session:"+id)
    // CORRECTION ICI : Le message est sur une seule ligne
    c.JSON(http.StatusOK, gin.H{"message": "User profile updated"})
}



func (app *App) deleteUser(c *gin.Context) {
	id := c.Param("id")
	shard := app.Shards.GetShard(id)
	shard.Exec(c.Request.Context(), "DELETE FROM users WHERE id=$1", id)

	app.Redis.Client.Del(c.Request.Context(), "session:"+id)
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}