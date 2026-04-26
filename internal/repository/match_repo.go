package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niko/citysnap-bot/internal/model"
)

type MatchRepo struct {
	pool *pgxpool.Pool
}

func NewMatchRepo(pool *pgxpool.Pool) *MatchRepo {
	return &MatchRepo{pool: pool}
}

func (r *MatchRepo) Create(ctx context.Context, match *model.Match) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO matches (user1_id, user2_id, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		match.User1ID, match.User2ID, match.IsActive,
	).Scan(&match.ID, &match.CreatedAt)
}

func (r *MatchRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Match, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user1_id, user2_id, is_active, created_at
		FROM matches
		WHERE (user1_id = $1 OR user2_id = $1) AND is_active = true
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("find matches: %w", err)
	}
	defer rows.Close()

	var matches []model.Match
	for rows.Next() {
		var m model.Match
		if err := rows.Scan(&m.ID, &m.User1ID, &m.User2ID, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}
