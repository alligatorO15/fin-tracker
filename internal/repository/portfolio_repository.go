package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PortfolioRepository interface {
	Create(ctx context.Context, portfolio *models.Portfolio) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Portfolio, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Portfolio, error)
	Update(ctx context.Context, id uuid.UUID, update *models.PortfolioUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type portfolioRepository struct {
	pool *pgxpool.Pool
}

func NewPortfolioRepository(pool *pgxpool.Pool) PortfolioRepository {
	return &portfolioRepository{pool: pool}
}

func (r *portfolioRepository) db(ctx context.Context) DBTX {
	return GetTxOrPool(ctx, r.pool)
}

func (r *portfolioRepository) Create(ctx context.Context, portfolio *models.Portfolio) error {
	query := `
		INSERT INTO portfolios (id, user_id, account_id, name, description, currency, broker_name, broker_account, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if portfolio.ID == uuid.Nil {
		portfolio.ID = uuid.New()
	}
	now := time.Now()
	portfolio.CreatedAt = now
	portfolio.UpdatedAt = now
	portfolio.IsActive = true

	_, err := r.db(ctx).Exec(ctx, query,
		portfolio.ID, portfolio.UserID, portfolio.AccountID, portfolio.Name,
		portfolio.Description, portfolio.Currency, portfolio.BrokerName,
		portfolio.BrokerAccount, portfolio.IsActive,
		portfolio.CreatedAt, portfolio.UpdatedAt,
	)
	return err
}

func (r *portfolioRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Portfolio, error) {
	query := `
		SELECT id, user_id, account_id, name, description, currency, broker_name, broker_account, is_active, created_at, updated_at
		FROM portfolios
		WHERE id = $1
	`

	var portfolio models.Portfolio
	err := r.db(ctx).QueryRow(ctx, query, id).Scan(
		&portfolio.ID, &portfolio.UserID, &portfolio.AccountID, &portfolio.Name,
		&portfolio.Description, &portfolio.Currency, &portfolio.BrokerName,
		&portfolio.BrokerAccount, &portfolio.IsActive,
		&portfolio.CreatedAt, &portfolio.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &portfolio, nil
}

func (r *portfolioRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Portfolio, error) {
	query := `
		SELECT id, user_id, account_id, name, description, currency, broker_name, broker_account, is_active, created_at, updated_at
		FROM portfolios
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db(ctx).Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var portfolios []models.Portfolio
	for rows.Next() {
		var portfolio models.Portfolio
		err := rows.Scan(
			&portfolio.ID, &portfolio.UserID, &portfolio.AccountID, &portfolio.Name,
			&portfolio.Description, &portfolio.Currency, &portfolio.BrokerName,
			&portfolio.BrokerAccount, &portfolio.IsActive,
			&portfolio.CreatedAt, &portfolio.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		portfolios = append(portfolios, portfolio)
	}
	return portfolios, rows.Err()
}

func (r *portfolioRepository) Update(ctx context.Context, id uuid.UUID, update *models.PortfolioUpdate) error {
	query := `
		UPDATE portfolios SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			broker_name = COALESCE($4, broker_name),
			broker_account = COALESCE($5, broker_account),
			is_active = COALESCE($6, is_active),
			updated_at = $7
		WHERE id = $1
	`

	_, err := r.db(ctx).Exec(ctx, query,
		id, update.Name, update.Description, update.BrokerName,
		update.BrokerAccount, update.IsActive, time.Now(),
	)
	return err
}

func (r *portfolioRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM portfolios WHERE id = $1`
	_, err := r.db(ctx).Exec(ctx, query, id)
	return err
}
