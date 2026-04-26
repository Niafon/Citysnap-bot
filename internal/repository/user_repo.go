package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niko/citysnap-bot/internal/model"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO users (telegram_id, nickname, age, description, photo_file_id, city, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		user.TelegramID, user.Nickname, user.Age, user.Description,
		user.PhotoFileID, user.City, user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepo) FindByTelegramID(ctx context.Context, tgID int64) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, telegram_id, nickname, age, description,
		       photo_file_id, city, is_active, created_at, updated_at
		FROM users WHERE telegram_id = $1`, tgID,
	).Scan(&u.ID, &u.TelegramID, &u.Nickname, &u.Age, &u.Description,
		&u.PhotoFileID, &u.City, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by tg_id: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, telegram_id, nickname, age, description,
		       photo_file_id, city, is_active, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.TelegramID, &u.Nickname, &u.Age, &u.Description,
		&u.PhotoFileID, &u.City, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET nickname=$1, age=$2, description=$3,
		       photo_file_id=$4, city=$5, is_active=$6, updated_at=now()
		WHERE id = $7`,
		user.Nickname, user.Age, user.Description,
		user.PhotoFileID, user.City, user.IsActive, user.ID)
	return err
}

func (r *UserRepo) FindCandidates(ctx context.Context, userID uuid.UUID, city string, limit int) ([]model.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, telegram_id, nickname, age, description, photo_file_id, city
		FROM users
		WHERE city = $1 AND id != $2 AND is_active = true
		  AND id NOT IN (SELECT swiped_id FROM swipes WHERE swiper_id = $2)
		ORDER BY RANDOM() LIMIT $3`, city, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("find candidates: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.Nickname, &u.Age,
			&u.Description, &u.PhotoFileID, &u.City); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
