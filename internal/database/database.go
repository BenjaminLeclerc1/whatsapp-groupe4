// package database

// import (
// 	"context"
// 	"os"
// 	"time"

// 	"github.com/jackc/pgx/v5/pgxpool"
// 	"github.com/whatsapp-groupe4/internal/logger"
// )

// var DB *pgxpool.Pool

// // Connect établit la connexion à PostgreSQL
// func Connect() error {
// 	databaseURL := os.Getenv("DATABASE_URL")
// 	if databaseURL == "" {
// 		log.Fatal("DATABASE_URL non définie. Vérifiez votre fichier .env")
// 	}

// 	config, err := pgxpool.ParseConfig(databaseURL)
// 	if err != nil {
// 		return err
// 	}

// 	// Configuration du pool de connexions
// 	config.MaxConns = 10
// 	config.MinConns = 2
// 	config.MaxConnLifetime = time.Hour
// 	config.MaxConnIdleTime = 30 * time.Minute

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	pool, err := pgxpool.NewWithConfig(ctx, config)
// 	if err != nil {
// 		return err
// 	}

// 	// Test de la connexion
// 	if err := pool.Ping(ctx); err != nil {
// 		return err
// 	}

// 	DB = pool
// 	logger.Info("Connexion à PostgreSQL établie")
// 	return nil
// }

// // Close ferme la connexion à la base de données
// func Close() {
// 	if DB != nil {
// 		DB.Close()
// 		logger.Info("Connexion PostgreSQL fermée")
// 	}
// }

package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
}

func NewPostgresPool(ctx context.Context) *PostgresDB {
	connStr := os.Getenv("DATABASE_URL")
	
	// 1. Setup Configuration
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v", err)
	}

	// 2. Configure Pool Settings (Standard for Production)
	config.MaxConns = 20                      // Max active connections
	config.MinConns = 5                       // Minimum idle connections
	config.MaxConnLifetime = time.Hour        // Refresh connections every hour
	config.MaxConnIdleTime = 30 * time.Minute // Close idle connections

	// 3. Create the Pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	// 4. Ping to verify connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	fmt.Println("Successfully connected to PostgreSQL with pgxpool!")
	return &PostgresDB{Pool: pool}
}

func (db *PostgresDB) Close() {
	db.Pool.Close()
}