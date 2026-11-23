package models

import "time"

// User представляет пользователя в системе
type User struct {
	ID           string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLogin    *time.Time
}

// RefreshToken представляет refresh токен
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	Revoked   bool
	RevokedAt *time.Time
	IPAddress string
	UserAgent string
}

// UserProfile представляет публичный профиль пользователя
type UserProfile struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TokenPair представляет пару токенов (access и refresh)
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}
