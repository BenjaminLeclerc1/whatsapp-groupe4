package sharding

import (
	"context"
	"hash/fnv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ShardManager struct {
	Shards []*pgxpool.Pool
}

// NewShardManager initializes a connection pool for each database URL provided
func NewShardManager(urls []string) (*ShardManager, error) {
    var pools []*pgxpool.Pool
    for _, url := range urls {
        config, err := pgxpool.ParseConfig(url)
        if err != nil {
            return nil, err
        }

        // --- CONFIGURATION ECS-61 (Connection Pooling) ---
        // On définit les limites pour chaque shard individuellement
        config.MaxConns = 25                // Maximum de connexions par shard
        config.MinConns = 5                 // Toujours 5 connexions prêtes
        config.MaxConnIdleTime = 5 * time.Minute
        // --------------------------------------------------
        
        pool, err := pgxpool.NewWithConfig(context.Background(), config)
        if err != nil {
            return nil, err
        }
        pools = append(pools, pool)
    }
    return &ShardManager{Shards: pools}, nil
}

// GetShard hashes the key (UserID) to consistently pick the same database
func (s *ShardManager) GetShard(key string) *pgxpool.Pool {
	h := fnv.New32a()
	h.Write([]byte(key))
	// Modulo ensures the index is always within the bounds of our Shards array
	index := h.Sum32() % uint32(len(s.Shards))
	return s.Shards[index]
}