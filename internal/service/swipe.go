package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/niko/citysnap-bot/internal/apperror"
	"github.com/niko/citysnap-bot/internal/model"
)

type SwipeService struct {
	swipes  SwipeRepository
	matches MatchRepository
	users   UserRepository
}

func NewSwipeService(swipes SwipeRepository, matches MatchRepository, users UserRepository) *SwipeService {
	return &SwipeService{swipes: swipes, matches: matches, users: users}
}

func (s *SwipeService) Swipe(ctx context.Context, swiperID, targetID uuid.UUID, swipeType string) (*model.Match, error) {
	log := slog.With("swiper", swiperID, "target", targetID, "type", swipeType)

	already, err := s.swipes.HasSwiped(ctx, swiperID, targetID)
	if err != nil {
		return nil, fmt.Errorf("check swiped: %w", err)
	}
	if already {
		return nil, apperror.ErrAlreadySwiped
	}

	swipe := &model.Swipe{
		SwiperID: swiperID,
		SwipedID: targetID,
		Type:     swipeType,
	}
	if err := s.swipes.Create(ctx, swipe); err != nil {
		return nil, fmt.Errorf("create swipe: %w", err)
	}

	if swipeType != "like" && swipeType != "superlike" {
		log.Info("swipe saved, no match check")
		return nil, nil
	}

	hasReverse, err := s.swipes.FindMatch(ctx, targetID, swiperID)
	if err != nil {
		return nil, fmt.Errorf("find match: %w", err)
	}
	if !hasReverse {
		log.Info("swipe saved, no reverse")
		return nil, nil
	}

	match := &model.Match{
		User1ID:  swiperID,
		User2ID:  targetID,
		IsActive: true,
	}
	if err := s.matches.Create(ctx, match); err != nil {
		return nil, fmt.Errorf("create match: %w", err)
	}

	log.Info("match created", "match_id", match.ID)
	return match, nil
}

func (s *SwipeService) GetCandidates(ctx context.Context, userID uuid.UUID, city string, limit int) ([]model.User, error) {
	return s.users.FindCandidates(ctx, userID, city, limit)
}

func (s *SwipeService) GetMatches(ctx context.Context, userID uuid.UUID) ([]model.Match, error) {
	return s.matches.FindByUserID(ctx, userID)
}
