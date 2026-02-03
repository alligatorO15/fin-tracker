package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	GetByFilter(ctx context.Context, userID uuid.UUID, filter *models.TransactionFilter) (*models.TransactionList, error)
	Update(ctx context.Context, id uuid.UUID, update *models.TransactionUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetTags(ctx context.Context, transactionID uuid.UUID) ([]string, error)
	SetTags(ctx context.Context, transactionID uuid.UUID, tags []string) error
	GetSumByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, txType models.TransactionType) (map[uuid.UUID]decimal.Decimal, error)
	GetSumByPeriod(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, groupBy string) ([]models.CashFlow, error)
}

type transactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) TransactionRepository {
	return &transactionRepository{pool: pool}
}

// db возвращает транзакцию из контекста или pool
func (r *transactionRepository) db(ctx context.Context) DBTX {
	return GetTxOrPool(ctx, r.pool)
}

func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (id, user_id, account_id, category_id, type, amount, currency, description, date, to_account_id, to_amount, is_recurring, recurrence_rule, parent_transaction_id, location, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	now := time.Now()
	tx.CreatedAt = now
	tx.UpdatedAt = now

	_, err := r.db(ctx).Exec(ctx, query,
		tx.ID, tx.UserID, tx.AccountID, tx.CategoryID, tx.Type,
		tx.Amount, tx.Currency, tx.Description, tx.Date,
		tx.ToAccountID, tx.ToAmount, tx.IsRecurring, tx.RecurrenceRule,
		tx.ParentTransactionID, tx.Location, tx.Notes,
		tx.CreatedAt, tx.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if len(tx.Tags) > 0 {
		return r.SetTags(ctx, tx.ID, tx.Tags)
	}

	return nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.account_id, t.category_id, t.type, t.amount, t.currency, t.description, t.date, t.to_account_id, t.to_amount, t.is_recurring, t.recurrence_rule, t.parent_transaction_id, t.location, t.notes, t.created_at, t.updated_at
		FROM transactions t
		WHERE t.id = $1 AND t.deleted_at IS NULL
	`

	var tx models.Transaction
	err := r.db(ctx).QueryRow(ctx, query, id).Scan(
		&tx.ID, &tx.UserID, &tx.AccountID, &tx.CategoryID, &tx.Type,
		&tx.Amount, &tx.Currency, &tx.Description, &tx.Date,
		&tx.ToAccountID, &tx.ToAmount, &tx.IsRecurring, &tx.RecurrenceRule,
		&tx.ParentTransactionID, &tx.Location, &tx.Notes,
		&tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	tx.Tags, _ = r.GetTags(ctx, id)

	return &tx, nil
}

func (r *transactionRepository) GetByFilter(ctx context.Context, userID uuid.UUID, filter *models.TransactionFilter) (*models.TransactionList, error) {
	baseQuery := `
		SELECT t.id, t.user_id, t.account_id, t.category_id, t.type, t.amount, t.currency, t.description, t.date, t.to_account_id, t.to_amount, t.is_recurring, t.recurrence_rule, t.parent_transaction_id, t.location, t.notes, t.created_at, t.updated_at
		FROM transactions t
		WHERE t.user_id = $1 AND t.deleted_at IS NULL
	`
	countQuery := `SELECT COUNT(*) FROM transactions t WHERE t.user_id = $1 AND t.deleted_at IS NULL`

	var conditions []string
	args := []interface{}{userID}
	argIndex := 2

	if filter.AccountID != nil {
		conditions = append(conditions, fmt.Sprintf("t.account_id = $%d", argIndex))
		args = append(args, *filter.AccountID)
		argIndex++
	}

	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("t.category_id = $%d", argIndex))
		args = append(args, *filter.CategoryID)
		argIndex++
	}

	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("t.type = $%d", argIndex))
		args = append(args, *filter.Type)
		argIndex++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("t.date >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("t.date <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}

	if filter.AmountMin != nil {
		conditions = append(conditions, fmt.Sprintf("t.amount >= $%d", argIndex))
		args = append(args, *filter.AmountMin)
		argIndex++
	}

	if filter.AmountMax != nil {
		conditions = append(conditions, fmt.Sprintf("t.amount <= $%d", argIndex))
		args = append(args, *filter.AmountMax)
		argIndex++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(t.description ILIKE $%d OR t.notes ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	var total int64
	err := r.db(ctx).QueryRow(ctx, countQuery+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	sortBy := "date"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	finalQuery := baseQuery + whereClause + fmt.Sprintf(" ORDER BY t.%s %s LIMIT $%d OFFSET $%d", sortBy, sortOrder, argIndex, argIndex+1)
	args = append(args, filter.Limit, offset)

	rows, err := r.db(ctx).Query(ctx, finalQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.AccountID, &tx.CategoryID, &tx.Type,
			&tx.Amount, &tx.Currency, &tx.Description, &tx.Date,
			&tx.ToAccountID, &tx.ToAmount, &tx.IsRecurring, &tx.RecurrenceRule,
			&tx.ParentTransactionID, &tx.Location, &tx.Notes,
			&tx.CreatedAt, &tx.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &models.TransactionList{
		Transactions: transactions,
		Total:        total,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   totalPages,
	}, nil
}

func (r *transactionRepository) Update(ctx context.Context, id uuid.UUID, update *models.TransactionUpdate) error {
	query := `
		UPDATE transactions SET
			account_id = COALESCE($2, account_id),
			category_id = COALESCE($3, category_id),
			amount = COALESCE($4, amount),
			description = COALESCE($5, description),
			date = COALESCE($6, date),
			to_account_id = COALESCE($7, to_account_id),
			to_amount = COALESCE($8, to_amount),
			location = COALESCE($9, location),
			notes = COALESCE($10, notes),
			updated_at = $11
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db(ctx).Exec(ctx, query,
		id, update.AccountID, update.CategoryID, update.Amount,
		update.Description, update.Date, update.ToAccountID, update.ToAmount,
		update.Location, update.Notes, time.Now(),
	)

	if err != nil {
		return err
	}

	if update.Tags != nil {
		return r.SetTags(ctx, id, update.Tags)
	}

	return nil
}

func (r *transactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE transactions SET deleted_at = $2 WHERE id = $1`
	_, err := r.db(ctx).Exec(ctx, query, id, time.Now())
	return err
}

func (r *transactionRepository) GetTags(ctx context.Context, transactionID uuid.UUID) ([]string, error) {
	query := `SELECT tag FROM transaction_tags WHERE transaction_id = $1`

	rows, err := r.db(ctx).Query(ctx, query, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *transactionRepository) SetTags(ctx context.Context, transactionID uuid.UUID, tags []string) error {
	_, err := r.db(ctx).Exec(ctx, `DELETE FROM transaction_tags WHERE transaction_id = $1`, transactionID)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		_, err := r.db(ctx).Exec(ctx, `INSERT INTO transaction_tags (transaction_id, tag) VALUES ($1, $2)`, transactionID, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *transactionRepository) GetSumByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, txType models.TransactionType) (map[uuid.UUID]decimal.Decimal, error) {
	query := `
		SELECT category_id, SUM(amount) 
		FROM transactions 
		WHERE user_id = $1 AND date >= $2 AND date <= $3 AND type = $4 AND deleted_at IS NULL
		GROUP BY category_id
	`

	rows, err := r.db(ctx).Query(ctx, query, userID, startDate, endDate, txType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]decimal.Decimal)
	for rows.Next() {
		var categoryID uuid.UUID
		var sum decimal.Decimal
		if err := rows.Scan(&categoryID, &sum); err != nil {
			return nil, err
		}
		result[categoryID] = sum
	}
	return result, rows.Err()
}

func (r *transactionRepository) GetSumByPeriod(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, groupBy string) ([]models.CashFlow, error) {
	var dateFormat string
	switch groupBy {
	case "day":
		dateFormat = "YYYY-MM-DD"
	case "week":
		dateFormat = "IYYY-IW"
	case "month":
		dateFormat = "YYYY-MM"
	case "year":
		dateFormat = "YYYY"
	default:
		dateFormat = "YYYY-MM"
	}

	query := fmt.Sprintf(`
		SELECT 
			TO_CHAR(date, '%s') as period,
			SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) as income,
			SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) as expenses
		FROM transactions 
		WHERE user_id = $1 AND date >= $2 AND date <= $3 AND deleted_at IS NULL
		GROUP BY period
		ORDER BY period
	`, dateFormat)

	rows, err := r.db(ctx).Query(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.CashFlow
	for rows.Next() {
		var cf models.CashFlow
		if err := rows.Scan(&cf.Period, &cf.Income, &cf.Expenses); err != nil {
			return nil, err
		}
		cf.Net = cf.Income.Sub(cf.Expenses)
		result = append(result, cf)
	}
	return result, rows.Err()
}
