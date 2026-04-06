package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // required for migrate postgres driver registration
	_ "github.com/golang-migrate/migrate/v4/source/file"       // required for migrate file source registration
	"github.com/jackc/pgx/v5/pgxpool"

	// --- Verify this matches your go.mod module name ---
	"github.com/whatsapp-groupe4/internal/chats"
	"github.com/whatsapp-groupe4/internal/middleware"
)

func main() {
	// 1. Init DB & Migrations
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	runMigrations(dbURL)

	pool, err := initDB(dbURL)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer pool.Close()

	// 2. Setup Chat Layers
	repo := chats.NewRepository(pool)
	svc := chats.NewService(repo)
	handler := chats.NewHandler(svc)

	// 3. Routes
	// r := gin.Default()

	// // FIX: Changed middleware.Auth() to middleware.ExtractUserID()
	// // to match your internal/middleware/auth.go file
	// api := r.Group("/api/v1/chats", middleware.ExtractUserID())
	// {
	// 	api.POST("/", handler.CreateChat)
	// 	api.GET("/my", handler.GetMyChats)
	// }

	// 3. Routes
  r := gin.Default()

    // 🛑 CRITICAL FOR DOCKER: Stop Gin from redirecting to internal hostnames
    r.RedirectTrailingSlash = false
    r.RedirectFixedPath = false

	r = newChatRouter(handler)
    r.Run(":8088")
}

func initDB(databaseURL string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}

func runMigrations(databaseURL string) {
	// Add ?x-migrations-table=migrations_chats to the URL
	// This prevents conflicts with other services in the same shard
	migrationURL := chatMigrationURL(databaseURL)

	m, err := migrate.New(
		"file://migrations/chat-service",
		migrationURL,
	)
	if err != nil {
		log.Fatalf("Could not create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Could not run up migrations: %v", err)
	}

	log.Println("Chat Service Migrations applied successfully!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func chatMigrationURL(databaseURL string) string {
	if strings.Contains(databaseURL, "?") {
		return databaseURL + "&x-migrations-table=migrations_chats"
	}
	return databaseURL + "?x-migrations-table=migrations_chats"
}

func newChatRouter(handler *chats.Handler) *gin.Engine {
	r := gin.Default()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	api := r.Group("/api/v1/chats")
	api.Use(middleware.ExtractUserID())
	api.GET("", handler.GetMyChats)
	api.GET("/", handler.GetMyChats)
	api.POST("", handler.CreateChat)
	api.POST("/", handler.CreateChat)
	api.PUT("/:id", handler.UpdateChat)
	api.DELETE("/:id", handler.DeleteChat)
	return r
}
