package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niko/citysnap-bot/internal/model"
)

type InterestRepo struct {
	pool *pgxpool.Pool
}

func NewInterestRepo(pool *pgxpool.Pool) *InterestRepo {
	return &InterestRepo{pool: pool}
}

func (r *InterestRepo) BatchCreate(ctx context.Context, interests []model.UserInterest) error {
	if len(interests) == 0 {
		return nil
	}

	rows := make([][]any, len(interests))
	for i, it := range interests {
		rows[i] = []any{it.UserID, it.Category, it.Value}
	}

	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"user_interests"},
		[]string{"user_id", "category", "value"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("batch insert interests: %w", err)
	}
	return nil
}

func (r *InterestRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserInterest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, category, value, created_at
		FROM user_interests
		WHERE user_id = $1
		ORDER BY category, value`, userID)
	if err != nil {
		return nil, fmt.Errorf("find interests: %w", err)
	}
	defer rows.Close()

	var interests []model.UserInterest
	for rows.Next() {
		var i model.UserInterest
		if err := rows.Scan(&i.ID, &i.UserID, &i.Category, &i.Value, &i.CreatedAt); err != nil {
			return nil, err
		}
		interests = append(interests, i)
	}
	return interests, rows.Err()
}

// FindCommon возвращает общие интересы двух пользователей.
func (r *InterestRepo) FindCommon(ctx context.Context, uid1, uid2 uuid.UUID) ([]model.UserInterest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.category, a.value
		FROM user_interests a
		INNER JOIN user_interests b
		  ON a.category = b.category AND a.value = b.value
		WHERE a.user_id = $1 AND b.user_id = $2
		ORDER BY a.category, a.value`, uid1, uid2)
	if err != nil {
		return nil, fmt.Errorf("find common: %w", err)
	}
	defer rows.Close()

	var common []model.UserInterest
	for rows.Next() {
		var i model.UserInterest
		if err := rows.Scan(&i.Category, &i.Value); err != nil {
			return nil, err
		}
		common = append(common, i)
	}
	return common, rows.Err()
}
