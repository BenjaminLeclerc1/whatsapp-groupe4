package main

import (
	"context"
	"log"
	"os"
	"time"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// --- Verify this matches your go.mod module name ---
	"github.com/whatsapp-groupe4/internal/chats"
	"github.com/whatsapp-groupe4/internal/middleware"
)

func main() {
	// 1. Init DB & Migrations
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://whatsapp:whatsapp_secret@localhost:5432/chat_db?sslmode=disable"
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

    api := r.Group("/api/v1/chats")
    {
        api.Use(middleware.ExtractUserID())

        // ✅ Handle both so Gin never feels the need to redirect
        api.GET("", handler.GetMyChats)  // matches /api/v1/chats
        api.GET("/", handler.GetMyChats) // matches /api/v1/chats/
        
        api.POST("", handler.CreateChat)
        api.POST("/", handler.CreateChat)
    }
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
    migrationURL := databaseURL
    if !strings.Contains(migrationURL, "?") {
        migrationURL += "?x-migrations-table=migrations_chats"
    } else {
        migrationURL += "&x-migrations-table=migrations_chats"
    }

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