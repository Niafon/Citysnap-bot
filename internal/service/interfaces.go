package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/niko/citysnap-bot/internal/model"
)

// UserRepository — контракт для доступа к данным пользователей.
// Определяется здесь (в service), реализуется в repository.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByTelegramID(ctx context.Context, tgID int64) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	FindCandidates(ctx context.Context, userID uuid.UUID, city string, limit int) ([]model.User, error)
}

type InterestRepository interface {
	BatchCreate(ctx context.Context, interests []model.UserInterest) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserInterest, error)
	FindCommon(ctx context.Context, uid1, uid2 uuid.UUID) ([]model.UserInterest, error)
}

type SwipeRepository interface {
	Create(ctx context.Context, swipe *model.Swipe) error
	HasSwiped(ctx context.Context, swiperID, swipedID uuid.UUID) (bool, error)
	FindMatch(ctx context.Context, swiperID, swipedID uuid.UUID) (bool, error)
}

type MatchRepository interface {
	Create(ctx context.Context, match *model.Match) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Match, error)
}

type DailyPhotoRepository interface {
	Create(ctx context.Context, photo *model.DailyPhoto) error
	FindActiveByUser(ctx context.Context, userID uuid.UUID) (*model.DailyPhoto, error)
	FindActiveByCity(ctx context.Context, city string) ([]model.DailyPhoto, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]model.DailyPhoto, error)
	HideExpired(ctx context.Context) ([]model.DailyPhoto, error)
	IncrementViews(ctx context.Context, photoID uuid.UUID) error
}

type PhotoCacheStore interface {
	GetCityFeed(ctx context.Context, city string) ([]uuid.UUID, error)
	SetCityFeed(ctx context.Context, city string, ids []uuid.UUID) error
	DeleteCityFeed(ctx context.Context, city string) error
}
