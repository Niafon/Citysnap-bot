package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/niko/citysnap-bot/internal/model"
)

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(ctx context.Context, tgID int64, nickname string) (*model.User, error) {
	existing, err := s.repo.FindByTelegramID(ctx, tgID)
	if err != nil {
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	user := &model.User{
		TelegramID: tgID,
		Nickname:   nickname,
		IsActive:   true,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	slog.Info("user registered", "telegram_id", tgID, "nickname", nickname)
	return user, nil
}

func (s *UserService) GetByTelegramID(ctx context.Context, tgID int64) (*model.User, error) {
	return s.repo.FindByTelegramID(ctx, tgID)
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) Update(ctx context.Context, user *model.User) error {
	return s.repo.Update(ctx, user)
}
