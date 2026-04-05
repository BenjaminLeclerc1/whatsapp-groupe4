package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupPresenceRouter() *gin.Engine {
	r := gin.New()
	api := r.Group("/api/v1/presence")
	{
		api.GET("/:userId", getUserPresence)
		api.GET("/bulk", getBulkPresence)
		api.POST("/update", updatePresence)
		api.POST("/typing", setTypingStatus)
		api.POST("/online", setOnlineStatus)
		api.POST("/offline", setOfflineStatus)
	}
	return r
}

func clearPresences() {
	mu.Lock()
	presences = make(map[string]Presence)
	mu.Unlock()
}

// ─── Constantes ──────────────────────────────────────────────────────────────

func TestPresenceStatus(t *testing.T) {
	if StatusOnline != "online" {
		t.Errorf("expected 'online', got '%s'", StatusOnline)
	}
	if StatusOffline != "offline" {
		t.Errorf("expected 'offline', got '%s'", StatusOffline)
	}
	if StatusTyping != "typing" {
		t.Errorf("expected 'typing', got '%s'", StatusTyping)
	}
}

func TestPresenceCreation(t *testing.T) {
	now := time.Now()
	p := Presence{
		UserID:       "user-123",
		Status:       StatusOnline,
		LastActivity: now,
		LastSeen:     now,
	}
	if p.UserID != "user-123" {
		t.Errorf("expected 'user-123', got '%s'", p.UserID)
	}
	if p.Status != StatusOnline {
		t.Errorf("expected online, got '%s'", p.Status)
	}
}

func TestTimeouts(t *testing.T) {
	if activityTimeout != 5*time.Minute {
		t.Errorf("activityTimeout expected 5m, got %v", activityTimeout)
	}
	if typingTimeout != 10*time.Second {
		t.Errorf("typingTimeout expected 10s, got %v", typingTimeout)
	}
}

func TestGetEnv(t *testing.T) {
	result := getEnv("PRESENCE_NONEXISTENT", "default_val")
	if result != "default_val" {
		t.Errorf("expected 'default_val', got '%s'", result)
	}

	os.Setenv("PRESENCE_TEST_VAR", "set_val")
	defer os.Unsetenv("PRESENCE_TEST_VAR")
	result = getEnv("PRESENCE_TEST_VAR", "default_val")
	if result != "set_val" {
		t.Errorf("expected 'set_val', got '%s'", result)
	}
}

// ─── getUserPresence ──────────────────────────────────────────────────────────

func TestGetUserPresence_NotFound(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/presence/unknown-user", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetUserPresence_Found(t *testing.T) {
	clearPresences()
	now := time.Now()
	mu.Lock()
	presences["user-1"] = Presence{
		UserID:       "user-1",
		Status:       StatusOnline,
		LastActivity: now,
		LastSeen:     now,
	}
	mu.Unlock()

	r := setupPresenceRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/presence/user-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp Presence
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", resp.UserID)
	}
}

func TestGetUserPresence_AutoOffline(t *testing.T) {
	clearPresences()
	// Simuler un utilisateur online depuis 10 minutes → doit passer offline
	mu.Lock()
	presences["user-timeout"] = Presence{
		UserID:       "user-timeout",
		Status:       StatusOnline,
		LastActivity: time.Now().Add(-10 * time.Minute),
		LastSeen:     time.Now().Add(-10 * time.Minute),
	}
	mu.Unlock()

	r := setupPresenceRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/presence/user-timeout", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp Presence
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != StatusOffline {
		t.Errorf("expected offline after timeout, got %s", resp.Status)
	}
}

// ─── getBulkPresence ─────────────────────────────────────────────────────────

func TestGetBulkPresence_MissingBody(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/presence/bulk", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetBulkPresence_UnknownUsers(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	body, _ := json.Marshal(map[string][]string{"user_ids": {"u1", "u2"}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/presence/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) != 2 {
		t.Errorf("expected count 2, got %v", resp["count"])
	}
}

func TestGetBulkPresence_KnownUsers(t *testing.T) {
	clearPresences()
	now := time.Now()
	mu.Lock()
	presences["known-1"] = Presence{UserID: "known-1", Status: StatusOnline, LastActivity: now, LastSeen: now}
	mu.Unlock()

	r := setupPresenceRouter()
	body, _ := json.Marshal(map[string][]string{"user_ids": {"known-1", "unknown-2"}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/presence/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─── updatePresence ───────────────────────────────────────────────────────────

func TestUpdatePresence_MissingFields(t *testing.T) {
	r := setupPresenceRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/update", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdatePresence_InvalidStatus(t *testing.T) {
	r := setupPresenceRouter()

	body, _ := json.Marshal(map[string]string{"user_id": "u1", "status": "invisible"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status, got %d", w.Code)
	}
}

func TestUpdatePresence_Online(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	body, _ := json.Marshal(map[string]string{"user_id": "u1", "status": "online"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdatePresence_Offline(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	body, _ := json.Marshal(map[string]string{"user_id": "u2", "status": "offline"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdatePresence_Typing(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	body, _ := json.Marshal(map[string]string{"user_id": "u3", "status": "typing", "chat_id": "chat-1"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─── setOnlineStatus ──────────────────────────────────────────────────────────

func TestSetOnlineStatus_MissingFields(t *testing.T) {
	r := setupPresenceRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/online", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetOnlineStatus_Valid(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	body, _ := json.Marshal(map[string]string{"user_id": "user-online"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/online", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp Presence
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != StatusOnline {
		t.Errorf("expected online, got %s", resp.Status)
	}
}

// ─── setOfflineStatus ─────────────────────────────────────────────────────────

func TestSetOfflineStatus_MissingFields(t *testing.T) {
	r := setupPresenceRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/offline", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetOfflineStatus_NewUser(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	body, _ := json.Marshal(map[string]string{"user_id": "new-offline-user"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/offline", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp Presence
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != StatusOffline {
		t.Errorf("expected offline, got %s", resp.Status)
	}
}

func TestSetOfflineStatus_ExistingUser(t *testing.T) {
	clearPresences()
	now := time.Now()
	mu.Lock()
	presences["existing-user"] = Presence{UserID: "existing-user", Status: StatusOnline, LastActivity: now, LastSeen: now}
	mu.Unlock()

	r := setupPresenceRouter()
	body, _ := json.Marshal(map[string]string{"user_id": "existing-user"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/offline", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp Presence
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != StatusOffline {
		t.Errorf("expected offline, got %s", resp.Status)
	}
}

// ─── setTypingStatus ──────────────────────────────────────────────────────────

func TestSetTypingStatus_MissingFields(t *testing.T) {
	r := setupPresenceRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/typing", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetTypingStatus_TypingTrue(t *testing.T) {
	clearPresences()
	r := setupPresenceRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"user_id": "typer-1",
		"chat_id": "chat-abc",
		"typing":  true,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/typing", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp Presence
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != StatusTyping {
		t.Errorf("expected typing, got %s", resp.Status)
	}
}

func TestSetTypingStatus_ExistingUserTypingTrue(t *testing.T) {
	clearPresences()
	now := time.Now()
	mu.Lock()
	presences["existing-typer"] = Presence{UserID: "existing-typer", Status: StatusOnline, LastActivity: now, LastSeen: now}
	mu.Unlock()

	r := setupPresenceRouter()
	body, _ := json.Marshal(map[string]interface{}{
		"user_id": "existing-typer",
		"chat_id": "chat-xyz",
		"typing":  true,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/typing", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
