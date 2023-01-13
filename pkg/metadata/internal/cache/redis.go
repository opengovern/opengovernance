package cache

import (
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	KeyMetadataPrefix = "metadata:"
)

type MetadataRedisCache struct {
	redis      *redis.Client
	defaultTTL time.Duration
}

func NewMetadataRedisCache(redis *redis.Client, ttl time.Duration) *MetadataRedisCache {
	return &MetadataRedisCache{
		redis:      redis,
		defaultTTL: ttl,
	}
}

func (c *MetadataRedisCache) Get(key string) (string, error) {
	return c.redis.Get(c.redis.Context(), KeyMetadataPrefix+key).Result()
}

func (c *MetadataRedisCache) Set(key string, value string) error {
	return c.redis.Set(c.redis.Context(), KeyMetadataPrefix+key, value, c.defaultTTL).Err()
}

func (c *MetadataRedisCache) Delete(key string) error {
	return c.redis.Del(c.redis.Context(), KeyMetadataPrefix+key).Err()
}
