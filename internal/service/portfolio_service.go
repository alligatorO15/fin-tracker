package service

import (
	"context"

	"github.com/alligatorO15/fin-tracker/internal/market"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PortfolioService interface {
	Create(ctx context.Context, userID uuid.UUID, input *models.PortfolioCreate) (*models.Portfolio, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Portfolio, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Portfolio, error)
	GetWithHoldings(ctx context.Context, id uuid.UUID) (*models.Portfolio, error)
	Update(ctx context.Context, id uuid.UUID, update *models.PortfolioUpdate) (*models.Portfolio, error)
	Delete(ctx context.Context, id uuid.UUID) error
	RefreshPrices(ctx context.Context, portfolioID uuid.UUID) error
}

type portfolioService struct {
	portfolioRepo  repository.PortfolioRepository
	holdingRepo    repository.HoldingRepository
	securityRepo   repository.SecurityRepository
	marketProvider *market.MultiProvider
}

func NewPortfolioService(
	portfolioRepo repository.PortfolioRepository,
	holdingRepo repository.HoldingRepository,
	securityRepo repository.SecurityRepository,
	marketProvider *market.MultiProvider,
) PortfolioService {
	return &portfolioService{
		portfolioRepo:  portfolioRepo,
		holdingRepo:    holdingRepo,
		securityRepo:   securityRepo,
		marketProvider: marketProvider,
	}
}

func (s *portfolioService) Create(ctx context.Context, userID uuid.UUID, input *models.PortfolioCreate) (*models.Portfolio, error) {
	portfolio := &models.Portfolio{
		UserID:        userID,
		AccountID:     input.AccountID,
		Name:          input.Name,
		Description:   input.Description,
		Currency:      input.Currency,
		BrokerName:    input.BrokerName,
		BrokerAccount: input.BrokerAccount,
	}

	if err := s.portfolioRepo.Create(ctx, portfolio); err != nil {
		return nil, err
	}

	return portfolio, nil
}

func (s *portfolioService) GetByID(ctx context.Context, id uuid.UUID) (*models.Portfolio, error) {
	return s.portfolioRepo.GetByID(ctx, id)
}

func (s *portfolioService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Portfolio, error) {
	portfolios, err := s.portfolioRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	for i := range portfolios {
		holdings, err := s.holdingRepo.GetByPortfolioID(ctx, portfolios[i].ID)
		if err != nil {
			continue
		}

		var totalValue, totalInvested decimal.Decimal
		for _, h := range holdings {
			totalValue = totalValue.Add(h.CurrentValue)
			totalInvested = totalInvested.Add(h.TotalCost)
		}

		portfolios[i].TotalValue = totalValue
		portfolios[i].TotalInvested = totalInvested
		portfolios[i].TotalProfit = totalValue.Sub(totalInvested)

		if totalInvested.GreaterThan(decimal.Zero) {
			portfolios[i].ProfitPercent = portfolios[i].TotalProfit.Div(totalInvested).Mul(decimal.NewFromInt(100))
		}
	}

	return portfolios, nil
}

func (s *portfolioService) GetWithHoldings(ctx context.Context, id uuid.UUID) (*models.Portfolio, error) {
	portfolio, err := s.portfolioRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	holdings, err := s.holdingRepo.GetByPortfolioID(ctx, id)
	if err != nil {
		return nil, err
	}

	portfolio.Holdings = holdings

	var totalValue, totalInvested decimal.Decimal
	for _, h := range holdings {
		totalValue = totalValue.Add(h.CurrentValue)
		totalInvested = totalInvested.Add(h.TotalCost)
	}

	portfolio.TotalValue = totalValue
	portfolio.TotalInvested = totalInvested
	portfolio.TotalProfit = totalValue.Sub(totalInvested)

	if totalInvested.GreaterThan(decimal.Zero) {
		portfolio.ProfitPercent = portfolio.TotalProfit.Div(totalInvested).Mul(decimal.NewFromInt(100))
	}

	return portfolio, nil
}

func (s *portfolioService) Update(ctx context.Context, id uuid.UUID, update *models.PortfolioUpdate) (*models.Portfolio, error) {
	if err := s.portfolioRepo.Update(ctx, id, update); err != nil {
		return nil, err
	}
	return s.portfolioRepo.GetByID(ctx, id)
}

func (s *portfolioService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.portfolioRepo.Delete(ctx, id)
}

func (s *portfolioService) RefreshPrices(ctx context.Context, portfolioID uuid.UUID) error {
	holdings, err := s.holdingRepo.GetByPortfolioID(ctx, portfolioID)
	if err != nil {
		return err
	}

	// группируем позиции в портфеле по биржам
	exchangeHoldings := make(map[models.Exchange][]*models.Holding)
	for i := range holdings {
		if holdings[i].Security != nil {
			exchange := holdings[i].Security.Exchange
			exchangeHoldings[exchange] = append(exchangeHoldings[exchange], &holdings[i])
		}
	}

	for exchange, hs := range exchangeHoldings {
		var tickers []string
		tickerToHolding := make(map[string]*models.Holding)

		for _, h := range hs {
			tickers = append(tickers, h.Security.Ticker)
			tickerToHolding[h.Security.Ticker] = h
		}

		quotes, err := s.marketProvider.GetQuotes(ctx, tickers, exchange)
		if err != nil {
			continue
		}

		// апдейтим цены бумаг
		for ticker, quote := range quotes {
			h := tickerToHolding[ticker]
			if h != nil && h.Security != nil {
				s.securityRepo.UpdatePrice(ctx, h.Security.ID, quote.LastPrice, quote.Change, quote.ChangePercent, quote.Volume)
			}
		}
	}

	return nil
}
