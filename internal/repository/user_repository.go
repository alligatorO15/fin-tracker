package repository

import (
	"context"
	"errors"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, id uuid.UUID, update *models.UserUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// userRepository - ПРИВАТНАЯ структура, реализующая интерфейс UserRepository
type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository - ПУБЛИЧНАЯ фабричная функция-конструктор
// Создает экземпляр userRepository и возвращает его как UserRepository интерфейс
// Это позволяет скрыть детали реализации
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, default_currency, timezone, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	now := time.Now()
	user.CreatedAt = now
	user.CreatedAt = now

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash,
		user.FirstName, user.LastName,
		user.DefaultCurrency, user.Timezone,
		user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, default_currency, timezone, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName,
		&user.DefaultCurrency, &user.Timezone,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// TODO:разделил так как потом можно добавить логирование для ноуроус
			return nil, err
		}
		return nil, err
	}
	return &user, nil

}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, default_currency, timezone, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	var user models.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName,
		&user.DefaultCurrency, &user.Timezone,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// TODO:разделил так как потом можно добавить логирование для ноуроус
			return nil, err
		}
		return nil, err
	}
	return &user, nil

}

func (r *userRepository) Update(ctx context.Context, id uuid.UUID, update *models.UserUpdate) error {
	query := `
		UPDATE users SET
			first_name = COALESCE($2, first_name),
			last_name = COALESCE($3, last_name),
			default_currency = COALESCE($4, default_currency),
			timezone = COALESCE($5, timezone),
			updated_at = $6
		WHERE id = $1 and deleted_at IS NOT NULL
	`

	_, err := r.pool.Exec(ctx, query, id, update.FirstName, update.LastName, update.DefaultCurrency,
		update.Timezone, time.Now(),
	)
	return err
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}
