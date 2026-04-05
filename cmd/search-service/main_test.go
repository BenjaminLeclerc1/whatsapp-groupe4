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

func setupSearchRouter() *gin.Engine {
	r := gin.New()
	api := r.Group("/api/v1/search")
	{
		api.GET("/messages", searchMessages)
		api.GET("/messages/chat/:chatId", searchInChat)
		api.GET("/messages/user/:userId", searchByUser)
		api.POST("/index", indexMessage)
		api.DELETE("/index/:messageId", removeFromIndex)
		api.GET("/stats", getSearchStats)
	}
	return r
}

func clearSearchIndex() {
	mu.Lock()
	messages = make(map[string]Message)
	invertedIndex = make(map[string][]string)
	mu.Unlock()
}

// ─── Fonctions pures ──────────────────────────────────────────────────────────

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"Hello world", 2},
		{"Le chat est noir", 4},
		{"", 0},
		{"a", 1},
	}
	for _, tc := range tests {
		result := tokenize(tc.input)
		if len(result) != tc.expected {
			t.Errorf("tokenize(%q) = %d mots, attendu %d", tc.input, len(result), tc.expected)
		}
	}
}

func TestTokenizeNormalization(t *testing.T) {
	result := tokenize("HELLO World!")
	expected := []string{"hello", "world"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d words, got %d", len(expected), len(result))
	}
	for i, word := range result {
		if word != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], word)
		}
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}
	if !contains(slice, "banana") {
		t.Error("expected true for 'banana'")
	}
	if contains(slice, "orange") {
		t.Error("expected false for 'orange'")
	}
	if contains([]string{}, "test") {
		t.Error("expected false for empty slice")
	}
}

func TestRemoveString(t *testing.T) {
	slice := []string{"a", "b", "c", "b"}
	result := removeString(slice, "b")
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if contains(result, "b") {
		t.Error("'b' should be removed")
	}
}

func TestRemoveString_NotFound(t *testing.T) {
	slice := []string{"a", "b"}
	result := removeString(slice, "z")
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestCreateHighlight_Short(t *testing.T) {
	content := "Short message"
	result := createHighlight(content, []string{"short"})
	if result != content {
		t.Errorf("expected '%s', got '%s'", content, result)
	}
}

func TestCreateHighlight_Long(t *testing.T) {
	content := "This is a very long message that exceeds one hundred characters and should be truncated with ellipsis at the end"
	result := createHighlight(content, []string{"long"})
	if len(result) > 103 { // 100 + "..."
		t.Errorf("highlight too long: %d chars", len(result))
	}
}

func TestGetEnv(t *testing.T) {
	result := getEnv("SEARCH_TEST_NONEXISTENT", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
	os.Setenv("SEARCH_TEST_VAR", "val")
	defer os.Unsetenv("SEARCH_TEST_VAR")
	result = getEnv("SEARCH_TEST_VAR", "default")
	if result != "val" {
		t.Errorf("expected 'val', got '%s'", result)
	}
}

func TestMessageCreation(t *testing.T) {
	now := time.Now()
	msg := Message{
		ID:        "msg-123",
		SenderID:  "user-456",
		Content:   "Test message",
		ChatID:    "chat-789",
		CreatedAt: now,
		Status:    "sent",
	}
	if msg.ID != "msg-123" {
		t.Errorf("expected 'msg-123', got '%s'", msg.ID)
	}
}

func TestSearchResultScore(t *testing.T) {
	r := SearchResult{Message: Message{ID: "m1"}, Score: 5}
	if r.Score != 5 {
		t.Errorf("expected 5, got %d", r.Score)
	}
}

// ─── performSearch ────────────────────────────────────────────────────────────

func TestPerformSearch_Empty(t *testing.T) {
	clearSearchIndex()
	results := performSearch("hello", "", "")
	if len(results) != 0 {
		t.Errorf("expected 0 results on empty index, got %d", len(results))
	}
}

func TestPerformSearch_NoQuery(t *testing.T) {
	clearSearchIndex()
	results := performSearch("", "", "")
	if len(results) != 0 {
		t.Errorf("expected 0 for empty query, got %d", len(results))
	}
}

func TestPerformSearch_WithResults(t *testing.T) {
	clearSearchIndex()
	mu.Lock()
	messages["msg-1"] = Message{ID: "msg-1", Content: "hello world", ChatID: "chat-1", SenderID: "user-1"}
	invertedIndex["hello"] = []string{"msg-1"}
	invertedIndex["world"] = []string{"msg-1"}
	mu.Unlock()

	results := performSearch("hello", "", "")
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestPerformSearch_FilterByChat(t *testing.T) {
	clearSearchIndex()
	mu.Lock()
	messages["m1"] = Message{ID: "m1", Content: "test", ChatID: "chat-A", SenderID: "u1"}
	messages["m2"] = Message{ID: "m2", Content: "test", ChatID: "chat-B", SenderID: "u2"}
	invertedIndex["test"] = []string{"m1", "m2"}
	mu.Unlock()

	results := performSearch("test", "chat-A", "")
	if len(results) != 1 {
		t.Errorf("expected 1 result filtered by chat, got %d", len(results))
	}
}

func TestPerformSearch_FilterByUser(t *testing.T) {
	clearSearchIndex()
	mu.Lock()
	messages["m1"] = Message{ID: "m1", Content: "bonjour", ChatID: "c1", SenderID: "alice"}
	messages["m2"] = Message{ID: "m2", Content: "bonjour", ChatID: "c2", SenderID: "bob"}
	invertedIndex["bonjour"] = []string{"m1", "m2"}
	mu.Unlock()

	results := performSearch("bonjour", "", "alice")
	if len(results) != 1 {
		t.Errorf("expected 1 result filtered by user, got %d", len(results))
	}
}

// ─── indexMessage handler ────────────────────────────────────────────────────

func TestIndexMessage_MissingID(t *testing.T) {
	clearSearchIndex()
	r := setupSearchRouter()

	body, _ := json.Marshal(map[string]string{"content": "hello"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Sans ID, l'index crée quand même le message (ID vide) → 200
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestIndexMessage_Valid(t *testing.T) {
	clearSearchIndex()
	r := setupSearchRouter()

	msg := Message{ID: "idx-1", Content: "bonjour le monde", ChatID: "c1", SenderID: "u1"}
	body, _ := json.Marshal(msg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message_id"] != "idx-1" {
		t.Errorf("expected message_id 'idx-1', got %v", resp["message_id"])
	}
}

func TestIndexMessage_SetsCreatedAt(t *testing.T) {
	clearSearchIndex()
	r := setupSearchRouter()

	// Sans created_at, le handler doit le définir automatiquement
	msg := map[string]string{"id": "idx-ts", "content": "timestamp test", "sender_id": "u1", "chat_id": "c1"}
	body, _ := json.Marshal(msg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─── searchMessages handler ───────────────────────────────────────────────────

func TestSearchMessages_NoQuery(t *testing.T) {
	r := setupSearchRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearchMessages_NoResults(t *testing.T) {
	clearSearchIndex()
	r := setupSearchRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages?q=inexistant", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) != 0 {
		t.Errorf("expected 0 results, got %v", resp["count"])
	}
}

func TestSearchMessages_WithResults(t *testing.T) {
	clearSearchIndex()
	mu.Lock()
	messages["s1"] = Message{ID: "s1", Content: "salut monde"}
	invertedIndex["salut"] = []string{"s1"}
	invertedIndex["monde"] = []string{"s1"}
	mu.Unlock()

	r := setupSearchRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages?q=salut", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) < 1 {
		t.Errorf("expected at least 1 result")
	}
}

// ─── searchInChat handler ─────────────────────────────────────────────────────

func TestSearchInChat(t *testing.T) {
	clearSearchIndex()
	mu.Lock()
	messages["c1m1"] = Message{ID: "c1m1", Content: "chat message", ChatID: "chat-99"}
	invertedIndex["chat"] = []string{"c1m1"}
	invertedIndex["message"] = []string{"c1m1"}
	mu.Unlock()

	r := setupSearchRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages/chat/chat-99?q=chat", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSearchInChat_NoMatch(t *testing.T) {
	clearSearchIndex()
	r := setupSearchRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages/chat/chat-000?q=nothing", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─── searchByUser handler ─────────────────────────────────────────────────────

func TestSearchByUser(t *testing.T) {
	clearSearchIndex()
	mu.Lock()
	messages["um1"] = Message{ID: "um1", Content: "user content", SenderID: "user-42"}
	invertedIndex["user"] = []string{"um1"}
	invertedIndex["content"] = []string{"um1"}
	mu.Unlock()

	r := setupSearchRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages/user/user-42?q=user", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─── removeFromIndex handler ──────────────────────────────────────────────────

func TestRemoveFromIndex_NotFound(t *testing.T) {
	clearSearchIndex()
	r := setupSearchRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/search/index/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRemoveFromIndex_Found(t *testing.T) {
	clearSearchIndex()
	mu.Lock()
	messages["del-1"] = Message{ID: "del-1", Content: "delete me"}
	invertedIndex["delete"] = []string{"del-1"}
	invertedIndex["me"] = []string{"del-1"}
	mu.Unlock()

	r := setupSearchRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/search/index/del-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─── getSearchStats handler ───────────────────────────────────────────────────

func TestGetSearchStats_Empty(t *testing.T) {
	clearSearchIndex()
	r := setupSearchRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/stats", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["indexed_messages"].(float64) != 0 {
		t.Errorf("expected 0 messages, got %v", resp["indexed_messages"])
	}
}

func TestGetSearchStats_AfterIndex(t *testing.T) {
	clearSearchIndex()
	mu.Lock()
	messages["s1"] = Message{ID: "s1", Content: "test"}
	invertedIndex["test"] = []string{"s1"}
	mu.Unlock()

	r := setupSearchRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/stats", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["indexed_messages"].(float64) != 1 {
		t.Errorf("expected 1 message, got %v", resp["indexed_messages"])
	}
}
