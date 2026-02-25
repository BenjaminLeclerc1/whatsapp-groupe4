package database

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/whatsapp-groupe4/internal/logger"
)

var DB *pgxpool.Pool

// Connect établit la connexion à PostgreSQL
func Connect() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL non définie. Vérifiez votre fichier .env")
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
	logger.Info("Connexion à PostgreSQL établie")
	return nil
}

// Close ferme la connexion à la base de données
func Close() {
	if DB != nil {
		DB.Close()
		logger.Info("Connexion PostgreSQL fermée")
	}
}
