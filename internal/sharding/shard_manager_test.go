package sharding

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewShardManager_InvalidURL(t *testing.T) {
	_, err := NewShardManager([]string{"bad://\x00-url"})
	if err == nil {
		t.Fatal("expected error for invalid shard URL")
	}
}

func TestGetShard_DeterministicAndInRange(t *testing.T) {
	s := &ShardManager{
		Shards: []*pgxpool.Pool{{}, {}, {}},
	}

	key := "user-123"
	first := s.GetShard(key)
	if first == nil {
		t.Fatal("expected non-nil shard")
	}

	for i := 0; i < 5; i++ {
		if s.GetShard(key) != first {
			t.Fatal("expected deterministic shard selection for same key")
		}
	}

	seen := map[*pgxpool.Pool]bool{}
	for _, candidate := range []string{"u1", "u2", "u3", "u4", "u5"} {
		seen[s.GetShard(candidate)] = true
	}
	if len(seen) == 0 {
		t.Fatal("expected at least one shard selected")
	}
}
