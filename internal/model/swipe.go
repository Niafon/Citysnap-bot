package model

import (
	"time"

	"github.com/google/uuid"
)

type Swipe struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	SwiperID  uuid.UUID `json:"swiper_id"  db:"swiper_id"`
	SwipedID  uuid.UUID `json:"swiped_id"  db:"swiped_id"`
	Type      string    `json:"type"       db:"type"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Match struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	User1ID   uuid.UUID `json:"user1_id"   db:"user1_id"`
	User2ID   uuid.UUID `json:"user2_id"   db:"user2_id"`
	IsActive  bool      `json:"is_active"  db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
