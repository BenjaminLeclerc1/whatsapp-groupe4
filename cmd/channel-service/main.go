package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/whatsapp-groupe4/internal/channels"
	"github.com/whatsapp-groupe4/internal/middleware"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// 1. Setup Config
	port := getEnv("PORT", "8085")
	databaseURL := requireEnv("DATABASE_URL")

	// 2. Initialize Database
	pool, err := initDB(databaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	// 3. Run Migrations
	runMigrations(databaseURL)

	// 4. Setup logic
	repo := channels.NewRepository(pool)
	svc := channels.NewService(repo)
	handler := channels.NewHandler(svc)

	router := newChannelRouter(handler)

	// 5. Start Server (This uses port, http, time, etc.)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Graceful shutdown (This uses os/signal, syscall, context)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Channel Service started on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down gracefully...")
}

func newChannelRouter(handler *channels.Handler) *gin.Engine {
	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "channel-service"})
	})
	api := router.Group("/api/v1", middleware.ExtractUserID())
	handler.RegisterRoutes(api)
	return router
}

func initDB(databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(context.Background(), config)
}

func channelMigrationURL(databaseURL string) string {
	if !strings.Contains(databaseURL, "?") {
		return databaseURL + "?x-migrations-table=migrations_channels"
	}
	return databaseURL + "&x-migrations-table=migrations_channels"
}

func runMigrations(databaseURL string) {
	targetURL := channelMigrationURL(databaseURL)

	m, err := migrate.New("file://migrations/channel-service", targetURL)
	if err != nil {
		log.Fatalf("migration init failed: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration up failed: %v", err)
	}
	log.Println("Channel migrations applied successfully!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return v
}
