package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	FirstName       string     `json:"first_name" db:"first_name"`
	LastName        string     `json:"last_name" db:"last_name"`
	DefaultCurrency string     `json:"default_currency" db:"default_currency"`
	Timezone        string     `json:"timezone" db:"timezone"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time `json:"-" db:"deleted_at"`
}

type UserRegistration struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	FirstName       string `json:"first_name" binding:"required"`
	LastName        string `json:"last_name"`
	DefaultCurrency string `json:"default_currency"`
}

type UserLogin struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserUpdate struct {
	FirstName       *string `json:"first_name"`
	LastName        *string `json:"last_name"`
	DefaultCurrency *string `json:"defaul_currency"`
	Timezone        *string `json:"timezone"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	User         User   `json:"json"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
