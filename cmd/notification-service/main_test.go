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

func setupRouter() *gin.Engine {
	// Réinitialise le state entre les tests
	mu.Lock()
	notifications = make(map[string]Notification)
	userNotifications = make(map[string][]string)
	mu.Unlock()

	r := gin.New()
	api := r.Group("/api/v1/notification")
	api.GET("/count/:userId", getUnreadCount)
	api.GET("/user/:userId", getNotificationsByUser)
	api.GET("/user/:userId/unread", getUnreadNotifications)
	api.GET("/:id", getNotificationByID)
	api.POST("", createNotification)
	api.PUT("/:id/read", markAsRead)
	api.PUT("/user/:userId/read-all", markAllAsRead)
	api.DELETE("/:id", deleteNotification)
	return r
}

// ─── getEnv ───────────────────────────────────────────────────────────────────

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("NOTIF_TEST_NONEXISTENT", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("NOTIF_TEST_VAR", "set_value")
	defer os.Unsetenv("NOTIF_TEST_VAR")
	result := getEnv("NOTIF_TEST_VAR", "default")
	if result != "set_value" {
		t.Errorf("expected 'set_value', got '%s'", result)
	}
}

// ─── createNotification ───────────────────────────────────────────────────────

func TestCreateNotification_Success(t *testing.T) {
	r := setupRouter()

	body := map[string]string{
		"user_id": "user-1",
		"content": "Tu as reçu un message",
	}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var notif Notification
	json.Unmarshal(w.Body.Bytes(), &notif)
	if notif.UserID != "user-1" {
		t.Errorf("unexpected UserID: %s", notif.UserID)
	}
	if notif.Read {
		t.Error("new notification should be unread")
	}
	if notif.Type != "message" {
		t.Errorf("expected default type 'message', got '%s'", notif.Type)
	}
}

func TestCreateNotification_MissingFields(t *testing.T) {
	r := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateNotification_CustomType(t *testing.T) {
	r := setupRouter()

	body := map[string]string{
		"user_id": "user-2",
		"content": "Alerte système",
		"type":    "system",
	}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var notif Notification
	json.Unmarshal(w.Body.Bytes(), &notif)
	if notif.Type != "system" {
		t.Errorf("expected type 'system', got '%s'", notif.Type)
	}
}

// ─── getUnreadCount ───────────────────────────────────────────────────────────

func TestGetUnreadCount_Zero(t *testing.T) {
	r := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/count/user-unknown", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp NotificationCount
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.UnreadCount != 0 {
		t.Errorf("expected 0, got %d", resp.UnreadCount)
	}
}

func TestGetUnreadCount_AfterCreate(t *testing.T) {
	r := setupRouter()

	// Crée une notification
	body := map[string]string{"user_id": "user-3", "content": "msg"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Vérifie le count
	w := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/notification/count/user-3", nil)
	r.ServeHTTP(w, req2)

	var resp NotificationCount
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.UnreadCount != 1 {
		t.Errorf("expected 1 unread, got %d", resp.UnreadCount)
	}
}

// ─── getNotificationsByUser ───────────────────────────────────────────────────

func TestGetNotificationsByUser_Empty(t *testing.T) {
	r := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/user/nobody", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) != 0 {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

func TestGetNotificationsByUser_WithNotifications(t *testing.T) {
	r := setupRouter()

	for i := 0; i < 3; i++ {
		body := map[string]string{"user_id": "user-4", "content": "msg"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/user/user-4", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) != 3 {
		t.Errorf("expected 3, got %v", resp["count"])
	}
}

// ─── getNotificationByID ─────────────────────────────────────────────────────

func TestGetNotificationByID_NotFound(t *testing.T) {
	r := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/nonexistent-id", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetNotificationByID_Found(t *testing.T) {
	r := setupRouter()

	// Crée une notification
	body := map[string]string{"user_id": "user-5", "content": "test"}
	b, _ := json.Marshal(body)
	wCreate := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wCreate, req)

	var created Notification
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	// Récupère par ID
	w := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/notification/"+created.ID, nil)
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─── markAsRead ───────────────────────────────────────────────────────────────

func TestMarkAsRead_NotFound(t *testing.T) {
	r := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification/bad-id/read", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMarkAsRead_Success(t *testing.T) {
	r := setupRouter()

	// Crée une notification
	body := map[string]string{"user_id": "user-6", "content": "lu"}
	b, _ := json.Marshal(body)
	wCreate := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wCreate, req)

	var created Notification
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	// Marque comme lue
	w := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/notification/"+created.ID+"/read", nil)
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var notif Notification
	json.Unmarshal(w.Body.Bytes(), &notif)
	if !notif.Read {
		t.Error("notification should be marked as read")
	}

	// Vérifie que le count est maintenant 0
	wCount := httptest.NewRecorder()
	reqCount := httptest.NewRequest(http.MethodGet, "/api/v1/notification/count/user-6", nil)
	r.ServeHTTP(wCount, reqCount)
	var count NotificationCount
	json.Unmarshal(wCount.Body.Bytes(), &count)
	if count.UnreadCount != 0 {
		t.Errorf("expected 0 unread after markAsRead, got %d", count.UnreadCount)
	}
}

// ─── markAllAsRead ────────────────────────────────────────────────────────────

func TestMarkAllAsRead(t *testing.T) {
	r := setupRouter()

	for i := 0; i < 3; i++ {
		body := map[string]string{"user_id": "user-7", "content": "msg"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification/user/user-7/read-all", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["marked_count"].(float64) != 3 {
		t.Errorf("expected 3 marked, got %v", resp["marked_count"])
	}
}

// ─── deleteNotification ───────────────────────────────────────────────────────

func TestDeleteNotification_NotFound(t *testing.T) {
	r := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notification/ghost-id", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteNotification_Success(t *testing.T) {
	r := setupRouter()

	body := map[string]string{"user_id": "user-8", "content": "à supprimer"}
	b, _ := json.Marshal(body)
	wCreate := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wCreate, req)

	var created Notification
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	// Supprime
	w := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/notification/"+created.ID, nil)
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Vérifie qu'il n'existe plus
	wGet := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/notification/"+created.ID, nil)
	r.ServeHTTP(wGet, req3)
	if wGet.Code != http.StatusNotFound {
		t.Error("notification should be deleted")
	}
}

// ─── getUnreadNotifications ───────────────────────────────────────────────────

func TestGetUnreadNotifications(t *testing.T) {
	r := setupRouter()

	// Crée 2 notifications
	for i := 0; i < 2; i++ {
		body := map[string]string{"user_id": "user-9", "content": "msg"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notification", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/user/user-9/unread", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) != 2 {
		t.Errorf("expected 2 unread, got %v", resp["count"])
	}
}
