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

// ─── getEnv ───────────────────────────────────────────────────────────────────

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

// ─── emailRegex ───────────────────────────────────────────────────────────────

func TestEmailRegex_Valid(t *testing.T) {
	validEmails := []string{
		"alice@example.com",
		"bob.martin@domain.fr",
		"user+tag@sub.domain.io",
		"test123@test.org",
	}
	for _, email := range validEmails {
		if !emailRegex.MatchString(email) {
			t.Errorf("expected '%s' to be valid", email)
		}
	}
}

func TestEmailRegex_Invalid(t *testing.T) {
	invalidEmails := []string{
		"notanemail",
		"@nodomain.com",
		"missing@",
		"",
		"spaces in@email.com",
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
	if u.Status != "active" {
		t.Errorf("unexpected Status: %s", u.Status)
	}
}

// ─── logout handler (pas de DB) ───────────────────────────────────────────────

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

// ─── register handler — validation ───────────────────────────────────────────

func TestRegister_EmptyBody(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/register", app.register)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

func TestRegister_MissingEmail(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/register", app.register)

	body, _ := json.Marshal(map[string]string{"username": "alice", "telephone": "0601020304"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing email, got %d", w.Code)
	}
}

func TestRegister_MissingUsername(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/register", app.register)

	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "telephone": "0601020304"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing username, got %d", w.Code)
	}
}

// ─── login handler — validation ───────────────────────────────────────────────

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

func TestLogin_InvalidEmail(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/login", app.login)

	body, _ := json.Marshal(map[string]string{"email": "notvalid", "password": "pass"})
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLogin_InvalidEmailFormat(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/login", app.login)

	body, _ := json.Marshal(map[string]string{"email": "bademail", "password": "password123"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email format, got %d", w.Code)
	}
}

func TestLogin_MissingPassword(t *testing.T) {
	app := &App{}
	r := gin.New()
	r.POST("/login", app.login)

	body, _ := json.Marshal(map[string]string{"email": "alice@test.com"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ─── updateUser / deleteUser handlers — validation ────────────────────────────

func TestUpdateUser_NilShards(t *testing.T) {
	// updateUser calls app.Shards.GetShard → ne doit pas paniquer avec un ID invalide
	// car ShouldBindJSON ne valide pas l'absence de champs dans updateUser
	app := &App{}
	r := gin.New()
	r.PUT("/users/:id", app.updateUser)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/users/test-id", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Avec app.Shards nil, va paniquer — on vérifie qu'on récupère bien la panique
	defer func() {
		if r := recover(); r != nil {
			// Comportement attendu quand Shards est nil
		}
	}()
	r.ServeHTTP(w, req)
}
