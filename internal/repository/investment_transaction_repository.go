package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type InvestmentTransactionRepository interface {
	Create(ctx context.Context, tx *models.InvestmentTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.InvestmentTransaction, error)
	GetByPortfolioID(ctx context.Context, portfolioID uuid.UUID, limit, offset int) ([]models.InvestmentTransaction, error)
	GetBySecurityID(ctx context.Context, portfolioID, securityID uuid.UUID) ([]models.InvestmentTransaction, error)
	GetByDateRange(ctx context.Context, portfolioID uuid.UUID, startDate, endDate time.Time) ([]models.InvestmentTransaction, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetTotalDividends(ctx context.Context, portfolioID uuid.UUID, year int) (decimal.Decimal, error)
	GetTotalCommissions(ctx context.Context, portfolioID uuid.UUID, year int) (decimal.Decimal, error)
}

type investmentTransactionRepository struct {
	pool *pgxpool.Pool
}

func NewInvestmentTransactionRepository(pool *pgxpool.Pool) InvestmentTransactionRepository {
	return &investmentTransactionRepository{pool: pool}
}

func (r *investmentTransactionRepository) db(ctx context.Context) DBTX {
	return GetTxOrPool(ctx, r.pool)
}

func (r *investmentTransactionRepository) Create(ctx context.Context, tx *models.InvestmentTransaction) error {
	query := `
		INSERT INTO investment_transactions (id, portfolio_id, security_id, type, date, quantity, price, amount, commission, currency, exchange_rate, notes, broker_ref, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	tx.CreatedAt = time.Now()

	if tx.ExchangeRate.IsZero() {
		tx.ExchangeRate = decimal.NewFromInt(1)
	}

	_, err := r.db(ctx).Exec(ctx, query,
		tx.ID, tx.PortfolioID, tx.SecurityID, tx.Type, tx.Date,
		tx.Quantity, tx.Price, tx.Amount, tx.Commission, tx.Currency,
		tx.ExchangeRate, tx.Notes, tx.BrokerRef, tx.CreatedAt,
	)
	return err
}

func (r *investmentTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.InvestmentTransaction, error) {
	query := `
		SELECT it.id, it.portfolio_id, it.security_id, it.type, it.date, it.quantity, it.price, it.amount, it.commission, it.currency, it.exchange_rate, it.notes, it.broker_ref, it.created_at,
		       s.ticker, s.name, s.type as security_type
		FROM investment_transactions it
		JOIN securities s ON it.security_id = s.id
		WHERE it.id = $1
	`

	var tx models.InvestmentTransaction
	var security models.Security
	err := r.db(ctx).QueryRow(ctx, query, id).Scan(
		&tx.ID, &tx.PortfolioID, &tx.SecurityID, &tx.Type, &tx.Date,
		&tx.Quantity, &tx.Price, &tx.Amount, &tx.Commission, &tx.Currency,
		&tx.ExchangeRate, &tx.Notes, &tx.BrokerRef, &tx.CreatedAt,
		&security.Ticker, &security.Name, &security.Type,
	)
	if err != nil {
		return nil, err
	}

	tx.Security = &security

	return &tx, nil
}

func (r *investmentTransactionRepository) GetByPortfolioID(ctx context.Context, portfolioID uuid.UUID, limit, offset int) ([]models.InvestmentTransaction, error) {
	query := `
		SELECT it.id, it.portfolio_id, it.security_id, it.type, it.date, it.quantity, it.price, it.amount, it.commission, it.currency, it.exchange_rate, it.notes, it.broker_ref, it.created_at,
		       s.ticker, s.name, s.type as security_type
		FROM investment_transactions it
		JOIN securities s ON it.security_id = s.id
		WHERE it.portfolio_id = $1
		ORDER BY it.date DESC, it.created_at DESC
		LIMIT $2 OFFSET $3
	`

	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db(ctx).Query(ctx, query, portfolioID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

func (r *investmentTransactionRepository) GetBySecurityID(ctx context.Context, portfolioID, securityID uuid.UUID) ([]models.InvestmentTransaction, error) {
	query := `
		SELECT it.id, it.portfolio_id, it.security_id, it.type, it.date, it.quantity, it.price, it.amount, it.commission, it.currency, it.exchange_rate, it.notes, it.broker_ref, it.created_at,
		       s.ticker, s.name, s.type as security_type
		FROM investment_transactions it
		JOIN securities s ON it.security_id = s.id
		WHERE it.portfolio_id = $1 AND it.security_id = $2
		ORDER BY it.date DESC
	`

	rows, err := r.db(ctx).Query(ctx, query, portfolioID, securityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

func (r *investmentTransactionRepository) GetByDateRange(ctx context.Context, portfolioID uuid.UUID, startDate, endDate time.Time) ([]models.InvestmentTransaction, error) {
	query := `
		SELECT it.id, it.portfolio_id, it.security_id, it.type, it.date, it.quantity, it.price, it.amount, it.commission, it.currency, it.exchange_rate, it.notes, it.broker_ref, it.created_at,
		       s.ticker, s.name, s.type as security_type
		FROM investment_transactions it
		JOIN securities s ON it.security_id = s.id
		WHERE it.portfolio_id = $1 AND it.date >= $2 AND it.date <= $3
		ORDER BY it.date DESC
	`

	rows, err := r.db(ctx).Query(ctx, query, portfolioID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

func (r *investmentTransactionRepository) scanTransactions(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]models.InvestmentTransaction, error) {
	var transactions []models.InvestmentTransaction
	for rows.Next() {
		var tx models.InvestmentTransaction
		var security models.Security
		err := rows.Scan(
			&tx.ID, &tx.PortfolioID, &tx.SecurityID, &tx.Type, &tx.Date,
			&tx.Quantity, &tx.Price, &tx.Amount, &tx.Commission, &tx.Currency,
			&tx.ExchangeRate, &tx.Notes, &tx.BrokerRef, &tx.CreatedAt,
			&security.Ticker, &security.Name, &security.Type,
		)
		if err != nil {
			return nil, err
		}
		tx.Security = &security
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

func (r *investmentTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM investment_transactions WHERE id = $1`
	_, err := r.db(ctx).Exec(ctx, query, id)
	return err
}

func (r *investmentTransactionRepository) GetTotalDividends(ctx context.Context, portfolioID uuid.UUID, year int) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM investment_transactions
		WHERE portfolio_id = $1 AND type = 'dividend' AND EXTRACT(YEAR FROM date) = $2
	`

	var total decimal.Decimal
	err := r.db(ctx).QueryRow(ctx, query, portfolioID, year).Scan(&total)
	return total, err
}

func (r *investmentTransactionRepository) GetTotalCommissions(ctx context.Context, portfolioID uuid.UUID, year int) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(commission), 0)
		FROM investment_transactions
		WHERE portfolio_id = $1 AND EXTRACT(YEAR FROM date) = $2
	`

	var total decimal.Decimal
	err := r.db(ctx).QueryRow(ctx, query, portfolioID, year).Scan(&total)
	return total, err
}
