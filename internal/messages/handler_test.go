package messages

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const validUUID = "123e4567-e89b-12d3-a456-426614174000"

type messageHandlerServiceStub struct {
	sendFn    func(ctx context.Context, userID string, req SendMessageRequest) (Message, error)
	getFn     func(ctx context.Context, userID, messageID string) (Message, error)
	historyFn func(ctx context.Context, userID, chatID, cursor string, limit int) (MessageListResponse, error)
	deleteFn  func(ctx context.Context, userID, messageID string) error
}

func (s messageHandlerServiceStub) SendMessage(ctx context.Context, userID string, req SendMessageRequest) (Message, error) {
	return s.sendFn(ctx, userID, req)
}
func (s messageHandlerServiceStub) GetMessage(ctx context.Context, userID, messageID string) (Message, error) {
	return s.getFn(ctx, userID, messageID)
}
func (s messageHandlerServiceStub) GetMessageHistory(ctx context.Context, userID, chatID, cursor string, limit int) (MessageListResponse, error) {
	return s.historyFn(ctx, userID, chatID, cursor, limit)
}
func (s messageHandlerServiceStub) DeleteMessage(ctx context.Context, userID, messageID string) error {
	return s.deleteFn(ctx, userID, messageID)
}

func TestRegisterRoutes_AndSendMessageSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(messageHandlerServiceStub{
		sendFn: func(_ context.Context, userID string, req SendMessageRequest) (Message, error) {
			if userID != validUUID {
				t.Fatalf("unexpected user id: %s", userID)
			}
			return Message{ID: "m1", ChatID: req.ChatID, Content: req.Content}, nil
		},
		getFn:     func(context.Context, string, string) (Message, error) { return Message{}, nil },
		historyFn: func(context.Context, string, string, string, int) (MessageListResponse, error) { return MessageListResponse{}, nil },
		deleteFn:  func(context.Context, string, string) error { return nil },
	})

	r := gin.New()
	api := r.Group("/api/v1/messages")
	api.Use(func(c *gin.Context) {
		c.Set("user_id", validUUID)
		c.Next()
	})
	h.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"chat_id":"`+validUUID+`","content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestSendMessage_InvalidChatID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(messageHandlerServiceStub{
		sendFn:    func(context.Context, string, SendMessageRequest) (Message, error) { return Message{}, nil },
		getFn:     func(context.Context, string, string) (Message, error) { return Message{}, nil },
		historyFn: func(context.Context, string, string, string, int) (MessageListResponse, error) { return MessageListResponse{}, nil },
		deleteFn:  func(context.Context, string, string) error { return nil },
	})

	r := gin.New()
	r.POST("/messages", func(c *gin.Context) {
		c.Set("user_id", validUUID)
		h.SendMessage(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(`{"chat_id":"bad-id","content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetMessage_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(messageHandlerServiceStub{
		sendFn:    func(context.Context, string, SendMessageRequest) (Message, error) { return Message{}, nil },
		getFn:     func(context.Context, string, string) (Message, error) { return Message{}, nil },
		historyFn: func(context.Context, string, string, string, int) (MessageListResponse, error) { return MessageListResponse{}, nil },
		deleteFn:  func(context.Context, string, string) error { return nil },
	})
	r := gin.New()
	r.GET("/messages/:id", func(c *gin.Context) {
		c.Set("user_id", validUUID)
		h.GetMessage(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/messages/not-a-uuid", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetMessageHistory_ForbiddenMaps403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(messageHandlerServiceStub{
		sendFn: func(context.Context, string, SendMessageRequest) (Message, error) { return Message{}, nil },
		getFn:  func(context.Context, string, string) (Message, error) { return Message{}, nil },
		historyFn: func(context.Context, string, string, string, int) (MessageListResponse, error) {
			return MessageListResponse{}, errors.New("forbidden: not member")
		},
		deleteFn: func(context.Context, string, string) error { return nil },
	})

	r := gin.New()
	r.GET("/messages/chat/:chatId", func(c *gin.Context) {
		c.Set("user_id", validUUID)
		h.GetMessageHistory(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/messages/chat/"+validUUID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestDeleteMessage_NotFoundMaps404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(messageHandlerServiceStub{
		sendFn:    func(context.Context, string, SendMessageRequest) (Message, error) { return Message{}, nil },
		getFn:     func(context.Context, string, string) (Message, error) { return Message{}, nil },
		historyFn: func(context.Context, string, string, string, int) (MessageListResponse, error) { return MessageListResponse{}, nil },
		deleteFn: func(context.Context, string, string) error {
			return errors.New("message not found")
		},
	})

	r := gin.New()
	r.DELETE("/messages/:id", func(c *gin.Context) {
		c.Set("user_id", validUUID)
		h.DeleteMessage(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/messages/"+validUUID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
