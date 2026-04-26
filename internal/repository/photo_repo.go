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

type DailyPhotoRepo struct {
	pool *pgxpool.Pool
}

func NewDailyPhotoRepo(pool *pgxpool.Pool) *DailyPhotoRepo {
	return &DailyPhotoRepo{pool: pool}
}

func (r *DailyPhotoRepo) Create(ctx context.Context, photo *model.DailyPhoto) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO daily_photos (user_id, city, photo_file_id, caption, expires_at, is_visible)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		photo.UserID, photo.City, photo.PhotoFileID, photo.Caption,
		photo.ExpiresAt, photo.IsVisible,
	).Scan(&photo.ID, &photo.CreatedAt)
}

func (r *DailyPhotoRepo) FindActiveByUser(ctx context.Context, userID uuid.UUID) (*model.DailyPhoto, error) {
	var p model.DailyPhoto
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, city, photo_file_id, caption, view_count,
		       created_at, expires_at, is_visible
		FROM daily_photos
		WHERE user_id = $1 AND is_visible = true AND expires_at > now()
		LIMIT 1`, userID,
	).Scan(&p.ID, &p.UserID, &p.City, &p.PhotoFileID, &p.Caption,
		&p.ViewCount, &p.CreatedAt, &p.ExpiresAt, &p.IsVisible)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find active by user: %w", err)
	}
	return &p, nil
}

func (r *DailyPhotoRepo) FindActiveByCity(ctx context.Context, city string) ([]model.DailyPhoto, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, city, photo_file_id, caption, view_count,
		       created_at, expires_at, is_visible
		FROM daily_photos
		WHERE city = $1 AND is_visible = true AND expires_at > now()
		ORDER BY created_at DESC`, city)
	if err != nil {
		return nil, fmt.Errorf("find by city: %w", err)
	}
	defer rows.Close()

	var photos []model.DailyPhoto
	for rows.Next() {
		var p model.DailyPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.City, &p.PhotoFileID, &p.Caption,
			&p.ViewCount, &p.CreatedAt, &p.ExpiresAt, &p.IsVisible); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

func (r *DailyPhotoRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]model.DailyPhoto, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, city, photo_file_id, caption, view_count,
		       created_at, expires_at, is_visible
		FROM daily_photos
		WHERE id = ANY($1) AND is_visible = true
		ORDER BY created_at DESC`, ids)
	if err != nil {
		return nil, fmt.Errorf("find by ids: %w", err)
	}
	defer rows.Close()

	var photos []model.DailyPhoto
	for rows.Next() {
		var p model.DailyPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.City, &p.PhotoFileID, &p.Caption,
			&p.ViewCount, &p.CreatedAt, &p.ExpiresAt, &p.IsVisible); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

// HideExpired переводит просроченные фото в is_visible=false и возвращает их.
func (r *DailyPhotoRepo) HideExpired(ctx context.Context) ([]model.DailyPhoto, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE daily_photos
		SET is_visible = false
		WHERE expires_at < now() AND is_visible = true
		RETURNING id, user_id, city, photo_file_id, caption, view_count,
		          created_at, expires_at, is_visible`)
	if err != nil {
		return nil, fmt.Errorf("hide expired: %w", err)
	}
	defer rows.Close()

	var photos []model.DailyPhoto
	for rows.Next() {
		var p model.DailyPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.City, &p.PhotoFileID, &p.Caption,
			&p.ViewCount, &p.CreatedAt, &p.ExpiresAt, &p.IsVisible); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

func (r *DailyPhotoRepo) IncrementViews(ctx context.Context, photoID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE daily_photos SET view_count = view_count + 1 WHERE id = $1`,
		photoID)
	return err
}
