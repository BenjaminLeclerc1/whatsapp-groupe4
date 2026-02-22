package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

// Connect établit la connexion à PostgreSQL
func Connect() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://whatsapp:whatsapp_secret@localhost:5432/whatsapp_db?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return err
	}

	// Configuration du pool de connexions
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}

	// Test de la connexion
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	DB = pool
	log.Println("Connexion à PostgreSQL établie")
	return nil
}

// Close ferme la connexion à la base de données
func Close() {
	if DB != nil {
		DB.Close()
		log.Println("Connexion PostgreSQL fermée")
	}
}
