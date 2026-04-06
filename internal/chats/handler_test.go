package chats

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type handlerChatServiceStub struct {
	createFn func(ctx context.Context, creatorID string, req CreateChatRequest) (Chat, error)
	listFn   func(ctx context.Context, userID string) ([]Chat, error)
	updateFn func(ctx context.Context, id string, name string) error
	deleteFn func(ctx context.Context, id string) error
}

func (s handlerChatServiceStub) CreateChat(ctx context.Context, creatorID string, req CreateChatRequest) (Chat, error) {
	return s.createFn(ctx, creatorID, req)
}
func (s handlerChatServiceStub) GetMyChats(ctx context.Context, userID string) ([]Chat, error) {
	return s.listFn(ctx, userID)
}
func (s handlerChatServiceStub) UpdateChat(ctx context.Context, id string, name string) error {
	return s.updateFn(ctx, id, name)
}
func (s handlerChatServiceStub) DeleteChat(ctx context.Context, id string) error {
	return s.deleteFn(ctx, id)
}

func TestCreateChatHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(handlerChatServiceStub{
		createFn: func(_ context.Context, creatorID string, _ CreateChatRequest) (Chat, error) {
			if creatorID != "user-1" {
				t.Fatalf("unexpected creator id: %s", creatorID)
			}
			return Chat{ID: "chat-1"}, nil
		},
		listFn:   func(context.Context, string) ([]Chat, error) { return nil, nil },
		updateFn: func(context.Context, string, string) error { return nil },
		deleteFn: func(context.Context, string) error { return nil },
	})

	r := gin.New()
	r.POST("/chats", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		h.CreateChat(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chats", strings.NewReader(`{"participants":["u2"],"type":"group","name":"g1"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestCreateChat_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(handlerChatServiceStub{
		createFn: func(context.Context, string, CreateChatRequest) (Chat, error) { return Chat{}, nil },
		listFn:   func(context.Context, string) ([]Chat, error) { return nil, nil },
		updateFn: func(context.Context, string, string) error { return nil },
		deleteFn: func(context.Context, string) error { return nil },
	})

	r := gin.New()
	r.POST("/chats", h.CreateChat)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chats", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetMyChats_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(handlerChatServiceStub{
		createFn: func(context.Context, string, CreateChatRequest) (Chat, error) { return Chat{}, nil },
		listFn:   func(context.Context, string) ([]Chat, error) { return nil, errors.New("db down") },
		updateFn: func(context.Context, string, string) error { return nil },
		deleteFn: func(context.Context, string) error { return nil },
	})

	r := gin.New()
	r.GET("/chats", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		h.GetMyChats(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/chats", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestUpdateAndDeleteChat_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(handlerChatServiceStub{
		createFn: func(context.Context, string, CreateChatRequest) (Chat, error) { return Chat{}, nil },
		listFn:   func(context.Context, string) ([]Chat, error) { return nil, nil },
		updateFn: func(_ context.Context, id, name string) error {
			if id != "chat-1" || name != "new-name" {
				t.Fatalf("unexpected update payload id=%s name=%s", id, name)
			}
			return nil
		},
		deleteFn: func(_ context.Context, id string) error {
			if id != "chat-1" {
				t.Fatalf("unexpected delete id: %s", id)
			}
			return nil
		},
	})

	r := gin.New()
	r.PUT("/chats/:id", h.UpdateChat)
	r.DELETE("/chats/:id", h.DeleteChat)

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPut, "/chats/chat-1", strings.NewReader(`{"name":"new-name"}`))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d", w1.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(w1.Body.Bytes(), &body)
	if body["message"] == "" {
		t.Fatal("expected update confirmation message")
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodDelete, "/chats/chat-1", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", w2.Code)
	}
}
