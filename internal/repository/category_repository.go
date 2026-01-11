package repository

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Category, error)
	GetByType(ctx context.Context, userID uuid.UUID, categoryType models.CategoryType) ([]models.Category, error)
	GetSystemCategories(ctx context.Context) ([]models.Category, error)
	Update(ctx context.Context, id uuid.UUID, update *models.CategoryUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type categoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) CategoryRepository {
	return &categoryRepository{pool: pool}
}

func (r *categoryRepository) Create(ctx context.Context, category *models.Category) error {
	query := `
		INSERT INTO categories (id, user_id, name, type, icon, color, parent_id, is_system, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if category.ID == uuid.Nil {
		category.ID = uuid.New()
	}
	now := time.Now()
	category.CreatedAt = now
	category.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		category.ID, category.UserID, category.Name, category.Type,
		category.Icon, category.Color, category.ParentID,
		category.IsSystem, category.SortOrder,
		category.CreatedAt, category.UpdatedAt,
	)
	return err
}

func (r *categoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	query := `
		SELECT id, user_id, name, type, icon, color, parent_id, is_system, sort_order, created_at, updated_at
		FROM categories
		WHERE id = $1
	`

	var category models.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&category.ID, &category.UserID, &category.Name, &category.Type,
		&category.Icon, &category.Color, &category.ParentID,
		&category.IsSystem, &category.SortOrder,
		&category.CreatedAt, &category.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Category, error) {
	query := `
		SELECT id, user_id, name, type, icon, color, parent_id, is_system, sort_order, created_at, updated_at
		FROM categories
		WHERE (user_id = $1 OR is_system = true)
		ORDER BY sort_order,name 
	`
	return r.queryCategories(ctx, query, userID)
}

func (r *categoryRepository) GetByType(ctx context.Context, userID uuid.UUID, categoryType models.CategoryType) ([]models.Category, error) {
	query := `
		SELECT id, user_id, name, type, icon, color, parent_id, is_system, sort_order, created_at, updated_at
		FROM categories
		WHERE (user_id = $1 OR is_system = true) AND type = $2
		ORDER BY sort_order, name
	`

	return r.queryCategories(ctx, query, userID, categoryType)
}

func (r *categoryRepository) GetSystemCategories(ctx context.Context) ([]models.Category, error) {
	query := `
		SELECT id, user_id, name, type, icon, color, parent_id, is_system, sort_order, created_at, updated_at
		FROM categories
		WHERE is_system = true
		ORDER BY sort_order, name
	`

	return r.queryCategories(ctx, query)
}
func (r *categoryRepository) queryCategories(ctx context.Context, query string, args ...interface{}) ([]models.Category, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID, &category.UserID, &category.Name, &category.Type,
			&category.Icon, &category.Color, &category.ParentID,
			&category.IsSystem, &category.SortOrder,
			&category.CreatedAt, &category.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (r *categoryRepository) Update(ctx context.Context, id uuid.UUID, update *models.CategoryUpdate) error {
	query := `
		UPDATE categories SET
			name = COALESCE($2, name),
			icon = COALESCE($3, icon),
			color = COALESCE($4, color),
			parent_id = COALESCE($5, parent_id),
			sort_order = COALESCE($6, sort_order),
			updated_at = $7
		WHERE id = $1 AND is_system = false
	`

	_, err := r.pool.Exec(ctx, query,
		id, update.Name, update.Icon, update.Color,
		update.ParentID, update.SortOrder, time.Now(),
	)
	return err
}

func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM categories WHERE id = $1 AND is_system = false`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
