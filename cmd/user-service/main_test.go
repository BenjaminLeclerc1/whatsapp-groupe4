package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("USER_TEST_NONEXISTENT", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("USER_TEST_VAR", "value")
	defer os.Unsetenv("USER_TEST_VAR")

	result := getEnv("USER_TEST_VAR", "default")
	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}
}

func TestGetEnv_EmptyFallback(t *testing.T) {
	os.Setenv("USER_EMPTY_VAR", "")
	defer os.Unsetenv("USER_EMPTY_VAR")

	result := getEnv("USER_EMPTY_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got '%s'", result)
	}
}

// ─── Email regex ──────────────────────────────────────────────────────────────

func TestEmailRegex_Valid(t *testing.T) {
	validEmails := []string{
		"alice@example.com",
		"bob.martin@domain.fr",
		"user+tag@sub.domain.io",
	}
	for _, email := range validEmails {
		if !emailRegex.MatchString(email) {
			t.Errorf("expected '%s' to be a valid email", email)
		}
	}
}

func TestEmailRegex_Invalid(t *testing.T) {
	invalidEmails := []string{
		"notanemail",
		"@nodomain.com",
		"missing@",
		"",
	}
	for _, email := range invalidEmails {
		if emailRegex.MatchString(email) {
			t.Errorf("expected '%s' to be invalid", email)
		}
	}
}

// ─── User struct ──────────────────────────────────────────────────────────────

func TestUserStruct(t *testing.T) {
	u := User{
		ID:        "user-1",
		Username:  "alice",
		Telephone: "0601020304",
		Email:     "alice@example.com",
		Role:      "user",
		Status:    "active",
	}
	if u.ID != "user-1" {
		t.Errorf("unexpected ID: %s", u.ID)
	}
	if u.Role != "user" {
		t.Errorf("unexpected Role: %s", u.Role)
	}
}

// ─── Logout handler ──────────────────────────────────────────────────────────

func TestLogout(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/logout", app.logout)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] != "Logged out successfully" {
		t.Errorf("unexpected message: %s", resp["message"])
	}
}

// ─── Register handler — validation ───────────────────────────────────────────

func TestRegister_MissingFields(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/register", app.register)

	// Corps vide → 400
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

// ─── Login handler — validation email ────────────────────────────────────────

func TestLogin_InvalidEmail(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/login", app.login)

	body := map[string]string{"email": "notvalid", "password": "pass"}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email, got %d", w.Code)
	}
}

func TestLogin_MissingCredentials(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/login", app.login)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
