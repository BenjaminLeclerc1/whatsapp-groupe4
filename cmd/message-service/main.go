package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/whatsapp-groupe4/internal/logger"
	"github.com/whatsapp-groupe4/internal/messages"
	"github.com/whatsapp-groupe4/internal/middleware"
)

func main() {
	logger.Init("message-service")
	defer logger.Close()

	port := getEnv("PORT", "8082")
	databaseURL := getEnv(
		"DATABASE_URL",
		"postgres://whatsapp:whatsapp_secret@localhost:5432/whatsapp_db?sslmode=disable",
	)

	runMigrations(databaseURL)

	pool, err := initDB(databaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()

	repo := messages.NewRepository(pool)
	service := messages.NewService(repo)
	handler := messages.NewHandler(service)

	rateLimiter := middleware.NewRateLimiter(60, time.Minute)
	defer rateLimiter.Stop()

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "connected"
		if err := pool.Ping(ctx); err != nil {
			dbStatus = "disconnected"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "message-service",
			"database": dbStatus,
		})
	})

	api := router.Group("/api/v1/messages")
	api.Use(middleware.ExtractUserID(), rateLimiter.Middleware())
	handler.RegisterRoutes(api)

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}

	startServer(server, port)
}

func startServer(server *http.Server, port string) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("Message Service started on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}

	log.Println("Message Service stopped")
}

func initDB(databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	log.Printf("PostgreSQL connected (min=%d max=%d)", config.MinConns, config.MaxConns)
	return pool, nil
}

func runMigrations(databaseURL string) {
	m, err := migrate.New(
		"file://migrations/message-service",
		databaseURL,
	)
	if err != nil {
		log.Fatalf("Migration init failed: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations applied successfully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
