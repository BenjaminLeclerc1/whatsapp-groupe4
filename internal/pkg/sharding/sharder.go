// internal/pkg/sharding/sharder.go

func GetShardNode(conversationID string, totalShards int) int {
    // Use a Hash function to consistently map a ID to a number
    hash := fnv.New32a()
    hash.Write([]byte(conversationID))
    return int(hash.Sum32() % uint32(totalShards))
}