package model

import (
	"time"

	"github.com/google/uuid"
)

type DailyPhoto struct {
	ID          uuid.UUID `json:"id"            db:"id"`
	UserID      uuid.UUID `json:"user_id"       db:"user_id"`
	City        string    `json:"city"          db:"city"`
	PhotoFileID string    `json:"photo_file_id" db:"photo_file_id"`
	Caption     string    `json:"caption"       db:"caption"`
	ViewCount   int       `json:"view_count"    db:"view_count"`
	CreatedAt   time.Time `json:"created_at"    db:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"    db:"expires_at"`
	IsVisible   bool      `json:"is_visible"    db:"is_visible"`
}

func (p *DailyPhoto) TimeLeft() time.Duration {
	left := time.Until(p.ExpiresAt)
	if left < 0 {
		return 0
	}
	return left
}

func (p *DailyPhoto) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

type PhotoReaction struct {
	ID           uuid.UUID `json:"id"            db:"id"`
	PhotoID      uuid.UUID `json:"photo_id"      db:"photo_id"`
	UserID       uuid.UUID `json:"user_id"       db:"user_id"`
	ReactionType string    `json:"reaction_type" db:"reaction_type"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
}
