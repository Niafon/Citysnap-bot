package fsm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const fsmTTL = 30 * time.Minute

type Storage struct {
	rdb *redis.Client
}

func NewStorage(rdb *redis.Client) *Storage {
	return &Storage{rdb: rdb}
}

func (s *Storage) Get(ctx context.Context, tgID int64) (State, error) {
	val, err := s.rdb.Get(ctx, key(tgID)).Result()
	if errors.Is(err, redis.Nil) {
		return StateIdle, nil
	}
	if err != nil {
		return "", fmt.Errorf("fsm get: %w", err)
	}
	return State(val), nil
}

func (s *Storage) Set(ctx context.Context, tgID int64, state State) error {
	return s.rdb.Set(ctx, key(tgID), string(state), fsmTTL).Err()
}

func (s *Storage) Clear(ctx context.Context, tgID int64) error {
	return s.rdb.Del(ctx, key(tgID)).Err()
}

// SetData сохраняет произвольные данные FSM (например, file_id во время загрузки snap).
func (s *Storage) SetData(ctx context.Context, tgID int64, dataKey, value string) error {
	return s.rdb.Set(ctx, dataKeyName(tgID, dataKey), value, fsmTTL).Err()
}

func (s *Storage) GetData(ctx context.Context, tgID int64, dataKey string) (string, error) {
	val, err := s.rdb.Get(ctx, dataKeyName(tgID, dataKey)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

func key(tgID int64) string {
	return fmt.Sprintf("fsm:%d", tgID)
}

func dataKeyName(tgID int64, k string) string {
	return fmt.Sprintf("fsm:data:%d:%s", tgID, k)
}
