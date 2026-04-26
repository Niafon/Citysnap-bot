package model

import (
	"testing"
	"time"
)

func TestUser_IsComplete(t *testing.T) {
	tests := []struct {
		name string
		user User
		want bool
	}{
		{
			name: "all fields filled",
			user: User{Nickname: "alex", Age: 25, City: "москва", PhotoFileID: "abc123"},
			want: true,
		},
		{
			name: "missing nickname",
			user: User{Age: 25, City: "москва", PhotoFileID: "abc123"},
			want: false,
		},
		{
			name: "age too young",
			user: User{Nickname: "alex", Age: 17, City: "москва", PhotoFileID: "abc123"},
			want: false,
		},
		{
			name: "age zero",
			user: User{Nickname: "alex", Age: 0, City: "москва", PhotoFileID: "abc123"},
			want: false,
		},
		{
			name: "missing city",
			user: User{Nickname: "alex", Age: 25, PhotoFileID: "abc123"},
			want: false,
		},
		{
			name: "missing photo",
			user: User{Nickname: "alex", Age: 25, City: "москва"},
			want: false,
		},
		{
			name: "completely empty",
			user: User{},
			want: false,
		},
		{
			name: "age exactly 18",
			user: User{Nickname: "alex", Age: 18, City: "москва", PhotoFileID: "abc123"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.IsComplete()
			if got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_NormalizeCity(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "uppercase", input: "МОСКВА", want: "москва"},
		{name: "spaces", input: "  москва  ", want: "москва"},
		{name: "mixed", input: "  Санкт-Петербург ", want: "санкт-петербург"},
		{name: "already clean", input: "москва", want: "москва"},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{City: tt.input}
			u.NormalizeCity()
			if u.City != tt.want {
				t.Errorf("NormalizeCity() = %q, want %q", u.City, tt.want)
			}
		})
	}
}

func TestDailyPhoto_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{name: "expired", expiresAt: time.Now().Add(-1 * time.Hour), want: true},
		{name: "not expired", expiresAt: time.Now().Add(1 * time.Hour), want: false},
		{name: "just expired", expiresAt: time.Now().Add(-1 * time.Second), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &DailyPhoto{ExpiresAt: tt.expiresAt}
			if got := p.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDailyPhoto_TimeLeft(t *testing.T) {
	p := &DailyPhoto{ExpiresAt: time.Now().Add(2 * time.Hour)}
	left := p.TimeLeft()
	if left < 1*time.Hour || left > 3*time.Hour {
		t.Errorf("TimeLeft() = %v, expected ~2h", left)
	}

	expired := &DailyPhoto{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	if expired.TimeLeft() != 0 {
		t.Errorf("TimeLeft() for expired should be 0, got %v", expired.TimeLeft())
	}
}
