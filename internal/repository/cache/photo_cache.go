package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PhotoCache struct {
	rdb *redis.Client
}

func NewPhotoCache(rdb *redis.Client) *PhotoCache {
	return &PhotoCache{rdb: rdb}
}

func (c *PhotoCache) GetCityFeed(ctx context.Context, city string) ([]uuid.UUID, error) {
	data, err := c.rdb.Get(ctx, "feed:"+city).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get feed: %w", err)
	}

	var ids []uuid.UUID
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("unmarshal feed: %w", err)
	}
	return ids, nil
}

func (c *PhotoCache) SetCityFeed(ctx context.Context, city string, ids []uuid.UUID) error {
	data, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshal feed: %w", err)
	}
	return c.rdb.Set(ctx, "feed:"+city, data, 5*time.Minute).Err()
}

func (c *PhotoCache) DeleteCityFeed(ctx context.Context, city string) error {
	return c.rdb.Del(ctx, "feed:"+city).Err()
}
