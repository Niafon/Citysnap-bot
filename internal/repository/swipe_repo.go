package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niko/citysnap-bot/internal/model"
)

type SwipeRepo struct {
	pool *pgxpool.Pool
}

func NewSwipeRepo(pool *pgxpool.Pool) *SwipeRepo {
	return &SwipeRepo{pool: pool}
}

func (r *SwipeRepo) Create(ctx context.Context, swipe *model.Swipe) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO swipes (swiper_id, swiped_id, type)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		swipe.SwiperID, swipe.SwipedID, swipe.Type,
	).Scan(&swipe.ID, &swipe.CreatedAt)
}

func (r *SwipeRepo) HasSwiped(ctx context.Context, swiperID, swipedID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM swipes
			WHERE swiper_id = $1 AND swiped_id = $2
		)`, swiperID, swipedID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has swiped: %w", err)
	}
	return exists, nil
}

// FindMatch проверяет, есть ли встречный лайк/суперлайк от swiperID к swipedID.
func (r *SwipeRepo) FindMatch(ctx context.Context, swiperID, swipedID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM swipes
			WHERE swiper_id = $1 AND swiped_id = $2
			  AND type IN ('like', 'superlike')
		)`, swiperID, swipedID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("find match: %w", err)
	}
	return exists, nil
}
