package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type HoldingRepository interface {
	Create(ctx context.Context, holding *models.Holding) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Holding, error)
	GetByPortfolioID(ctx context.Context, portfolioID uuid.UUID) ([]models.Holding, error)
	GetByPortfolioAndSecurity(ctx context.Context, portfolioID, securityID uuid.UUID) (*models.Holding, error)
	Update(ctx context.Context, id uuid.UUID, quantity, avgPrice, totalCost decimal.Decimal) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteIfZero(ctx context.Context, portfolioID, securityID uuid.UUID) error
}

type holdingRepository struct {
	pool *pgxpool.Pool
}

func NewHoldingRepository(pool *pgxpool.Pool) HoldingRepository {
	return &holdingRepository{pool: pool}
}

func (r *holdingRepository) Create(ctx context.Context, holding *models.Holding) error {
	query := `
		INSERT INTO holdings (id, portfolio_id, security_id, quantity, average_price, total_cost, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (portfolio_id, security_id) DO UPDATE SET
			quantity = holdings.quantity + EXCLUDED.quantity,
			total_cost = holdings.total_cost + EXCLUDED.total_cost,
			average_price = (holdings.total_cost + EXCLUDED.total_cost) / NULLIF(holdings.quantity + EXCLUDED.quantity, 0),
			updated_at = EXCLUDED.updated_at
	`

	if holding.ID == uuid.Nil {
		holding.ID = uuid.New()
	}
	now := time.Now()
	holding.CreatedAt = now
	holding.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		holding.ID, holding.PortfolioID, holding.SecurityID,
		holding.Quantity, holding.AveragePrice, holding.TotalCost,
		holding.CreatedAt, holding.UpdatedAt,
	)
	return err
}

func (r *holdingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Holding, error) {
	query := `
		SELECT h.id, h.portfolio_id, h.security_id, h.quantity, h.average_price, h.total_cost, h.created_at, h.updated_at,
		       s.ticker, s.name, s.type, s.exchange, s.currency, s.last_price
		FROM holdings h
		JOIN securities s ON h.security_id = s.id
		WHERE h.id = $1
	`

	var h models.Holding
	var security models.Security
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&h.ID, &h.PortfolioID, &h.SecurityID,
		&h.Quantity, &h.AveragePrice, &h.TotalCost,
		&h.CreatedAt, &h.UpdatedAt,
		&security.Ticker, &security.Name, &security.Type,
		&security.Exchange, &security.Currency, &security.LastPrice,
	)
	if err != nil {
		return nil, err
	}

	h.Security = &security
	h.CalculateValues()

	return &h, nil
}

func (r *holdingRepository) GetByPortfolioID(ctx context.Context, portfolioID uuid.UUID) ([]models.Holding, error) {
	query := `
		SELECT h.id, h.portfolio_id, h.security_id, h.quantity, h.average_price, h.total_cost, h.created_at, h.updated_at,
		       s.ticker, s.name, s.type, s.exchange, s.currency, s.last_price
		FROM holdings h
		JOIN securities s ON h.security_id = s.id
		WHERE h.portfolio_id = $1
		ORDER BY h.total_cost DESC
	`

	rows, err := r.pool.Query(ctx, query, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []models.Holding
	for rows.Next() {
		var h models.Holding
		var security models.Security
		err := rows.Scan(
			&h.ID, &h.PortfolioID, &h.SecurityID,
			&h.Quantity, &h.AveragePrice, &h.TotalCost,
			&h.CreatedAt, &h.UpdatedAt,
			&security.Ticker, &security.Name, &security.Type,
			&security.Exchange, &security.Currency, &security.LastPrice,
		)
		if err != nil {
			return nil, err
		}
		h.Security = &security
		h.CalculateValues()
		holdings = append(holdings, h)
	}
	return holdings, rows.Err()
}

func (r *holdingRepository) GetByPortfolioAndSecurity(ctx context.Context, portfolioID, securityID uuid.UUID) (*models.Holding, error) {
	query := `
		SELECT id, portfolio_id, security_id, quantity, average_price, total_cost, created_at, updated_at
		FROM holdings
		WHERE portfolio_id = $1 AND security_id = $2
	`

	var h models.Holding
	err := r.pool.QueryRow(ctx, query, portfolioID, securityID).Scan(
		&h.ID, &h.PortfolioID, &h.SecurityID,
		&h.Quantity, &h.AveragePrice, &h.TotalCost,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *holdingRepository) Update(ctx context.Context, id uuid.UUID, quantity, avgPrice, totalCost decimal.Decimal) error {
	query := `
		UPDATE holdings SET
			quantity = $2,
			average_price = $3,
			total_cost = $4,
			updated_at = $5
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, quantity, avgPrice, totalCost, time.Now())
	return err
}

func (r *holdingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM holdings WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *holdingRepository) DeleteIfZero(ctx context.Context, portfolioID, securityID uuid.UUID) error {
	query := `DELETE FROM holdings WHERE portfolio_id = $1 AND security_id = $2 AND quantity <= 0`
	_, err := r.pool.Exec(ctx, query, portfolioID, securityID)
	return err
}
