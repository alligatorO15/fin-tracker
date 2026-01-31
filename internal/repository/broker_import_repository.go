package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BrokerImportRepository interface {
	Create(ctx context.Context, imp *models.BrokerStatementImport) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.BrokerStatementImport, error)
	GetByPortfolioID(cxt context.Context, portfolioID uuid.UUID) ([]models.BrokerStatementImport, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMessage string, transactionImported int) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type brokerImportRepository struct {
	pool *pgxpool.Pool
}

func NewBrokerImportRepository(pool *pgxpool.Pool) BrokerImportRepository {
	return &brokerImportRepository{pool: pool}
}

func (r *brokerImportRepository) Create(ctx context.Context, imp *models.BrokerStatementImport) error {
	query := `
		INSERT INTO broker_imports (
			id, portfolio_id, broker_type, file_name, import_date,
			period_start, period_end, status, error_message, transactions_imported, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	imp.ID = uuid.New()
	imp.ImportDate = time.Now()
	imp.CreatedAt = time.Now()
	if imp.Status == "" {
		imp.Status = "pending"
	}

	_, err := r.pool.Exec(ctx, query,
		imp.ID,
		imp.PortfolioID,
		imp.BrokerType,
		imp.FileName,
		imp.ImportDate,
		imp.PeriodStart,
		imp.PeriodEnd,
		imp.Status,
		imp.ErrorMessage,
		imp.TransactionsImported,
		imp.CreatedAt,
	)
	return err
}

func (r *brokerImportRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.BrokerStatementImport, error) {
	query := `
		SELECT id, portfolio_id, broker_type, file_name, import_date,
			   period_start, period_end, status, error_message, transactions_imported, created_at
		FROM broker_imports
		WHERE id = $1
	`

	var imp models.BrokerStatementImport
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&imp.ID,
		&imp.PortfolioID,
		&imp.BrokerType,
		&imp.FileName,
		&imp.ImportDate,
		&imp.PeriodStart,
		&imp.PeriodEnd,
		&imp.Status,
		&imp.ErrorMessage,
		&imp.TransactionsImported,
		&imp.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &imp, nil
}

func (r *brokerImportRepository) GetByPortfolioID(ctx context.Context, portfolioID uuid.UUID) ([]models.BrokerStatementImport, error) {
	query := `
		SELECT id, portfolio_id, broker_type, file_name, import_date,
			   period_start, period_end, status, error_message, transactions_imported, created_at
		FROM broker_imports
		WHERE portfolio_id = $1
		ORDER BY import_date DESC
	`

	rows, err := r.pool.Query(ctx, query, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imports []models.BrokerStatementImport
	for rows.Next() {
		var imp models.BrokerStatementImport
		err := rows.Scan(
			&imp.ID,
			&imp.PortfolioID,
			&imp.BrokerType,
			&imp.FileName,
			&imp.ImportDate,
			&imp.PeriodStart,
			&imp.PeriodEnd,
			&imp.Status,
			&imp.ErrorMessage,
			&imp.TransactionsImported,
			&imp.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		imports = append(imports, imp)
	}

	return imports, nil
}

func (r *brokerImportRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMessage string, transactionsImported int) error {
	query := `
		UPDATE broker_imports
		SET status = $2, error_message = $3, transactions_imported = $4
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, status, errorMessage, transactionsImported)
	return err
}

func (r *brokerImportRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM broker_imports WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
