package service

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BudgetService interface {
	Create(ctx context.Context, userID uuid.UUID, input *models.BudgetCreate) (*models.Budget, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Budget, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, activeOnly bool) ([]models.Budget, error)
	GetSummary(ctx context.Context, userID uuid.UUID) (*models.BudgetSummary, error)
	GetAlerts(ctx context.Context, userID uuid.UUID) ([]models.BudgetAlert, error)
	Update(ctx context.Context, id uuid.UUID, update *models.BudgetUpdate) (*models.Budget, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type budgetService struct {
	budgetRepo      repository.BudgetRepository
	transactionRepo repository.TransactionRepository
	categoryRepo    repository.CategoryRepository
}

func NewBudgetService(budgetRepo repository.BudgetRepository, transactionRepo repository.TransactionRepository, categoryRepo repository.CategoryRepository) BudgetService {
	return &budgetService{
		budgetRepo:      budgetRepo,
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
	}
}

func (s *budgetService) Create(ctx context.Context, userID uuid.UUID, input *models.BudgetCreate) (*models.Budget, error) {
	budget := &models.Budget{
		UserID:       userID,
		CategoryID:   input.CategoryID,
		Name:         input.Name,
		Amount:       input.Amount,
		Currency:     input.Currency,
		Period:       input.Period,
		StartDate:    input.StartDate,
		EndDate:      input.EndDate,
		AlertPercent: input.AlertPercent,
		Notes:        input.Notes,
	}

	if budget.AlertPercent == 0 {
		budget.AlertPercent = 80
	}

	if err := s.budgetRepo.Create(ctx, budget); err != nil {
		return nil, err
	}

	// вычисляем поля
	return s.calculateBudgetSpent(ctx, budget)
}

func (s *budgetService) GetByID(ctx context.Context, id uuid.UUID) (*models.Budget, error) {
	budget, err := s.budgetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.calculateBudgetSpent(ctx, budget)
}

func (s *budgetService) GetByUserID(ctx context.Context, userID uuid.UUID, activeOnly bool) ([]models.Budget, error) {
	budgets, err := s.budgetRepo.GetByUserID(ctx, userID, activeOnly)
	if err != nil {
		return nil, err
	}

	for i := range budgets {
		updated, _ := s.calculateBudgetSpent(ctx, &budgets[i])
		if updated != nil {
			budgets[i] = *updated
		}
	}

	return budgets, nil
}

func (s *budgetService) GetSummary(ctx context.Context, userID uuid.UUID) (*models.BudgetSummary, error) {
	budgets, err := s.GetByUserID(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	summary := &models.BudgetSummary{
		Budgets: budgets,
	}

	for _, budget := range budgets {
		summary.TotalBudgeted = summary.TotalBudgeted.Add(budget.Amount)
		summary.TotalSpent = summary.TotalSpent.Add(budget.Spent)

		if budget.SpentPercent >= 100 {
			summary.OverBudgetCount++
		}
	}

	summary.TotalRemaining = summary.TotalBudgeted.Sub(summary.TotalSpent)

	return summary, nil
}

func (s *budgetService) GetAlerts(ctx context.Context, userID uuid.UUID) ([]models.BudgetAlert, error) {
	budgets, err := s.GetByUserID(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	var alerts []models.BudgetAlert
	for _, budget := range budgets {
		if budget.SpentPercent >= float64(budget.AlertPercent) {
			alertType := "warning"
			if budget.SpentPercent >= 100 {
				alertType = "exceeded"
			}

			alerts = append(alerts, models.BudgetAlert{
				BudgetID:   budget.ID,
				BudgetName: budget.Name,
				Amount:     budget.Amount,
				Spent:      budget.Spent,
				Percent:    budget.SpentPercent,
				AlertType:  alertType,
			})
		}
	}

	return alerts, nil
}

func (s *budgetService) Update(ctx context.Context, id uuid.UUID, update *models.BudgetUpdate) (*models.Budget, error) {
	if err := s.budgetRepo.Update(ctx, id, update); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *budgetService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.budgetRepo.Delete(ctx, id)
}

func (s *budgetService) calculateBudgetSpent(ctx context.Context, budget *models.Budget) (*models.Budget, error) {
	// вычисляем начало и конец бюджетирования
	startDate, endDate := s.getBudgetPeriodDates(budget)

	// расходы
	var spent decimal.Decimal

	if budget.CategoryID != nil {
		sums, err := s.transactionRepo.GetSumByCategory(ctx, budget.UserID, startDate, endDate, models.TransactionTypeExpense)
		if err == nil {
			if sum, ok := sums[*budget.CategoryID]; ok {
				spent = sum
			}
		}
	} else {
		// все категории
		sums, err := s.transactionRepo.GetSumByCategory(ctx, budget.UserID, startDate, endDate, models.TransactionTypeExpense)
		if err == nil {
			for _, sum := range sums {
				spent = spent.Add(sum)
			}
		}
	}

	budget.Spent = spent
	budget.Remaining = budget.Amount.Sub(spent)

	if budget.Amount.GreaterThan(decimal.Zero) {
		budget.SpentPercent = spent.Div(budget.Amount).Mul(decimal.NewFromInt(100)).InexactFloat64()
	}

	// достаем инфу о категории и добавляем в поле
	if budget.CategoryID != nil {
		category, err := s.categoryRepo.GetByID(ctx, *budget.CategoryID)
		if err == nil {
			budget.Category = category
		}
	}

	return budget, nil
}

func (s *budgetService) getBudgetPeriodDates(budget *models.Budget) (time.Time, time.Time) {
	now := time.Now()
	// логика такая: если указываем период не кастом то отсчитывается начало и конец от тек времени(budget.StartDate, *budget.EndDate игнорируюся ), если кастом то берется budget.StartDate, *budget.EndDate или now
	switch budget.Period {
	case models.BudgetPeriodWeekly:
		// начало тек. недели (пн)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := now.AddDate(0, 0, -weekday+1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 6)
		return start, end

	case models.BudgetPeriodMonthly:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, -1)
		return start, end

	case models.BudgetPeriodQuarterly:
		quarter := (int(now.Month()) - 1) / 3
		start := time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 3, -1)
		return start, end

	case models.BudgetPeriodYearly:
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())
		return start, end

	case models.BudgetPeriodCustom:
		if budget.EndDate != nil {
			return budget.StartDate, *budget.EndDate
		}
		return budget.StartDate, now

	default:
		return budget.StartDate, now
	}
}
