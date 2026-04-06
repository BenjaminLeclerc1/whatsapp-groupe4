package redis

import "testing"

func TestNewRedisClient_SetsAddress(t *testing.T) {
	const addr = "localhost:6379"
	client := NewRedisClient(addr)
	if client == nil {
		t.Fatal("expected client, got nil")
	}
	if got := client.Options().Addr; got != addr {
		t.Fatalf("expected addr %q, got %q", addr, got)
	}
}
