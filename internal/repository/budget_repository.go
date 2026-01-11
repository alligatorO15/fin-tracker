package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BudgetRepository interface {
	Create(ctx context.Context, budget *models.Budget) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Budget, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, activeOnly bool) ([]models.Budget, error)
	GetByCategory(ctx context.Context, userID uuid.UUID, categoryID uuid.UUID) ([]models.Budget, error)
	Update(ctx context.Context, id uuid.UUID, update *models.BudgetUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type budgetRepository struct {
	pool *pgxpool.Pool
}

func NewBudgetRepository(pool *pgxpool.Pool) BudgetRepository {
	return &budgetRepository{pool: pool}
}

func (r *budgetRepository) Create(ctx context.Context, budget *models.Budget) error {
	query := `
		INSERT INTO budgets (id, user_id, category_id, name, amount, currency, period, start_date, end_date, is_active, alert_percent, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	if budget.ID == uuid.Nil {
		budget.ID = uuid.New()
	}
	now := time.Now()
	budget.CreatedAt = now
	budget.UpdatedAt = now
	budget.IsActive = true

	if budget.AlertPercent == 0 {
		budget.AlertPercent = 80
	}

	_, err := r.pool.Exec(ctx, query,
		budget.ID, budget.UserID, budget.CategoryID, budget.Name,
		budget.Amount, budget.Currency, budget.Period,
		budget.StartDate, budget.EndDate, budget.IsActive,
		budget.AlertPercent, budget.Notes,
		budget.CreatedAt, budget.UpdatedAt,
	)
	return err
}

func (r *budgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Budget, error) {
	query := `
		SELECT id, user_id, category_id, name, amount, currency, period, start_date, end_date, is_active, alert_percent, notes, created_at, updated_at
		FROM budgets
		WHERE id = $1
	`

	var budget models.Budget
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Name,
		&budget.Amount, &budget.Currency, &budget.Period,
		&budget.StartDate, &budget.EndDate, &budget.IsActive,
		&budget.AlertPercent, &budget.Notes,
		&budget.CreatedAt, &budget.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

func (r *budgetRepository) GetByUserID(ctx context.Context, userID uuid.UUID, activeOnly bool) ([]models.Budget, error) {
	query := `
		SELECT id, user_id, category_id, name, amount, currency, period, start_date, end_date, is_active, alert_percent, notes, created_at, updated_at
		FROM budgets
		WHERE user_id = $1
	`

	if activeOnly {
		query += " AND is_active = true"
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var budget models.Budget
		err := rows.Scan(
			&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Name,
			&budget.Amount, &budget.Currency, &budget.Period,
			&budget.StartDate, &budget.EndDate, &budget.IsActive,
			&budget.AlertPercent, &budget.Notes,
			&budget.CreatedAt, &budget.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, budget)
	}
	return budgets, rows.Err()
}

func (r *budgetRepository) GetByCategory(ctx context.Context, userID uuid.UUID, categoryID uuid.UUID) ([]models.Budget, error) {
	query := `
		SELECT id, user_id, category_id, name, amount, currency, period, start_date, end_date, is_active, alert_percent, notes, created_at, updated_at
		FROM budgets
		WHERE user_id = $1 AND category_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var budget models.Budget
		err := rows.Scan(
			&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Name,
			&budget.Amount, &budget.Currency, &budget.Period,
			&budget.StartDate, &budget.EndDate, &budget.IsActive,
			&budget.AlertPercent, &budget.Notes,
			&budget.CreatedAt, &budget.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, budget)
	}
	return budgets, rows.Err()
}

func (r *budgetRepository) Update(ctx context.Context, id uuid.UUID, update *models.BudgetUpdate) error {
	query := `
		UPDATE budgets SET
			category_id = COALESCE($2, category_id),
			name = COALESCE($3, name),
			amount = COALESCE($4, amount),
			period = COALESCE($5, period),
			start_date = COALESCE($6, start_date),
			end_date = COALESCE($7, end_date),
			is_active = COALESCE($8, is_active),
			alert_percent = COALESCE($9, alert_percent),
			notes = COALESCE($10, notes),
			updated_at = $11
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		id, update.CategoryID, update.Name, update.Amount,
		update.Period, update.StartDate, update.EndDate,
		update.IsActive, update.AlertPercent, update.Notes,
		time.Now(),
	)
	return err
}

func (r *budgetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM budgets WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
