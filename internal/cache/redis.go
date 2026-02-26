package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient(addr string) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisClient{Client: rdb}
}

// SetSession now converts the object to JSON before saving
func (r *RedisClient) SetSession(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, "session:"+key, data, expiration).Err()
}

// GetSession now takes a pointer (target) to unmarshal the JSON into
func (r *RedisClient) GetSession(ctx context.Context, key string, target interface{}) error {
	val, err := r.Client.Get(ctx, "session:"+key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), target)
}