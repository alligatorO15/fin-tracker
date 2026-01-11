package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type SecurityRepository interface {
	Create(ctx context.Context, security *models.Security) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Security, error)
	GetByTicker(ctx context.Context, ticker string, exchange models.Exchange) (*models.Security, error)
	GetByExchange(ctx context.Context, exchange models.Exchange) ([]models.Security, error)
	Search(ctx context.Context, query string, limit int) ([]models.Security, error)
	Update(ctx context.Context, id uuid.UUID, security *models.Security) error
	UpdatePrice(ctx context.Context, id uuid.UUID, price decimal.Decimal, change decimal.Decimal, changePercent decimal.Decimal, volume int64) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type securityRepository struct {
	pool *pgxpool.Pool
}

func NewSecurityRepository(pool *pgxpool.Pool) SecurityRepository {
	return &securityRepository{pool: pool}
}

func (r *securityRepository) Create(ctx context.Context, security *models.Security) error {
	query := `
		INSERT INTO securities (id, ticker, isin, name, short_name, type, exchange, currency, country, sector, industry, lot_size, min_price_increment, is_active, face_value, coupon_rate, maturity_date, coupon_freq, expense_ratio, last_price, price_change, price_change_percent, volume, updated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		ON CONFLICT (ticker, exchange) DO UPDATE SET
			name = EXCLUDED.name,
			short_name = EXCLUDED.short_name,
			sector = EXCLUDED.sector,
			industry = EXCLUDED.industry,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`

	if security.ID == uuid.Nil {
		security.ID = uuid.New()
	}
	now := time.Now()
	security.CreatedAt = now
	security.UpdatedAt = now

	if security.LotSize == 0 {
		security.LotSize = 1
	}

	_, err := r.pool.Exec(ctx, query,
		security.ID, security.Ticker, security.ISIN, security.Name, security.ShortName,
		security.Type, security.Exchange, security.Currency, security.Country,
		security.Sector, security.Industry, security.LotSize, security.MinPriceIncrement,
		security.IsActive, security.FaceValue, security.CouponRate, security.MaturityDate,
		security.CouponFreq, security.ExpenseRatio, security.LastPrice, security.PriceChange,
		security.PriceChangePercent, security.Volume, security.UpdatedAt, security.CreatedAt,
	)
	return err
}

func (r *securityRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Security, error) {
	query := `
		SELECT id, ticker, isin, name, short_name, type, exchange, currency, country, sector, industry, lot_size, min_price_increment, is_active, face_value, coupon_rate, maturity_date, coupon_freq, expense_ratio, last_price, price_change, price_change_percent, volume, updated_at, created_at
		FROM securities
		WHERE id = $1
	`

	var s models.Security
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Ticker, &s.ISIN, &s.Name, &s.ShortName,
		&s.Type, &s.Exchange, &s.Currency, &s.Country,
		&s.Sector, &s.Industry, &s.LotSize, &s.MinPriceIncrement,
		&s.IsActive, &s.FaceValue, &s.CouponRate, &s.MaturityDate,
		&s.CouponFreq, &s.ExpenseRatio, &s.LastPrice, &s.PriceChange,
		&s.PriceChangePercent, &s.Volume, &s.UpdatedAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *securityRepository) GetByTicker(ctx context.Context, ticker string, exchange models.Exchange) (*models.Security, error) {
	query := `
		SELECT id, ticker, isin, name, short_name, type, exchange, currency, country, sector, industry, lot_size, min_price_increment, is_active, face_value, coupon_rate, maturity_date, coupon_freq, expense_ratio, last_price, price_change, price_change_percent, volume, updated_at, created_at
		FROM securities
		WHERE ticker = $1 AND exchange = $2
	`

	var s models.Security
	err := r.pool.QueryRow(ctx, query, ticker, exchange).Scan(
		&s.ID, &s.Ticker, &s.ISIN, &s.Name, &s.ShortName,
		&s.Type, &s.Exchange, &s.Currency, &s.Country,
		&s.Sector, &s.Industry, &s.LotSize, &s.MinPriceIncrement,
		&s.IsActive, &s.FaceValue, &s.CouponRate, &s.MaturityDate,
		&s.CouponFreq, &s.ExpenseRatio, &s.LastPrice, &s.PriceChange,
		&s.PriceChangePercent, &s.Volume, &s.UpdatedAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *securityRepository) GetByExchange(ctx context.Context, exchange models.Exchange) ([]models.Security, error) {
	query := `
		SELECT id, ticker, isin, name, short_name, type, exchange, currency, country, sector, industry, lot_size, min_price_increment, is_active, face_value, coupon_rate, maturity_date, coupon_freq, expense_ratio, last_price, price_change, price_change_percent, volume, updated_at, created_at
		FROM securities
		WHERE exchange = $1 AND is_active = true
		ORDER BY ticker
	`

	rows, err := r.pool.Query(ctx, query, exchange)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var securities []models.Security
	for rows.Next() {
		var s models.Security
		err := rows.Scan(
			&s.ID, &s.Ticker, &s.ISIN, &s.Name, &s.ShortName,
			&s.Type, &s.Exchange, &s.Currency, &s.Country,
			&s.Sector, &s.Industry, &s.LotSize, &s.MinPriceIncrement,
			&s.IsActive, &s.FaceValue, &s.CouponRate, &s.MaturityDate,
			&s.CouponFreq, &s.ExpenseRatio, &s.LastPrice, &s.PriceChange,
			&s.PriceChangePercent, &s.Volume, &s.UpdatedAt, &s.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		securities = append(securities, s)
	}
	return securities, rows.Err()
}

func (r *securityRepository) Search(ctx context.Context, query string, limit int) ([]models.Security, error) {
	sqlQuery := `
		SELECT id, ticker, isin, name, short_name, type, exchange, currency, country, sector, industry, lot_size, min_price_increment, is_active, face_value, coupon_rate, maturity_date, coupon_freq, expense_ratio, last_price, price_change, price_change_percent, volume, updated_at, created_at
		FROM securities
		WHERE (ticker ILIKE $1 OR name ILIKE $1 OR short_name ILIKE $1 OR isin ILIKE $1) AND is_active = true
		ORDER BY ticker
		LIMIT $2
	`

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx, sqlQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var securities []models.Security
	for rows.Next() {
		var s models.Security
		err := rows.Scan(
			&s.ID, &s.Ticker, &s.ISIN, &s.Name, &s.ShortName,
			&s.Type, &s.Exchange, &s.Currency, &s.Country,
			&s.Sector, &s.Industry, &s.LotSize, &s.MinPriceIncrement,
			&s.IsActive, &s.FaceValue, &s.CouponRate, &s.MaturityDate,
			&s.CouponFreq, &s.ExpenseRatio, &s.LastPrice, &s.PriceChange,
			&s.PriceChangePercent, &s.Volume, &s.UpdatedAt, &s.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		securities = append(securities, s)
	}
	return securities, rows.Err()
}

func (r *securityRepository) Update(ctx context.Context, id uuid.UUID, security *models.Security) error {
	query := `
		UPDATE securities SET
			name = $2,
			short_name = $3,
			sector = $4,
			industry = $5,
			is_active = $6,
			face_value = $7,
			coupon_rate = $8,
			maturity_date = $9,
			coupon_freq = $10,
			expense_ratio = $11,
			updated_at = $12
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		id, security.Name, security.ShortName, security.Sector, security.Industry,
		security.IsActive, security.FaceValue, security.CouponRate, security.MaturityDate,
		security.CouponFreq, security.ExpenseRatio, time.Now(),
	)
	return err
}

func (r *securityRepository) UpdatePrice(ctx context.Context, id uuid.UUID, price decimal.Decimal, change decimal.Decimal, changePercent decimal.Decimal, volume int64) error {
	query := `
		UPDATE securities SET
			last_price = $2,
			price_change = $3,
			price_change_percent = $4,
			volume = $5,
			updated_at = $6
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, price, change, changePercent, volume, time.Now())
	return err
}

func (r *securityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE securities SET is_active = false WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
