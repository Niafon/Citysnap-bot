package apperror

import "errors"

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrAlreadySwiped = errors.New("already swiped this user")
	ErrSnapActive    = errors.New("active snap already exists")
	ErrPhotoExpired  = errors.New("photo has expired")
	ErrRateLimit     = errors.New("daily swipe limit exceeded")
	ErrInvalidInput  = errors.New("invalid input")
)
