package service

import (
	"context"

	"github.com/alligatorO15/fin-tracker/internal/market"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountService interface {
	Create(ctx context.Context, userID uuid.UUID, input *models.AccountCreate) (*models.Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Account, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Account, error)
	GetSummary(ctx context.Context, userID uuid.UUID) (*models.AccountSummary, error)
	Update(ctx context.Context, id uuid.UUID, update *models.AccountUpdate) (*models.Account, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type accountService struct {
	accountRepo    repository.AccountRepository
	userRepo       repository.UserRepository
	marketProvider *market.MultiProvider
}

func NewAccountService(accountRepo repository.AccountRepository, userRepo repository.UserRepository, marketProvider *market.MultiProvider) AccountService {
	return &accountService{
		accountRepo:    accountRepo,
		userRepo:       userRepo,
		marketProvider: marketProvider,
	}
}

func (s *accountService) Create(ctx context.Context, userID uuid.UUID, input *models.AccountCreate) (*models.Account, error) {
	account := &models.Account{
		UserID:         userID,
		Name:           input.Name,
		Type:           input.Type,
		Currency:       input.Currency,
		InitialBalance: input.InitialBalance,
		Icon:           input.Icon,
		Color:          input.Color,
		Institution:    input.Institution,
		AccountNumber:  input.AccountNumber,
		Notes:          input.Notes,
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, err
	}

	return account, nil
}

func (s *accountService) GetByID(ctx context.Context, id uuid.UUID) (*models.Account, error) {
	return s.accountRepo.GetByID(ctx, id)
}

func (s *accountService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Account, error) {
	return s.accountRepo.GetByUserID(ctx, userID)
}

func (s *accountService) GetSummary(ctx context.Context, userID uuid.UUID) (*models.AccountSummary, error) {
	summary, err := s.accountRepo.GetSummary(ctx, userID)
	if err != nil {
		return nil, err
	}

	// получаем базовую валюту пользователя
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	baseCurrency := user.DefaultCurrency
	if baseCurrency == "" {
		baseCurrency = "RUB"
	}
	summary.BaseCurrency = baseCurrency

	// конвертируем все валюты в базовую для расчёта TotalBalance
	for currency, balance := range summary.BalanceByCurrency {
		if currency == baseCurrency {
			summary.TotalBalance = summary.TotalBalance.Add(balance)
		} else {
			// получаем курс валюты
			rate, err := s.marketProvider.GetCurrencyRate(ctx, currency, baseCurrency)
			if err != nil {
				// если не удалось получить курс, пропускаем эту валюту
				continue
			}
			summary.TotalBalance = summary.TotalBalance.Add(balance.Mul(rate))
		}
	}

	return summary, nil
}

func (s *accountService) Update(ctx context.Context, id uuid.UUID, update *models.AccountUpdate) (*models.Account, error) {
	if err := s.accountRepo.Update(ctx, id, update); err != nil {
		return nil, err
	}
	return s.accountRepo.GetByID(ctx, id)
}

func (s *accountService) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return s.accountRepo.UpdateBalance(ctx, id, amount)
}

func (s *accountService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.accountRepo.Delete(ctx, id)
}
