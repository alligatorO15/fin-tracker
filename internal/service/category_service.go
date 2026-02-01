package service

import (
	"context"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/google/uuid"
)

type CategoryService interface {
	Create(ctx context.Context, userID uuid.UUID, input *models.CategoryCreate) (*models.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Category, error)
	GetByType(ctx context.Context, userID uuid.UUID, categoryType models.CategoryType) ([]models.Category, error)
	Update(ctx context.Context, id uuid.UUID, update *models.CategoryUpdate) (*models.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type categoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) CategoryService {
	return &categoryService{categoryRepo: categoryRepo}
}

func (s *categoryService) Create(ctx context.Context, userID uuid.UUID, input *models.CategoryCreate) (*models.Category, error) {
	existingCategories, err := s.categoryRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// находим максимальный SortOrder
	maxSortOrder := 0
	for _, cat := range existingCategories {
		if cat.Type == input.Type && cat.SortOrder > maxSortOrder {
			maxSortOrder = cat.SortOrder
		}
	}

	category := &models.Category{
		UserID:    &userID,
		Name:      input.Name,
		Type:      input.Type,
		Icon:      input.Icon,
		Color:     input.Color,
		ParentID:  input.ParentID,
		IsSystem:  false,
		SortOrder: maxSortOrder + 1, // следующий порядковый номер
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *categoryService) GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *categoryService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Category, error) {
	categories, err := s.categoryRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// создаем дерево категорий
	return s.buildCategoryTree(categories), nil
}

func (s *categoryService) GetByType(ctx context.Context, userID uuid.UUID, categoryType models.CategoryType) ([]models.Category, error) {
	return s.categoryRepo.GetByType(ctx, userID, categoryType)
}

func (s *categoryService) Update(ctx context.Context, id uuid.UUID, update *models.CategoryUpdate) (*models.Category, error) {
	if err := s.categoryRepo.Update(ctx, id, update); err != nil {
		return nil, err
	}
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *categoryService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.categoryRepo.Delete(ctx, id)
}

func (s *categoryService) buildCategoryTree(categories []models.Category) []models.Category {
	// создаем хэш-таблицу (мапу) чтобы время поиска О(1)
	categoryMap := make(map[uuid.UUID]*models.Category)
	for i := range categories {
		categoryMap[categories[i].ID] = &categories[i]
	}

	// создаем дерево категорий родителей, а в поле Children будут дети (смотр. в models)
	var rootCategories []models.Category
	for i := range categories {
		if categories[i].ParentID == nil {
			rootCategories = append(rootCategories, categories[i])
		} else {
			if parent, ok := categoryMap[*categories[i].ParentID]; ok {
				parent.Children = append(parent.Children, categories[i])
			}
		}
	}

	return rootCategories
}
