package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/niko/citysnap-bot/internal/apperror"
	"github.com/niko/citysnap-bot/internal/model"
)

type DailyPhotoService struct {
	repo  DailyPhotoRepository
	cache PhotoCacheStore
}

func NewDailyPhotoService(repo DailyPhotoRepository, cache PhotoCacheStore) *DailyPhotoService {
	return &DailyPhotoService{repo: repo, cache: cache}
}

func (s *DailyPhotoService) Create(ctx context.Context, userID uuid.UUID, city, fileID, caption string) (*model.DailyPhoto, error) {
	existing, err := s.repo.FindActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check active: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrSnapActive
	}

	photo := &model.DailyPhoto{
		UserID:      userID,
		City:        city,
		PhotoFileID: fileID,
		Caption:     caption,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		IsVisible:   true,
	}
	if err := s.repo.Create(ctx, photo); err != nil {
		return nil, fmt.Errorf("create photo: %w", err)
	}

	_ = s.cache.DeleteCityFeed(ctx, city)

	slog.Info("snap created", "user_id", userID, "photo_id", photo.ID, "expires_at", photo.ExpiresAt)
	return photo, nil
}

func (s *DailyPhotoService) GetCityFeed(ctx context.Context, city string, excludeUID uuid.UUID) ([]model.DailyPhoto, error) {
	ids, err := s.cache.GetCityFeed(ctx, city)
	if err != nil {
		slog.Warn("cache miss", "city", city, "error", err)
	}

	if ids != nil {
		photos, err := s.repo.FindByIDs(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("find by ids: %w", err)
		}
		return filterOut(photos, excludeUID), nil
	}

	photos, err := s.repo.FindActiveByCity(ctx, city)
	if err != nil {
		return nil, fmt.Errorf("find active: %w", err)
	}

	cacheIDs := make([]uuid.UUID, len(photos))
	for i, p := range photos {
		cacheIDs[i] = p.ID
	}
	_ = s.cache.SetCityFeed(ctx, city, cacheIDs)

	return filterOut(photos, excludeUID), nil
}

func (s *DailyPhotoService) GetActiveByUser(ctx context.Context, userID uuid.UUID) (*model.DailyPhoto, error) {
	return s.repo.FindActiveByUser(ctx, userID)
}

func (s *DailyPhotoService) IncrementViews(ctx context.Context, photoID uuid.UUID) error {
	return s.repo.IncrementViews(ctx, photoID)
}

func (s *DailyPhotoService) StartCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	slog.Info("cleanup worker started", "interval", "5m")

	for {
		select {
		case <-ticker.C:
			expired, err := s.repo.HideExpired(ctx)
			if err != nil {
				slog.Error("cleanup failed", "error", err)
				continue
			}
			if len(expired) > 0 {
				slog.Info("photos expired", "count", len(expired))
				cities := uniqueCities(expired)
				for _, city := range cities {
					_ = s.cache.DeleteCityFeed(ctx, city)
				}
			}
		case <-ctx.Done():
			slog.Info("cleanup worker stopped")
			return
		}
	}
}

func filterOut(photos []model.DailyPhoto, excludeUID uuid.UUID) []model.DailyPhoto {
	result := make([]model.DailyPhoto, 0, len(photos))
	for _, p := range photos {
		if p.UserID != excludeUID {
			result = append(result, p)
		}
	}
	return result
}

func uniqueCities(photos []model.DailyPhoto) []string {
	seen := make(map[string]struct{})
	var cities []string
	for _, p := range photos {
		if _, ok := seen[p.City]; !ok {
			seen[p.City] = struct{}{}
			cities = append(cities, p.City)
		}
	}
	return cities
}
