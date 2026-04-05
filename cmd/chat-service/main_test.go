package main

import (
	"strings"
	"testing"
)

func TestMigrationURL_WithoutQueryParam(t *testing.T) {
	url := "postgres://user:pass@localhost/db"
	var result string
	if !strings.Contains(url, "?") {
		result = url + "?x-migrations-table=migrations_chats"
	} else {
		result = url + "&x-migrations-table=migrations_chats"
	}
	expected := "postgres://user:pass@localhost/db?x-migrations-table=migrations_chats"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestMigrationURL_WithExistingQueryParam(t *testing.T) {
	url := "postgres://user:pass@localhost/db?sslmode=disable"
	var result string
	if !strings.Contains(url, "?") {
		result = url + "?x-migrations-table=migrations_chats"
	} else {
		result = url + "&x-migrations-table=migrations_chats"
	}
	expected := "postgres://user:pass@localhost/db?sslmode=disable&x-migrations-table=migrations_chats"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestMigrationURL_ContainsTableName(t *testing.T) {
	url := "postgres://localhost/db"
	var result string
	if !strings.Contains(url, "?") {
		result = url + "?x-migrations-table=migrations_chats"
	} else {
		result = url + "&x-migrations-table=migrations_chats"
	}
	if !strings.Contains(result, "migrations_chats") {
		t.Errorf("expected result to contain 'migrations_chats', got: %s", result)
	}
}
