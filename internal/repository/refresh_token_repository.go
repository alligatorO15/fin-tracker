package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	RevokedAt time.Time `json:"revoked_at"`
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	GetByToken(ctx context.Context, token string) (*RefreshToken, error)
	Revoke(cxt context.Context, token string) error
	RevokeAllForUser(cxt context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

type refreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) RefreshTokenRepository {
	return &refreshTokenRepository{pool: pool}
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (r *refreshTokenRepository) Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query,
		uuid.New(),
		userID,
		hashToken(token),
		expiresAt,
		time.Now(),
	)
	return err
}

func (r *refreshTokenRepository) GetByToken(ctx context.Context, token string) (*RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`

	var rt RefreshToken
	err := r.pool.QueryRow(ctx, query, hashToken(token)).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash,
		&rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r refreshTokenRepository) Revoke(ctx context.Context, token string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() where token_hash = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, hashToken(token))
	return err
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() where _id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

func (r *refreshTokenRepository) DeleteExpired(cxt context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW() OR revoked_at IS NOT NULL`
	_, err := r.pool.Exec(cxt, query)
	return err
}
