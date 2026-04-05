package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ─── IsValidUUID ──────────────────────────────────────────────────────────────

func TestIsValidUUID_Valid(t *testing.T) {
	valid := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"123e4567-e89b-12d3-a456-426614174000",
		"00000000-0000-0000-0000-000000000000",
	}
	for _, id := range valid {
		if !IsValidUUID(id) {
			t.Errorf("expected '%s' to be a valid UUID", id)
		}
	}
}

func TestIsValidUUID_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"not-a-uuid",
		"550e8400-e29b-41d4-a716",
		"550e8400e29b41d4a716446655440000",
		"ZZZZZZZZ-e29b-41d4-a716-446655440000",
	}
	for _, id := range invalid {
		if IsValidUUID(id) {
			t.Errorf("expected '%s' to be invalid", id)
		}
	}
}

// ─── ParsePagination ──────────────────────────────────────────────────────────

func TestParsePagination_Defaults(t *testing.T) {
	r := gin.New()
	var gotCursor string
	var gotLimit int

	r.GET("/test", func(c *gin.Context) {
		gotCursor, gotLimit = ParsePagination(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if gotCursor != "" {
		t.Errorf("expected empty cursor, got '%s'", gotCursor)
	}
	if gotLimit != 50 {
		t.Errorf("expected limit 50, got %d", gotLimit)
	}
}

func TestParsePagination_ValidCursor(t *testing.T) {
	r := gin.New()
	var gotCursor string

	r.GET("/test", func(c *gin.Context) {
		gotCursor, _ = ParsePagination(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?cursor=550e8400-e29b-41d4-a716-446655440000", nil)
	r.ServeHTTP(w, req)

	if gotCursor != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected valid cursor, got '%s'", gotCursor)
	}
}

func TestParsePagination_InvalidCursor(t *testing.T) {
	r := gin.New()
	var gotCursor string

	r.GET("/test", func(c *gin.Context) {
		gotCursor, _ = ParsePagination(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?cursor=not-valid", nil)
	r.ServeHTTP(w, req)

	if gotCursor != "" {
		t.Errorf("expected empty cursor for invalid UUID, got '%s'", gotCursor)
	}
}

func TestParsePagination_ValidLimit(t *testing.T) {
	r := gin.New()
	var gotLimit int

	r.GET("/test", func(c *gin.Context) {
		_, gotLimit = ParsePagination(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=25", nil)
	r.ServeHTTP(w, req)

	if gotLimit != 25 {
		t.Errorf("expected limit 25, got %d", gotLimit)
	}
}

func TestParsePagination_LimitTooHigh(t *testing.T) {
	r := gin.New()
	var gotLimit int

	r.GET("/test", func(c *gin.Context) {
		_, gotLimit = ParsePagination(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=999", nil)
	r.ServeHTTP(w, req)

	if gotLimit != 50 {
		t.Errorf("expected default limit 50 for too-high value, got %d", gotLimit)
	}
}

func TestParsePagination_LimitZero(t *testing.T) {
	r := gin.New()
	var gotLimit int

	r.GET("/test", func(c *gin.Context) {
		_, gotLimit = ParsePagination(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=0", nil)
	r.ServeHTTP(w, req)

	if gotLimit != 50 {
		t.Errorf("expected default limit 50 for zero, got %d", gotLimit)
	}
}

func TestParsePagination_LimitNotNumber(t *testing.T) {
	r := gin.New()
	var gotLimit int

	r.GET("/test", func(c *gin.Context) {
		_, gotLimit = ParsePagination(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=abc", nil)
	r.ServeHTTP(w, req)

	if gotLimit != 50 {
		t.Errorf("expected default limit 50 for non-number, got %d", gotLimit)
	}
}

// ─── ExtractUserID middleware ─────────────────────────────────────────────────

func setupExtractRouter() *gin.Engine {
	r := gin.New()
	r.GET("/test", ExtractUserID(), func(c *gin.Context) {
		userID := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestExtractUserID_Missing(t *testing.T) {
	r := setupExtractRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing X-User-ID, got %d", w.Code)
	}
}

func TestExtractUserID_InvalidFormat(t *testing.T) {
	r := setupExtractRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "not-a-uuid")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestExtractUserID_Valid(t *testing.T) {
	r := setupExtractRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "550e8400-e29b-41d4-a716-446655440000")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetUserID_FromContext(t *testing.T) {
	r := gin.New()
	var extractedID string

	r.GET("/test", ExtractUserID(), func(c *gin.Context) {
		extractedID = GetUserID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "550e8400-e29b-41d4-a716-446655440000")
	r.ServeHTTP(w, req)

	if extractedID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected extracted user ID, got '%s'", extractedID)
	}
}

func TestGetUserID_NotSet(t *testing.T) {
	r := gin.New()
	var extractedID string

	r.GET("/test", func(c *gin.Context) {
		extractedID = GetUserID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if extractedID != "" {
		t.Errorf("expected empty ID when not set, got '%s'", extractedID)
	}
}

// ─── RateLimiter ─────────────────────────────────────────────────────────────

func TestNewRateLimiter_Created(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	if rl == nil {
		t.Fatal("expected non-nil RateLimiter")
	}
	rl.Stop()
}

func TestRateLimiter_AllowsRequests(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	defer rl.Stop()

	for i := 0; i < 5; i++ {
		if !rl.allow("user-1") {
			t.Errorf("expected request %d to be allowed", i+1)
		}
	}
}

func TestRateLimiter_BlocksAfterMax(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	defer rl.Stop()

	for i := 0; i < 3; i++ {
		rl.allow("user-block")
	}

	if rl.allow("user-block") {
		t.Error("expected request to be blocked after max")
	}
}

func TestRateLimiter_DifferentUsers(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	defer rl.Stop()

	// user1 épuise son quota
	rl.allow("user-1")
	rl.allow("user-1") // bloqué

	// user2 doit encore être autorisé
	if !rl.allow("user-2") {
		t.Error("user-2 should be allowed independently")
	}
}

func TestRateLimiter_Middleware_NoUserID(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	defer rl.Stop()

	r := gin.New()
	r.GET("/test", rl.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Sans user_id dans le contexte, le middleware passe
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no user_id, got %d", w.Code)
	}
}

func TestRateLimiter_Middleware_WithUserID(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	defer rl.Stop()

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set(UserIDKey, "user-rl")
		c.Next()
	}, rl.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRateLimiter_Middleware_RateLimit(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	defer rl.Stop()

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set(UserIDKey, "user-limited")
		c.Next()
	}, rl.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Épuiser le quota (2 requêtes)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)
	}

	// La 3ème doit être bloquée
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}
