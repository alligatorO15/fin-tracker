package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type AccountRepository interface {
	Create(ctx context.Context, account *models.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Account, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Account, error)
	Update(ctx context.Context, id uuid.UUID, update *models.AccountUpdate) error
	UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetSummary(ctx context.Context, userID uuid.UUID) (*models.AccountSummary, error)
}

type accountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) AccountRepository {
	return &accountRepository{pool: pool}
}

func (r *accountRepository) Create(ctx context.Context, account *models.Account) error {
	query := `
		INSERT INTO accounts (id, user_id, name, type, currency, balance, initial_balance, icon, color, is_active, institution, account_number, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15 )
	`
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now
	account.Balance = account.InitialBalance
	account.IsActive = true

	_, err := r.pool.Exec(ctx, query,
		account.ID, account.UserID, account.Name, account.Type,
		account.Currency, account.Balance, account.InitialBalance,
		account.Icon, account.Color, account.IsActive,
		account.Institution, account.AccountNumber, account.Notes,
		account.CreatedAt, account.UpdatedAt,
	)
	return err
}

func (r *accountRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Account, error) {
	query := `
		SELECT (id, user_id, name, type, currency, balance, initial_balance, icon, color, is_active, institution, account_number, notes, created_at, updated_at)
		FROM accounts
		WHERE id = $1 and deleted_at IS NULL
	`
	var account models.Account
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&account.ID, &account.UserID, &account.Name, &account.Type,
		&account.Currency, &account.Balance, &account.InitialBalance,
		&account.Icon, &account.Color, &account.IsActive,
		&account.Institution, &account.AccountNumber, &account.Notes,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *accountRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Account, error) {
	query := `
		SELECT id, user_id, name, type, currency, balance, initial_balance, icon, color, is_active, institution, account_number, notes, created_at, updated_at
		FROM accounts
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var account models.Account
		err := rows.Scan(
			&account.ID, &account.UserID, &account.Name, &account.Type,
			&account.Currency, &account.Balance, &account.InitialBalance,
			&account.Icon, &account.Color, &account.IsActive,
			&account.Institution, &account.AccountNumber, &account.Notes,
			&account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (r *accountRepository) Update(ctx context.Context, id uuid.UUID, update *models.AccountUpdate) error {
	query := `
		UPDATE accounts SET
			name = COALESCE($2, name),
			icon = COALESCE($3, icon),
			color = COALESCE($4, color),
			is_active = COALESCE($5, is_active),
			institution = COALESCE($6, institution),
			account_number = COALESCE($7, account_number),
			notes = COALESCE($8, notes),
			updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query,
		id, update.Name, update.Icon, update.Color,
		update.IsActive, update.Institution,
		update.AccountNumber, update.Notes, time.Now(),
	)
	return err
}

func (r *accountRepository) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	query := `
		Update accounts SET
			balance = balance + $2
			updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}

func (r *accountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE accounts SET deleted_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}

func (r *accountRepository) GetSummary(ctx context.Context, userID uuid.UUID) (*models.AccountSummary, error) {
	accounts, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	summary := &models.AccountSummary{
		TotalBalance:      decimal.Zero,
		BalanceByCurrency: make(map[string]decimal.Decimal),
		AccountsByType:    make(map[models.AccountType]int),
		Accounts:          accounts,
	}

	for _, acc := range accounts {
		if acc.IsActive {
			summary.BalanceByCurrency[acc.Currency] = summary.BalanceByCurrency[acc.Currency].Add(acc.Balance)
			summary.AccountsByType[acc.Type]++
		}
	}
	//TotalBalance будет в сервисе. конвертация валюты это бизнес логика
	return summary, nil
}
