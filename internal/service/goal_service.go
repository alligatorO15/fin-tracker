package service

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type GoalService interface {
	Create(ctx context.Context, userID uuid.UUID, input *models.GoalCreate) (*models.Goal, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Goal, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, status *models.GoalStatus) ([]models.Goal, error)
	Update(ctx context.Context, id uuid.UUID, update *models.GoalUpdate) (*models.Goal, error)
	AddContribution(ctx context.Context, goalID uuid.UUID, input *models.GoalContributionCreate) (*models.Goal, error)
	GetContributions(ctx context.Context, goalID uuid.UUID) ([]models.GoalContribution, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type goalService struct {
	goalRepo repository.GoalRepository
}

func NewGoalService(goalRepo repository.GoalRepository) GoalService {
	return &goalService{goalRepo: goalRepo}
}

func (s *goalService) Create(ctx context.Context, userID uuid.UUID, input *models.GoalCreate) (*models.Goal, error) {
	goal := &models.Goal{
		UserID:           userID,
		AccountID:        input.AccountID,
		Name:             input.Name,
		Description:      input.Description,
		TargetAmount:     input.TargetAmount,
		CurrentAmount:    input.CurrentAmount,
		Currency:         input.Currency,
		TargetDate:       input.TargetDate,
		Icon:             input.Icon,
		Color:            input.Color,
		Priority:         input.Priority,
		AutoContribute:   input.AutoContribute,
		ContributeAmount: input.ContributeAmount,
		ContributeFreq:   input.ContributeFreq,
	}

	if err := s.goalRepo.Create(ctx, goal); err != nil {
		return nil, err
	}

	goal, err := s.goalRepo.GetByID(ctx, goal.ID)
	if err != nil {
		return nil, err
	}
	s.enrichGoal(goal)
	return goal, nil
}

func (s *goalService) GetByID(ctx context.Context, id uuid.UUID) (*models.Goal, error) {
	goal, err := s.goalRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichGoal(goal)
	return goal, nil
}

func (s *goalService) GetByUserID(ctx context.Context, userID uuid.UUID, status *models.GoalStatus) ([]models.Goal, error) {
	goals, err := s.goalRepo.GetByUserID(ctx, userID, status)
	if err != nil {
		return nil, err
	}
	for i := range goals {
		s.enrichGoal(&goals[i])
	}
	return goals, nil
}

// enrichGoal вычисляет прогресс, дни до завершения и необходимый ежемесячный взнос
func (s *goalService) enrichGoal(goal *models.Goal) {
	// прогресс в процентах
	if !goal.TargetAmount.IsZero() {
		progress := goal.CurrentAmount.Div(goal.TargetAmount).Mul(decimal.NewFromInt(100))
		if f := progress.InexactFloat64(); f >= 0 && f <= 100 {
			goal.Progress = f
		} else if f > 100 {
			goal.Progress = 100
		}
	}

	// дни до целевой даты
	if goal.TargetDate != nil {
		days := int(time.Until(*goal.TargetDate).Hours() / 24)
		if days >= 0 {
			goal.DaysRemaining = days
		}
	}

	// необходимый ежемесячный взнос
	if goal.TargetDate != nil && !goal.TargetAmount.IsZero() {
		remaining := goal.TargetAmount.Sub(goal.CurrentAmount)
		if remaining.IsPositive() {
			months := time.Until(*goal.TargetDate).Hours() / 24 / 30.44 // Среднее кол-во дней в месяце
			if months > 0 {
				goal.RequiredMonthly = remaining.Div(decimal.NewFromFloat(months))
			}
		}
	}
}

func (s *goalService) Update(ctx context.Context, id uuid.UUID, update *models.GoalUpdate) (*models.Goal, error) {
	if err := s.goalRepo.Update(ctx, id, update); err != nil {
		return nil, err
	}
	goal, err := s.goalRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichGoal(goal)
	return goal, nil
}

func (s *goalService) AddContribution(ctx context.Context, goalID uuid.UUID, input *models.GoalContributionCreate) (*models.Goal, error) {
	contribution := &models.GoalContribution{
		Amount: input.Amount,
		Date:   input.Date,
		Notes:  input.Notes,
	}

	if err := s.goalRepo.AddContribution(ctx, goalID, contribution); err != nil {
		return nil, err
	}

	goal, err := s.goalRepo.GetByID(ctx, goalID)
	if err != nil {
		return nil, err
	}
	s.enrichGoal(goal)
	return goal, nil
}

func (s *goalService) GetContributions(ctx context.Context, goalID uuid.UUID) ([]models.GoalContribution, error) {
	return s.goalRepo.GetContributions(ctx, goalID)
}

func (s *goalService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.goalRepo.Delete(ctx, id)
}
