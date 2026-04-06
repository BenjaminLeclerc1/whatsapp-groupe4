package database

import (
	"testing"
)

func TestBuildPgxPoolConfig_InvalidURL(t *testing.T) {
	invalid := []string{
		"http://not-postgres",
		"postgres://user:pass@host:999999999/db",
	}
	for _, connStr := range invalid {
		if _, err := BuildPgxPoolConfig(connStr); err != nil {
			return
		}
	}
	t.Fatal("expected BuildPgxPoolConfig to fail for at least one invalid connection string")
}

func TestBuildPgxPoolConfig_ValidFormat(t *testing.T) {
	cfg, err := BuildPgxPoolConfig("postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxConns != 20 || cfg.MinConns != 5 {
		t.Fatalf("unexpected pool limits: max=%d min=%d", cfg.MaxConns, cfg.MinConns)
	}
}

func TestPostgresDB_Close_NilSafe(t *testing.T) {
	var db *PostgresDB
	db.Close()

	db = &PostgresDB{}
	db.Close()
}
