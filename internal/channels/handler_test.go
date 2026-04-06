package channels

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const (
	validChannelUUID = "123e4567-e89b-12d3-a456-426614174111"

	channelCollectionPath = "/channels"
	channelRouteIDByID    = "/channels/:id"
	channelPathPrefix     = "/channels/"
	channelRouteMembers   = channelRouteIDByID + "/members"
	channelRouteMsgs      = channelRouteIDByID + "/messages"

	headerContentType    = "Content-Type"
	contentTypeJSON      = "application/json"
	errExpected200Format = "expected 200, got %d"
)

type channelHandlerServiceStub struct {
	createFn      func(ctx context.Context, userID string, req CreateChannelRequest) (ChannelResponse, error)
	getFn         func(ctx context.Context, userID, channelID string) (ChannelResponse, error)
	updateFn      func(ctx context.Context, userID, channelID string, req UpdateChannelRequest) (ChannelResponse, error)
	deleteFn      func(ctx context.Context, userID, channelID string) error
	listFn        func(ctx context.Context, userID string) (ChannelListResponse, error)
	addMemberFn   func(ctx context.Context, userID, channelID string, req AddMemberRequest) (Participant, error)
	removeFn      func(ctx context.Context, userID, channelID, targetUserID string) error
	listMembersFn func(ctx context.Context, userID, channelID string) (MemberListResponse, error)
	messagesFn    func(ctx context.Context, userID, channelID, cursor string, limit int) (MessageListResponse, error)
}

func (s channelHandlerServiceStub) CreateChannel(ctx context.Context, userID string, req CreateChannelRequest) (ChannelResponse, error) {
	return s.createFn(ctx, userID, req)
}
func (s channelHandlerServiceStub) GetChannel(ctx context.Context, userID, channelID string) (ChannelResponse, error) {
	return s.getFn(ctx, userID, channelID)
}
func (s channelHandlerServiceStub) UpdateChannel(ctx context.Context, userID, channelID string, req UpdateChannelRequest) (ChannelResponse, error) {
	return s.updateFn(ctx, userID, channelID, req)
}
func (s channelHandlerServiceStub) DeleteChannel(ctx context.Context, userID, channelID string) error {
	return s.deleteFn(ctx, userID, channelID)
}
func (s channelHandlerServiceStub) ListMyChannels(ctx context.Context, userID string) (ChannelListResponse, error) {
	return s.listFn(ctx, userID)
}
func (s channelHandlerServiceStub) AddMember(ctx context.Context, userID, channelID string, req AddMemberRequest) (Participant, error) {
	return s.addMemberFn(ctx, userID, channelID, req)
}
func (s channelHandlerServiceStub) RemoveMember(ctx context.Context, userID, channelID, targetUserID string) error {
	return s.removeFn(ctx, userID, channelID, targetUserID)
}
func (s channelHandlerServiceStub) ListMembers(ctx context.Context, userID, channelID string) (MemberListResponse, error) {
	return s.listMembersFn(ctx, userID, channelID)
}
func (s channelHandlerServiceStub) ListMessages(ctx context.Context, userID, channelID, cursor string, limit int) (MessageListResponse, error) {
	return s.messagesFn(ctx, userID, channelID, cursor, limit)
}

func baseChannelStub() channelHandlerServiceStub {
	return channelHandlerServiceStub{
		createFn: func(context.Context, string, CreateChannelRequest) (ChannelResponse, error) {
			return ChannelResponse{Channel: Channel{ID: "c1"}}, nil
		},
		getFn: func(context.Context, string, string) (ChannelResponse, error) {
			return ChannelResponse{Channel: Channel{ID: "c1"}}, nil
		},
		updateFn: func(context.Context, string, string, UpdateChannelRequest) (ChannelResponse, error) {
			return ChannelResponse{Channel: Channel{ID: "c1"}}, nil
		},
		deleteFn:      func(context.Context, string, string) error { return nil },
		listFn:        func(context.Context, string) (ChannelListResponse, error) { return ChannelListResponse{}, nil },
		addMemberFn:   func(context.Context, string, string, AddMemberRequest) (Participant, error) { return Participant{}, nil },
		removeFn:      func(context.Context, string, string, string) error { return nil },
		listMembersFn: func(context.Context, string, string) (MemberListResponse, error) { return MemberListResponse{}, nil },
		messagesFn:    func(context.Context, string, string, string, int) (MessageListResponse, error) { return MessageListResponse{}, nil },
	}
}

func TestGetChannel_InvalidChannelID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(baseChannelStub())
	r := gin.New()
	r.GET(channelRouteIDByID, func(c *gin.Context) {
		c.Set("user_id", validChannelUUID)
		h.GetChannel(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, channelPathPrefix+"bad-id", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateChannel_AndDeleteChannel_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(baseChannelStub())
	r := gin.New()
	r.POST(channelCollectionPath, func(c *gin.Context) {
		c.Set("user_id", validChannelUUID)
		h.CreateChannel(c)
	})
	r.DELETE(channelRouteIDByID, func(c *gin.Context) {
		c.Set("user_id", validChannelUUID)
		h.DeleteChannel(c)
	})

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, channelCollectionPath, strings.NewReader(`{"name":"general","is_group":true}`))
	req1.Header.Set(headerContentType, contentTypeJSON)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodDelete, channelPathPrefix+validChannelUUID, nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf(errExpected200Format, w2.Code)
	}
}

func TestUpdateChannel_ServiceNotFoundMaps404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := baseChannelStub()
	stub.updateFn = func(context.Context, string, string, UpdateChannelRequest) (ChannelResponse, error) {
		return ChannelResponse{}, errors.New("channel not found")
	}
	h := NewHandler(stub)
	r := gin.New()
	r.PUT(channelRouteIDByID, func(c *gin.Context) {
		c.Set("user_id", validChannelUUID)
		h.UpdateChannel(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, channelPathPrefix+validChannelUUID, strings.NewReader(`{"name":"new"}`))
	req.Header.Set(headerContentType, contentTypeJSON)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAddMember_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(baseChannelStub())
	r := gin.New()
	r.POST(channelRouteMembers, func(c *gin.Context) {
		c.Set("user_id", validChannelUUID)
		h.AddMember(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, channelPathPrefix+validChannelUUID+"/members", strings.NewReader(`{"user_id":"not-uuid"}`))
	req.Header.Set(headerContentType, contentTypeJSON)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListMembers_AndListMessages_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := baseChannelStub()
	stub.listMembersFn = func(context.Context, string, string) (MemberListResponse, error) {
		return MemberListResponse{Count: 1}, nil
	}
	stub.messagesFn = func(context.Context, string, string, string, int) (MessageListResponse, error) {
		return MessageListResponse{Count: 1, ChatID: validChannelUUID}, nil
	}
	h := NewHandler(stub)
	r := gin.New()
	r.GET(channelRouteMembers, func(c *gin.Context) {
		c.Set("user_id", validChannelUUID)
		h.ListMembers(c)
	})
	r.GET(channelRouteMsgs, func(c *gin.Context) {
		c.Set("user_id", validChannelUUID)
		h.ListMessages(c)
	})

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, channelPathPrefix+validChannelUUID+"/members", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf(errExpected200Format, w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, channelPathPrefix+validChannelUUID+"/messages?limit=10", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf(errExpected200Format, w2.Code)
	}
}
