package cache

import (
	"context"
	"testing"
	"time"
)

func TestNewRedisClient_SetsAddress(t *testing.T) {
	const addr = "127.0.0.1:6379"
	rc := NewRedisClient(addr)
	if rc == nil || rc.Client == nil {
		t.Fatal("expected redis client to be initialized")
	}
	if got := rc.Client.Options().Addr; got != addr {
		t.Fatalf("expected addr %q, got %q", addr, got)
	}
}

func TestSetSession_MarshalError(t *testing.T) {
	rc := NewRedisClient("127.0.0.1:6379")

	// channels cannot be marshaled to JSON, SetSession should return early.
	err := rc.SetSession(context.Background(), "k1", make(chan int), time.Minute)
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
}
