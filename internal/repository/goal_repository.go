package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type GoalRepository interface {
	Create(ctx context.Context, goal *models.Goal) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Goal, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, status *models.GoalStatus) ([]models.Goal, error)
	Update(ctx context.Context, id uuid.UUID, update *models.GoalUpdate) error
	UpdateAmount(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddContribution(ctx context.Context, goalID uuid.UUID, contribution *models.GoalContribution) error
	GetContributions(ctx context.Context, goalID uuid.UUID) ([]models.GoalContribution, error)
}

type goalRepository struct {
	pool *pgxpool.Pool
}

func NewGoalRepository(pool *pgxpool.Pool) GoalRepository {
	return &goalRepository{pool: pool}
}

func (r *goalRepository) Create(ctx context.Context, goal *models.Goal) error {
	query := `
		INSERT INTO goals (id, user_id, account_id, name, description, target_amount, current_amount, currency, target_date, icon, color, status, priority, auto_contribute, contribute_amount, contribute_freq, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	if goal.ID == uuid.Nil {
		goal.ID = uuid.New()
	}
	now := time.Now()
	goal.CreatedAt = now
	goal.UpdatedAt = now
	goal.Status = models.GoalStatusActive

	_, err := r.pool.Exec(ctx, query,
		goal.ID, goal.UserID, goal.AccountID, goal.Name, goal.Description,
		goal.TargetAmount, goal.CurrentAmount, goal.Currency, goal.TargetDate,
		goal.Icon, goal.Color, goal.Status, goal.Priority,
		goal.AutoContribute, goal.ContributeAmount, goal.ContributeFreq,
		goal.CreatedAt, goal.UpdatedAt,
	)
	return err
}

func (r *goalRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Goal, error) {
	query := `
		SELECT id, user_id, account_id, name, description, target_amount, current_amount, currency, target_date, icon, color, status, priority, auto_contribute, contribute_amount, contribute_freq, created_at, updated_at, completed_at
		FROM goals
		WHERE id = $1
	`

	var goal models.Goal
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&goal.ID, &goal.UserID, &goal.AccountID, &goal.Name, &goal.Description,
		&goal.TargetAmount, &goal.CurrentAmount, &goal.Currency, &goal.TargetDate,
		&goal.Icon, &goal.Color, &goal.Status, &goal.Priority,
		&goal.AutoContribute, &goal.ContributeAmount, &goal.ContributeFreq,
		&goal.CreatedAt, &goal.UpdatedAt, &goal.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	if goal.TargetAmount.GreaterThan(decimal.Zero) {
		goal.Progress = goal.CurrentAmount.Div(goal.TargetAmount).Mul(decimal.NewFromInt(100)).InexactFloat64()
	}

	if goal.TargetDate != nil {
		goal.DaysRemaining = int(time.Until(*goal.TargetDate).Hours() / 24)
		if goal.DaysRemaining < 0 {
			goal.DaysRemaining = 0
		}

		if goal.DaysRemaining > 0 {
			remaining := goal.TargetAmount.Sub(goal.CurrentAmount)
			monthsRemaining := decimal.NewFromFloat(float64(goal.DaysRemaining) / 30)
			if monthsRemaining.GreaterThan(decimal.Zero) {
				goal.RequiredMonthly = remaining.Div(monthsRemaining)
			}
		}
	}

	return &goal, nil
}

func (r *goalRepository) GetByUserID(ctx context.Context, userID uuid.UUID, status *models.GoalStatus) ([]models.Goal, error) {
	query := `
		SELECT id, user_id, account_id, name, description, target_amount, current_amount, currency, target_date, icon, color, status, priority, auto_contribute, contribute_amount, contribute_freq, created_at, updated_at, completed_at
		FROM goals
		WHERE user_id = $1
	`

	args := []interface{}{userID}
	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}
	query += " ORDER BY priority DESC, created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var goal models.Goal
		err := rows.Scan(
			&goal.ID, &goal.UserID, &goal.AccountID, &goal.Name, &goal.Description,
			&goal.TargetAmount, &goal.CurrentAmount, &goal.Currency, &goal.TargetDate,
			&goal.Icon, &goal.Color, &goal.Status, &goal.Priority,
			&goal.AutoContribute, &goal.ContributeAmount, &goal.ContributeFreq,
			&goal.CreatedAt, &goal.UpdatedAt, &goal.CompletedAt,
		)
		if err != nil {
			return nil, err
		}

		if goal.TargetAmount.GreaterThan(decimal.Zero) {
			goal.Progress = goal.CurrentAmount.Div(goal.TargetAmount).Mul(decimal.NewFromInt(100)).InexactFloat64()
		}

		goals = append(goals, goal)
	}
	return goals, rows.Err()
}

func (r *goalRepository) Update(ctx context.Context, id uuid.UUID, update *models.GoalUpdate) error {
	query := `
		UPDATE goals SET
			account_id = COALESCE($2, account_id),
			name = COALESCE($3, name),
			description = COALESCE($4, description),
			target_amount = COALESCE($5, target_amount),
			current_amount = COALESCE($6, current_amount),
			target_date = COALESCE($7, target_date),
			icon = COALESCE($8, icon),
			color = COALESCE($9, color),
			status = COALESCE($10, status),
			priority = COALESCE($11, priority),
			auto_contribute = COALESCE($12, auto_contribute),
			contribute_amount = COALESCE($13, contribute_amount),
			contribute_freq = COALESCE($14, contribute_freq),
			updated_at = $15
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		id, update.AccountID, update.Name, update.Description,
		update.TargetAmount, update.CurrentAmount, update.TargetDate,
		update.Icon, update.Color, update.Status, update.Priority,
		update.AutoContribute, update.ContributeAmount, update.ContributeFreq,
		time.Now(),
	)
	return err
}

func (r *goalRepository) UpdateAmount(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	query := `
		UPDATE goals SET
			current_amount = current_amount + $2,
			updated_at = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, amount, time.Now())
	if err != nil {
		return err
	}

	checkQuery := `
		UPDATE goals SET 
			status = 'completed', 
			completed_at = $2 
		WHERE id = $1 AND current_amount >= target_amount AND status = 'active'
	`
	_, err = r.pool.Exec(ctx, checkQuery, id, time.Now())
	return err
}

func (r *goalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM goals WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *goalRepository) AddContribution(ctx context.Context, goalID uuid.UUID, contribution *models.GoalContribution) error {
	query := `
		INSERT INTO goal_contributions (id, goal_id, amount, date, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if contribution.ID == uuid.Nil {
		contribution.ID = uuid.New()
	}
	contribution.GoalID = goalID
	contribution.CreatedAt = time.Now()
	if contribution.Date.IsZero() {
		contribution.Date = time.Now()
	}

	_, err := r.pool.Exec(ctx, query,
		contribution.ID, contribution.GoalID, contribution.Amount,
		contribution.Date, contribution.Notes, contribution.CreatedAt,
	)
	if err != nil {
		return err
	}

	return r.UpdateAmount(ctx, goalID, contribution.Amount)
}

func (r *goalRepository) GetContributions(ctx context.Context, goalID uuid.UUID) ([]models.GoalContribution, error) {
	query := `
		SELECT id, goal_id, amount, date, notes, created_at
		FROM goal_contributions
		WHERE goal_id = $1
		ORDER BY date DESC
	`

	rows, err := r.pool.Query(ctx, query, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contributions []models.GoalContribution
	for rows.Next() {
		var c models.GoalContribution
		err := rows.Scan(&c.ID, &c.GoalID, &c.Amount, &c.Date, &c.Notes, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		contributions = append(contributions, c)
	}
	return contributions, rows.Err()
}
