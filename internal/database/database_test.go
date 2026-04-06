package database

import (
	"testing"
)

func TestBuildPgxPoolConfig_InvalidURL(t *testing.T) {
	_, err := BuildPgxPoolConfig("")
	if err == nil {
		t.Fatal("expected parse error for empty DATABASE_URL")
	}
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
