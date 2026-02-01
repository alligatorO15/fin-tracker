package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/config"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// кастомные ошибки
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserExists         = errors.New("user with this email already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenRevoked       = errors.New("token revoked")
)

type AuthService interface {
	Register(ctx context.Context, input *models.UserRegistration) (*models.AuthResponse, error)
	Login(ctx context.Context, input *models.UserLogin) (*models.AuthResponse, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*models.AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	ValidateToken(tokenString string) (*Claims, error)
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

type authService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	config           *config.Config
}

func NewAuthService(userRepo repository.UserRepository, refreshTokenRepo repository.RefreshTokenRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		config:           cfg,
	}
}

func (s *authService) Register(ctx context.Context, input *models.UserRegistration) (*models.AuthResponse, error) {
	// смотрим существует ли юзер
	existing, _ := s.userRepo.GetByEmail(ctx, input.Email)
	if existing != nil {
		return nil, ErrUserExists
	}

	// хэшируем пароль,  bcrypt.DefaultCost = 10 компромисс между безопасностью и скоростью
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// устанавливаем дефолт валюту
	defaultCurrency := input.DefaultCurrency
	if defaultCurrency == "" {
		defaultCurrency = s.config.DefaultCurrency
	}

	user := &models.User{
		ID:              uuid.New(),
		Email:           input.Email,
		PasswordHash:    string(hashedPassword),
		FirstName:       input.FirstName,
		LastName:        input.LastName,
		DefaultCurrency: defaultCurrency,
		Timezone:        "Europe/Moscow",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// генерируем access и refresh токена на сессию
	return s.generateAuthResponse(ctx, user)
}

func (s *authService) Login(ctx context.Context, input *models.UserLogin) (*models.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Сравниваем пароль с его хэшем
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.generateAuthResponse(ctx, user)
}

func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	// ищем рефреш токен в бд
	storedToken, err := s.refreshTokenRepo.GetByToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if storedToken == nil {
		return nil, ErrInvalidToken
	}

	// берем юзера
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, err
	}

	// отзываем и удаляем старый рефреш токен
	if err := s.refreshTokenRepo.Revoke(ctx, refreshToken); err != nil {
		return nil, err
	}

	// создаем новую пару
	return s.generateAuthResponse(ctx, user)
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	return s.refreshTokenRepo.Revoke(ctx, refreshToken)
}

// отзываем все рефреш токены пользователя (выход из всех устройств)
func (s *authService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.RevokeAllForUser(ctx, userID)
}

// валидация jwt-токена
func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *authService) generateAuthResponse(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	accessToken, expiresAt, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt.Unix(),
		User:         *user,
	}, nil
}

func (s *authService) generateAccessToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.config.AccessTokenExpiration)

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "fintracker",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (s *authService) generateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(bytes)

	expiresAt := time.Now().Add(s.config.RefreshTokenExpiration)
	if err := s.refreshTokenRepo.Create(ctx, userID, token, expiresAt); err != nil {
		return "", err
	}

	return token, nil
}
