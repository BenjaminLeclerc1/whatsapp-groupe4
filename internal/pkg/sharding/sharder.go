package sharding

import (
	"hash/fnv"
)

// GetShardNode maps a conversation ID to a shard index in [0, totalShards).
func GetShardNode(conversationID string, totalShards int) int {
	if totalShards <= 0 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(conversationID))
	return int(h.Sum32() % uint32(totalShards))
}
