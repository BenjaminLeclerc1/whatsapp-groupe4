// package main

// import (
// 	"context"
// 	"log"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/jackc/pgx/v5/pgxpool"
// 	"github.com/whatsapp-groupe4/internal/logger"
// 	"github.com/whatsapp-groupe4/internal/messages"
// 	"github.com/whatsapp-groupe4/internal/middleware"

// 	"github.com/golang-migrate/migrate/v4"
//     _ "github.com/golang-migrate/migrate/v4/database/postgres"
//     _ "github.com/golang-migrate/migrate/v4/source/file"
// )

// func main() {
// // 1. Initialize Logger and Config
//     logger.Init("message-service")
//     defer logger.Close()
//     port := getEnv("PORT", "8082")

//     // --- FIX STARTS HERE ---
//     // Get the URL once at the top of main
//     databaseURL := getEnv("DATABASE_URL", "postgres://whatsapp:whatsapp_secret@localhost:5432/whatsapp_db?sslmode=disable")

//     // 2. Initialize Database (Pass the URL into initDB now)
//     pool, err := initDB(databaseURL) 
//     if err != nil {
//         log.Fatalf("database connection failed: %v", err)
//     }
//     defer pool.Close()

//     // NEW: Run the migrations (Now databaseURL is defined!)
//     runMigrations(databaseURL)
// 	// 3. Setup Router & Middleware
// 	router := gin.Default()
	
// 	repo := messages.NewRepository(pool)
// 	svc := messages.NewService(repo)
// 	handler := messages.NewHandler(svc)

// 	rateLimiter := middleware.NewRateLimiter(60, time.Minute)
// 	defer rateLimiter.Stop()

// 	// 4. Routes
// 	router.GET("/health", func(c *gin.Context) {
// 		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
// 		defer cancel()

// 		dbStatus := "connected"
// 		if err := pool.Ping(ctx); err != nil {
// 			dbStatus = "disconnected"
// 		}
// 		c.JSON(http.StatusOK, gin.H{
// 			"status":   "healthy",
// 			"service":  "message-service",
// 			"database": dbStatus,
// 		})
// 	})

// 	// api := router.Group("/api/v1", middleware.ExtractUserID(), rateLimiter.Middleware())
// 	// handler.RegisterRoutes(api)

// 	// In message-service/main.go
// api := router.Group("/api/v1/messages", middleware.ExtractUserID(), rateLimiter.Middleware())
// handler.RegisterRoutes(api)
// 	// 5. Graceful Shutdown Setup
// 	srv := &http.Server{
// 		Addr:           ":" + port,
// 		Handler:        router,
// 		ReadTimeout:    10 * time.Second,
// 		WriteTimeout:   15 * time.Second,
// 		IdleTimeout:    120 * time.Second,
// 		MaxHeaderBytes: 1 << 16,
// 	}

// 	// Channel to listen for interrupt signals
// 	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
// 	defer stop()

// 	go func() {
// 		logger.Info("Message Service started on port %s", port)
// 		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
// 			logger.Fatal("Server error: %v", err)
// 		}
// 	}()

// 	// Wait for signal
// 	<-ctx.Done()
// 	log.Println("Shutting down gracefully...")

// 	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	if err := srv.Shutdown(shutdownCtx); err != nil {
// 		log.Fatalf("Forced shutdown: %v", err)
// 	}

// 	log.Println("Message Service stopped")
// }

// func initDB(databaseURL string) (*pgxpool.Pool, error) {
//     // We no longer call getEnv here because we pass databaseURL as an argument
//     config, err := pgxpool.ParseConfig(databaseURL)
//     if err != nil {
//         return nil, err
//     }

//     // Connection Pool Settings
//     config.MaxConns = 50
//     config.MinConns = 10
//     config.MaxConnLifetime = time.Hour
//     config.MaxConnIdleTime = 30 * time.Minute
//     config.HealthCheckPeriod = 30 * time.Second

//     // Create a context for the connection attempt
//     ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//     defer cancel()

//     // Create the connection pool
//     pool, err := pgxpool.NewWithConfig(ctx, config)
//     if err != nil {
//         return nil, err
//     }

//     // Verify the connection is actually working
//     if err := pool.Ping(ctx); err != nil {
//         pool.Close()
//         return nil, err
//     }

//     log.Printf("PostgreSQL connected (pool: min=%d max=%d)", config.MinConns, config.MaxConns)
//     return pool, nil
// }

// func getEnv(key, defaultValue string) string {
// 	if value := os.Getenv(key); value != "" {
// 		return value
// 	}
// 	return defaultValue
// }


// func runMigrations(databaseURL string) {
//     // We point to the specific subfolder for this service!
//     m, err := migrate.New(
//         "file://migrations/message-service", 
//         databaseURL,
//     )
//     if err != nil {
//         log.Fatalf("Could not create migrate instance: %v", err)
//     }

//     if err := m.Up(); err != nil && err != migrate.ErrNoChange {
//         log.Fatalf("Could not run up migrations: %v", err)
//     }

//     log.Println("Migrations applied successfully!")
// }

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
	databaseURL := requireEnv("DATABASE_URL")

	// -----------------------------
	// Database
	// -----------------------------
	runMigrations(databaseURL)

	pool, err := initDB(databaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()

	// -----------------------------
	// Services
	// -----------------------------
	repo := messages.NewRepository(pool)
	service := messages.NewService(repo)
	handler := messages.NewHandler(service)

	rateLimiter := middleware.NewRateLimiter(60, time.Minute)
	defer rateLimiter.Stop()

	// -----------------------------
	// Router
	// -----------------------------
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

	// -----------------------------
	// API Routes
	// -----------------------------
	api := router.Group("/api/v1/messages")
	api.Use(middleware.ExtractUserID(), rateLimiter.Middleware())

	handler.RegisterRoutes(api)

	// -----------------------------
	// HTTP Server
	// -----------------------------
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

// --------------------------------------------------
// Server startup with graceful shutdown
// --------------------------------------------------

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

// --------------------------------------------------
// Database initialization
// --------------------------------------------------

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

// --------------------------------------------------
// Run database migrations
// --------------------------------------------------

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

// --------------------------------------------------
// Environment helper
// --------------------------------------------------

func getEnv(key, defaultValue string) string {

	if value := os.Getenv(key); value != "" {
		return value
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
