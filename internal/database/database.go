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

// BuildPgxPoolConfig parse DATABASE_URL et applique les réglages du pool (testable sans connexion réelle).
func BuildPgxPoolConfig(connStr string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	return config, nil
}

func NewPostgresPool(ctx context.Context) *PostgresDB {
	connStr := os.Getenv("DATABASE_URL")

	config, err := BuildPgxPoolConfig(connStr)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	fmt.Println("Successfully connected to PostgreSQL with pgxpool!")
	return &PostgresDB{Pool: pool}
}

func (db *PostgresDB) Close() {
	if db == nil || db.Pool == nil {
		return
	}
	db.Pool.Close()
}
