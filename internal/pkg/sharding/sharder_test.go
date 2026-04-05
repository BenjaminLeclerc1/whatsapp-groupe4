package sharding

import (
	"testing"
)

func TestGetShardNode_InRange(t *testing.T) {
	totalShards := 4
	ids := []string{"user-1", "user-2", "chat-abc", "msg-xyz", "hello-world"}

	for _, id := range ids {
		shard := GetShardNode(id, totalShards)
		if shard < 0 || shard >= totalShards {
			t.Errorf("shard %d out of range [0, %d) for id '%s'", shard, totalShards, id)
		}
	}
}

func TestGetShardNode_ZeroShards(t *testing.T) {
	result := GetShardNode("any-id", 0)
	if result != 0 {
		t.Errorf("expected 0 for totalShards=0, got %d", result)
	}
}

func TestGetShardNode_NegativeShards(t *testing.T) {
	result := GetShardNode("any-id", -1)
	if result != 0 {
		t.Errorf("expected 0 for negative totalShards, got %d", result)
	}
}

func TestGetShardNode_OneShard(t *testing.T) {
	result := GetShardNode("any-id", 1)
	if result != 0 {
		t.Errorf("expected 0 for single shard, got %d", result)
	}
}

func TestGetShardNode_Deterministic(t *testing.T) {
	id := "user-stable"
	r1 := GetShardNode(id, 8)
	r2 := GetShardNode(id, 8)
	if r1 != r2 {
		t.Errorf("expected same shard for same id: got %d and %d", r1, r2)
	}
}

func TestGetShardNode_EmptyString(t *testing.T) {
	result := GetShardNode("", 4)
	if result < 0 || result >= 4 {
		t.Errorf("shard %d out of range for empty string", result)
	}
}

func TestGetShardNode_Distribution(t *testing.T) {
	totalShards := 4
	counts := make(map[int]int)
	ids := []string{
		"user-1", "user-2", "user-3", "user-4",
		"chat-a", "chat-b", "msg-1", "msg-2",
	}
	for _, id := range ids {
		shard := GetShardNode(id, totalShards)
		counts[shard]++
	}
	// Vérifie qu'au moins 2 shards différents sont utilisés (pas tout sur 1)
	if len(counts) < 2 {
		t.Errorf("expected distribution across multiple shards, got %v", counts)
	}
}
