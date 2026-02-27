package main

import (
	"context"
	// "database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/whatsapp-groupe4/internal/channels"
	"github.com/whatsapp-groupe4/internal/middleware"
)

func main() {
	port := getEnv("PORT", "8085")

	pool, err := initDB()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	// --- NEW: RUN MIGRATIONS AUTOMATICALLY ---
	if err := runMigrations(pool); err != nil {
		log.Printf("Migration status: %v", err)
	}

	repo := channels.NewRepository(pool)
	svc := channels.NewService(repo)
	handler := channels.NewHandler(svc)

	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		dbStatus := "connected"
		if err := pool.Ping(ctx); err != nil {
			dbStatus = "disconnected"
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "channel-service", "database": dbStatus})
	})

	api := router.Group("/api/v1", middleware.ExtractUserID())
	handler.RegisterRoutes(api)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

// runMigrations handles the automatic table creation
func runMigrations(pool *pgxpool.Pool) error {
	// Convert pgxpool to standard *sql.DB for the migrate library
	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	// Make sure your .sql files are in a folder named "migrations"
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Println("Database migrations applied successfully")
	return nil
}

func initDB() (*pgxpool.Pool, error) {
	databaseURL := getEnv("DATABASE_URL", "postgres://whatsapp:whatsapp_secret@localhost:5432/whatsapp_db?sslmode=disable")
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 50
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}