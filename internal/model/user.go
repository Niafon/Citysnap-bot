package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"            db:"id"`
	TelegramID  int64     `json:"telegram_id"   db:"telegram_id"`
	Nickname    string    `json:"nickname"       db:"nickname"`
	Age         int       `json:"age"            db:"age"`
	Description string    `json:"description"    db:"description"`
	PhotoFileID string    `json:"photo_file_id"  db:"photo_file_id"`
	City        string    `json:"city"           db:"city"`
	IsActive    bool      `json:"is_active"      db:"is_active"`
	CreatedAt   time.Time `json:"created_at"     db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"     db:"updated_at"`
}

func (u *User) NormalizeCity() {
	u.City = strings.ToLower(strings.TrimSpace(u.City))
}

func (u *User) IsComplete() bool {
	return u.Nickname != "" &&
		u.Age >= 18 &&
		u.City != "" &&
		u.PhotoFileID != ""
}

type UserInterest struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	UserID    uuid.UUID `json:"user_id"    db:"user_id"`
	Category  string    `json:"category"   db:"category"`
	Value     string    `json:"value"      db:"value"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
