package service

import (
	"context"
	"errors"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/market"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrSecurityNotFound   = errors.New("security not found")
	ErrInsufficientShares = errors.New("insufficient shares for sale")
)

type InvestmentService interface {
	// ценные ьумаги
	SearchSecurities(ctx context.Context, query string, securityType *models.SecurityType, exchange *models.Exchange) ([]models.Security, error)
	GetSecurityByID(ctx context.Context, id uuid.UUID) (*models.Security, error)
	GetSecurityQuote(ctx context.Context, ticker string, exchange models.Exchange) (*models.MarketQuote, error)

	// транзакции
	AddTransaction(ctx context.Context, input *models.InvestmentTransactionCreate) (*models.InvestmentTransaction, error)
	GetTransactions(ctx context.Context, portfolioID uuid.UUID, limit, offset int) ([]models.InvestmentTransaction, error)
	GetTransactionsByDateRange(ctx context.Context, portfolioID uuid.UUID, start, end time.Time) ([]models.InvestmentTransaction, error)
	DeleteTransaction(ctx context.Context, id uuid.UUID) error

	// позиции(holdings)
	GetHoldings(ctx context.Context, portfolioID uuid.UUID) ([]models.Holding, error)
	GetHolding(ctx context.Context, portfolioID, securityID uuid.UUID) (*models.Holding, error)

	// получение аналитики
	GetPortfolioAnalytics(ctx context.Context, portfolioID uuid.UUID) (*models.PortfolioAnalytics, error)
	GetTaxReport(ctx context.Context, portfolioID uuid.UUID, year int) (*models.TaxReport, error)

	// дивидендные выплаты по портфелю
	GetUpcomingDividends(ctx context.Context, portfolioID uuid.UUID) ([]models.Dividend, error)
}

type investmentService struct {
	portfolioRepo  repository.PortfolioRepository
	holdingRepo    repository.HoldingRepository
	securityRepo   repository.SecurityRepository
	investmentRepo repository.InvestmentTransactionRepository
	marketProvider *market.MultiProvider
	txManager      repository.TxManager
}

func NewInvestmentService(
	portfolioRepo repository.PortfolioRepository,
	holdingRepo repository.HoldingRepository,
	securityRepo repository.SecurityRepository,
	investmentRepo repository.InvestmentTransactionRepository,
	marketProvider *market.MultiProvider,
	txManager repository.TxManager,
) InvestmentService {
	return &investmentService{
		portfolioRepo:  portfolioRepo,
		holdingRepo:    holdingRepo,
		securityRepo:   securityRepo,
		investmentRepo: investmentRepo,
		txManager:      txManager,
		marketProvider: marketProvider,
	}
}

func (s *investmentService) SearchSecurities(ctx context.Context, query string, securityType *models.SecurityType, exchange *models.Exchange) ([]models.Security, error) {
	// сначала ищем в бд
	dbResults, err := s.securityRepo.Search(ctx, query, 20)
	if err == nil && len(dbResults) > 0 {
		var filtered []models.Security
		for _, sec := range dbResults {
			if securityType != nil && sec.Type != *securityType {
				continue
			}
			if exchange != nil && sec.Exchange != *exchange {
				continue
			}
			filtered = append(filtered, sec)
		}
		if len(filtered) > 0 {
			return filtered, nil
		}
	}

	// если нет в бд, то делаем запрос к рын. провайдеру
	results, err := s.marketProvider.SearchSecurities(ctx, query, securityType, exchange)
	if err != nil {
		return nil, err
	}

	// сохраняем полученные бумаги в бд
	for i := range results {
		s.securityRepo.Create(ctx, &results[i])
	}

	return results, nil
}

func (s *investmentService) GetSecurityByID(ctx context.Context, id uuid.UUID) (*models.Security, error) {
	return s.securityRepo.GetByID(ctx, id)
}

func (s *investmentService) GetSecurityQuote(ctx context.Context, ticker string, exchange models.Exchange) (*models.MarketQuote, error) {
	return s.marketProvider.GetQuote(ctx, ticker, exchange)
}

func (s *investmentService) AddTransaction(ctx context.Context, input *models.InvestmentTransactionCreate) (*models.InvestmentTransaction, error) {
	security, err := s.securityRepo.GetByID(ctx, input.SecurityID)
	if err != nil {
		return nil, ErrSecurityNotFound
	}

	portfolio, err := s.portfolioRepo.GetByID(ctx, input.PortfolioID)
	if err != nil {
		return nil, err
	}

	// создаем транзакцию
	tx := &models.InvestmentTransaction{
		PortfolioID:  input.PortfolioID,
		SecurityID:   input.SecurityID,
		Type:         input.Type,
		Date:         input.Date,
		Quantity:     input.Quantity,
		Price:        input.Price,
		Amount:       input.Quantity.Mul(input.Price).Add(input.Commission),
		Commission:   input.Commission,
		Currency:     input.Currency,
		ExchangeRate: input.ExchangeRate,
		Notes:        input.Notes,
	}

	if tx.Currency == "" {
		tx.Currency = portfolio.Currency
	}
	if tx.ExchangeRate.IsZero() {
		tx.ExchangeRate = decimal.NewFromInt(1)
	}

	// атомарная операция: создание транзакции + обновление холдинга
	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Создаем транзакцию
		if err := s.investmentRepo.Create(txCtx, tx); err != nil {
			return err
		}

		// обновляем холдинги
		switch input.Type {
		case models.InvestmentTransactionTypeBuy:
			return s.updateHoldingOnBuy(txCtx, input.PortfolioID, input.SecurityID, input.Quantity, input.Price, input.Commission)
		case models.InvestmentTransactionTypeSell:
			return s.updateHoldingOnSell(txCtx, input.PortfolioID, input.SecurityID, input.Quantity)
		case models.InvestmentTransactionTypeDividend, models.InvestmentTransactionTypeCoupon:
			// при получении дивидендов/купонов холдинги не меняются
			return nil
		case models.InvestmentTransactionTypeSplit:
			return s.updateHoldingOnSplit(txCtx, input.PortfolioID, input.SecurityID, input.Quantity)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	tx.Security = security
	return tx, nil
}

func (s *investmentService) updateHoldingOnBuy(ctx context.Context, portfolioID, securityID uuid.UUID, quantity, price, commission decimal.Decimal) error {
	totalCost := quantity.Mul(price).Add(commission)

	holding := &models.Holding{
		PortfolioID:  portfolioID,
		SecurityID:   securityID,
		Quantity:     quantity,
		AveragePrice: price,
		TotalCost:    totalCost,
	}

	// это либо перезапишит либо создаст (on conflict)
	return s.holdingRepo.Create(ctx, holding)
}

func (s *investmentService) updateHoldingOnSell(ctx context.Context, portfolioID, securityID uuid.UUID, quantity decimal.Decimal) error {
	holding, err := s.holdingRepo.GetByPortfolioAndSecurity(ctx, portfolioID, securityID)
	if err != nil {
		return ErrInsufficientShares
	}

	if holding.Quantity.LessThan(quantity) {
		return ErrInsufficientShares
	}

	newQuantity := holding.Quantity.Sub(quantity)

	if newQuantity.IsZero() || newQuantity.LessThan(decimal.Zero) { // вообще отриц не должно быть прост на всякий
		return s.holdingRepo.DeleteIfZero(ctx, portfolioID, securityID)
	}

	costReduction := quantity.Div(holding.Quantity).Mul(holding.TotalCost)
	newTotalCost := holding.TotalCost.Sub(costReduction)
	newAvgPrice := holding.AveragePrice // средняя цена не меняется

	return s.holdingRepo.Update(ctx, holding.ID, newQuantity, newAvgPrice, newTotalCost)
}

func (s *investmentService) updateHoldingOnSplit(ctx context.Context, portfolioID, securityID uuid.UUID, ratio decimal.Decimal) error {
	holding, err := s.holdingRepo.GetByPortfolioAndSecurity(ctx, portfolioID, securityID)
	if err != nil {
		return err
	}

	newQuantity := holding.Quantity.Mul(ratio)
	newAvgPrice := holding.AveragePrice.Div(ratio)
	// Total cost не меняется

	return s.holdingRepo.Update(ctx, holding.ID, newQuantity, newAvgPrice, holding.TotalCost)
}

// revertBuyTransaction откатывает покупку (уменьшает холдинг)
func (s *investmentService) revertBuyTransaction(ctx context.Context, tx *models.InvestmentTransaction) error {
	holding, err := s.holdingRepo.GetByPortfolioAndSecurity(ctx, tx.PortfolioID, tx.SecurityID)
	if err != nil {
		// если холдинга нет, значит его уже удалили вручную - ничего не делаем
		return nil
	}

	// уменьшаем количество и себестоимость
	newQuantity := holding.Quantity.Sub(tx.Quantity)
	costReduction := tx.Amount // Amount включает цену + комиссию
	newTotalCost := holding.TotalCost.Sub(costReduction)

	if newQuantity.LessThanOrEqual(decimal.Zero) || newTotalCost.LessThanOrEqual(decimal.Zero) {
		// Если количество стало 0 или отрицательным - удаляем холдинг
		return s.holdingRepo.DeleteIfZero(ctx, tx.PortfolioID, tx.SecurityID)
	}

	// пересчитываем среднюю цену
	newAvgPrice := newTotalCost.Div(newQuantity)

	return s.holdingRepo.Update(ctx, holding.ID, newQuantity, newAvgPrice, newTotalCost)
}

// revertSellTransaction откатывает продажу (увеличивает холдинг)
func (s *investmentService) revertSellTransaction(ctx context.Context, tx *models.InvestmentTransaction) error {
	// обратная операция для продажи = добавить акции обратно
	// используем цену и комиссию из исходной транзакции
	return s.updateHoldingOnBuy(ctx, tx.PortfolioID, tx.SecurityID, tx.Quantity, tx.Price, tx.Commission)
}

// revertSplitTransaction откатывает сплит
func (s *investmentService) revertSplitTransaction(ctx context.Context, tx *models.InvestmentTransaction) error {
	holding, err := s.holdingRepo.GetByPortfolioAndSecurity(ctx, tx.PortfolioID, tx.SecurityID)
	if err != nil {
		return nil // Холдинг уже удалён
	}

	// Обратный сплит: если было split 2:1 (ratio=2), то откат = 1:2 (ratio=0.5)
	reverseRatio := decimal.NewFromInt(1).Div(tx.Quantity)
	newQuantity := holding.Quantity.Mul(reverseRatio)
	newAvgPrice := holding.AveragePrice.Div(reverseRatio)

	return s.holdingRepo.Update(ctx, holding.ID, newQuantity, newAvgPrice, holding.TotalCost)
}

func (s *investmentService) GetTransactions(ctx context.Context, portfolioID uuid.UUID, limit, offset int) ([]models.InvestmentTransaction, error) {
	return s.investmentRepo.GetByPortfolioID(ctx, portfolioID, limit, offset)
}

func (s *investmentService) GetTransactionsByDateRange(ctx context.Context, portfolioID uuid.UUID, start, end time.Time) ([]models.InvestmentTransaction, error) {
	return s.investmentRepo.GetByDateRange(ctx, portfolioID, start, end)
}

func (s *investmentService) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	// получаем транзакцию перед удалением для отката холдинга
	tx, err := s.investmentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// атомарная операция: удаление транзакции + откат холдинга
	return s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// удаляем транзакцию
		if err := s.investmentRepo.Delete(txCtx, id); err != nil {
			return err
		}

		// Откатываем изменения в холдинге в зависимости от типа транзакции
		switch tx.Type {
		case models.InvestmentTransactionTypeBuy:
			// обратная операция для покупки = продажа
			return s.revertBuyTransaction(txCtx, tx)
		case models.InvestmentTransactionTypeSell:
			// обратная операция для продажи = покупка
			return s.revertSellTransaction(txCtx, tx)
		case models.InvestmentTransactionTypeSplit:
			// обратная операция для сплита = обратный сплит
			return s.revertSplitTransaction(txCtx, tx)
		case models.InvestmentTransactionTypeDividend, models.InvestmentTransactionTypeCoupon:
			// дивиденды/купоны не влияют на холдинги
			return nil
		}
		return nil
	})
}

func (s *investmentService) GetHoldings(ctx context.Context, portfolioID uuid.UUID) ([]models.Holding, error) {
	holdings, err := s.holdingRepo.GetByPortfolioID(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	// обогащаем холдинги текущими котировками
	if err := s.enrichHoldings(ctx, holdings); err != nil {
		return holdings, nil // возвращаем без обогащения при ошибке
	}

	return holdings, nil
}

func (s *investmentService) GetHolding(ctx context.Context, portfolioID, securityID uuid.UUID) (*models.Holding, error) {
	holding, err := s.holdingRepo.GetByPortfolioAndSecurity(ctx, portfolioID, securityID)
	if err != nil {
		return nil, err
	}

	// обогащаем холдинг текущей котировкой
	holdings := []models.Holding{*holding}
	if err := s.enrichHoldings(ctx, holdings); err != nil {
		return holding, nil // возвращаем без обогащения при ошибке
	}

	enriched := holdings[0]
	return &enriched, nil
}

func (s *investmentService) GetPortfolioAnalytics(ctx context.Context, portfolioID uuid.UUID) (*models.PortfolioAnalytics, error) {
	holdings, err := s.holdingRepo.GetByPortfolioID(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	// обогащаем холдинги текущими котировками для расчета аналитики
	if err := s.enrichHoldings(ctx, holdings); err != nil {
		return nil, err
	}

	analytics := &models.PortfolioAnalytics{
		PortfolioID:          portfolioID,
		AllocationByType:     make(map[models.SecurityType]decimal.Decimal),
		AllocationBySector:   make(map[string]decimal.Decimal),
		AllocationByCurrency: make(map[string]decimal.Decimal),
	}

	var totalValue, totalInvested decimal.Decimal

	for _, h := range holdings {
		totalValue = totalValue.Add(h.CurrentValue)
		totalInvested = totalInvested.Add(h.TotalCost)

		if h.Security != nil {
			// Type allocation
			analytics.AllocationByType[h.Security.Type] = analytics.AllocationByType[h.Security.Type].Add(h.CurrentValue)

			// Sector allocation
			if h.Security.Sector != "" {
				analytics.AllocationBySector[h.Security.Sector] = analytics.AllocationBySector[h.Security.Sector].Add(h.CurrentValue)
			}

			// Currency allocation
			analytics.AllocationByCurrency[h.Security.Currency] = analytics.AllocationByCurrency[h.Security.Currency].Add(h.CurrentValue)
		}
	}

	analytics.TotalReturn = totalValue.Sub(totalInvested)
	if totalInvested.GreaterThan(decimal.Zero) {
		analytics.TotalReturnPct = analytics.TotalReturn.Div(totalInvested).Mul(decimal.NewFromInt(100))
	}

	// конвертим абсол значения в относительные
	if totalValue.GreaterThan(decimal.Zero) {
		for k, v := range analytics.AllocationByType {
			analytics.AllocationByType[k] = v.Div(totalValue).Mul(decimal.NewFromInt(100))
		}
		for k, v := range analytics.AllocationBySector {
			analytics.AllocationBySector[k] = v.Div(totalValue).Mul(decimal.NewFromInt(100))
		}
		for k, v := range analytics.AllocationByCurrency {
			analytics.AllocationByCurrency[k] = v.Div(totalValue).Mul(decimal.NewFromInt(100))
		}
	}

	// получаем дивиденды за прошлый год
	lastYear := time.Now().Year() - 1
	totalDividends, _ := s.investmentRepo.GetTotalDividends(ctx, portfolioID, lastYear)
	if totalValue.GreaterThan(decimal.Zero) {
		analytics.DividendYield = totalDividends.Div(totalValue).Mul(decimal.NewFromInt(100))
	}

	return analytics, nil
}

func (s *investmentService) GetTaxReport(ctx context.Context, portfolioID uuid.UUID, year int) (*models.TaxReport, error) {
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	// все транзакции за год
	transactions, err := s.investmentRepo.GetByDateRange(ctx, portfolioID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	report := &models.TaxReport{
		Year:        year,
		PortfolioID: portfolioID,
	}

	// собираем холдинги для расчёта себестоимости
	holdings, err := s.holdingRepo.GetByPortfolioID(ctx, portfolioID)
	if err != nil {
		holdings = []models.Holding{} // продолжаем без холдингов
	}
	holdingMap := make(map[uuid.UUID]*models.Holding)
	for i := range holdings {
		holdingMap[holdings[i].SecurityID] = &holdings[i]
	}

	for _, tx := range transactions {
		switch tx.Type {
		case models.InvestmentTransactionTypeDividend:
			report.TotalDividends = report.TotalDividends.Add(tx.Amount)
		case models.InvestmentTransactionTypeCoupon:
			report.TotalCoupons = report.TotalCoupons.Add(tx.Amount)
		case models.InvestmentTransactionTypeSell:
			// рассчитываем реализованную прибыль/убыток
			// выручка = Quantity × Price - Commission
			proceeds := tx.Quantity.Mul(tx.Price).Sub(tx.Commission)

			// себестоимость = Quantity × AveragePrice (на момент продажи)
			// используем текущий AveragePrice из холдинга как приближение
			var costBasis decimal.Decimal
			if holding, exists := holdingMap[tx.SecurityID]; exists {
				costBasis = tx.Quantity.Mul(holding.AveragePrice)
			} else {
				// если холдинга нет (продали всё), используем цену транзакции
				// это приближение, в реальности нужно хранить историю покупок и FIFO принцип
				costBasis = tx.Quantity.Mul(tx.Price)
			}

			// Прибыль/Убыток = Выручка - Себестоимость
			profitLoss := proceeds.Sub(costBasis)

			if profitLoss.GreaterThanOrEqual(decimal.Zero) {
				report.RealizedGains = report.RealizedGains.Add(profitLoss)
			} else {
				report.RealizedLosses = report.RealizedLosses.Add(profitLoss.Abs())
			}
		}
	}

	report.Transactions = transactions

	taxableIncome := report.TotalDividends.Add(report.TotalCoupons)
	if report.RealizedGains.GreaterThan(report.RealizedLosses) {
		report.NetGain = report.RealizedGains.Sub(report.RealizedLosses)
		taxableIncome = taxableIncome.Add(report.NetGain)
	}

	report.TaxableAmount = taxableIncome
	report.EstimatedTax = taxableIncome.Mul(decimal.NewFromFloat(0.13))

	return report, nil
}

func (s *investmentService) GetUpcomingDividends(ctx context.Context, portfolioID uuid.UUID) ([]models.Dividend, error) {
	// получаем все активы портфеля
	holdings, err := s.holdingRepo.GetByPortfolioID(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	var allDividends []models.Dividend

	// для каждой бумаги получаем дивиденды
	for _, h := range holdings {
		if h.Security == nil || h.Quantity.IsZero() {
			continue
		}

		// получаем дивиденды из API провайдера
		divs, err := s.marketProvider.GetDividends(ctx, h.Security.Ticker, h.Security.Exchange)
		if err != nil {
			// пропускаем при ошибке, продолжаем с другими
			continue
		}

		// подставляем SecurityID и Security
		for i := range divs {
			divs[i].SecurityID = h.Security.ID
			divs[i].Security = h.Security
		}

		allDividends = append(allDividends, divs...)
	}

	return allDividends, nil
}

// enrichHoldings обогащает холдинги текущими рыночными котировками
func (s *investmentService) enrichHoldings(ctx context.Context, holdings []models.Holding) error {
	if len(holdings) == 0 {
		return nil
	}

	// группируем тикеры по биржам
	type exchangeGroup struct {
		exchange models.Exchange
		tickers  []string
		indexes  []int // индексы холдингов
	}

	exchangeGroups := make(map[models.Exchange]*exchangeGroup)

	for i := range holdings {
		if holdings[i].Security == nil {
			continue
		}

		exchange := holdings[i].Security.Exchange
		if group, exists := exchangeGroups[exchange]; exists {
			group.tickers = append(group.tickers, holdings[i].Security.Ticker)
			group.indexes = append(group.indexes, i)
		} else {
			exchangeGroups[exchange] = &exchangeGroup{
				exchange: exchange,
				tickers:  []string{holdings[i].Security.Ticker},
				indexes:  []int{i},
			}
		}
	}

	// получаем котировки для каждой биржи
	allQuotes := make(map[string]*models.MarketQuote)
	for _, group := range exchangeGroups {
		quotes, err := s.marketProvider.GetQuotes(ctx, group.tickers, group.exchange)
		if err != nil {
			// Пропускаем ошибки для конкретной биржи, продолжаем с остальными
			continue
		}
		for ticker, quote := range quotes {
			allQuotes[ticker] = quote
		}
	}

	// рассчитываем общую стоимость портфеля для Weight
	var totalPortfolioValue decimal.Decimal
	for i := range holdings {
		if holdings[i].Security != nil {
			if quote, ok := allQuotes[holdings[i].Security.Ticker]; ok {
				currentValue := holdings[i].Quantity.Mul(quote.LastPrice)
				totalPortfolioValue = totalPortfolioValue.Add(currentValue)
			}
		}
	}

	// заполняем вычисляемые поля для каждого холдинга
	for i := range holdings {
		if holdings[i].Security == nil {
			continue
		}

		ticker := holdings[i].Security.Ticker
		quote, ok := allQuotes[ticker]
		if !ok {
			continue
		}

		// CurrentPrice - текущая рыночная цена
		holdings[i].CurrentPrice = quote.LastPrice

		// CurrentValue = Quantity × CurrentPrice
		holdings[i].CurrentValue = holdings[i].Quantity.Mul(quote.LastPrice)

		// Profit = CurrentValue - TotalCost
		holdings[i].Profit = holdings[i].CurrentValue.Sub(holdings[i].TotalCost)

		// ProfitPercent = (Profit / TotalCost) × 100
		if holdings[i].TotalCost.GreaterThan(decimal.Zero) {
			holdings[i].ProfitPercent = holdings[i].Profit.Div(holdings[i].TotalCost).Mul(decimal.NewFromInt(100))
		}

		// Weight = (CurrentValue / TotalPortfolioValue) × 100
		if totalPortfolioValue.GreaterThan(decimal.Zero) {
			holdings[i].Weight = holdings[i].CurrentValue.Div(totalPortfolioValue).Mul(decimal.NewFromInt(100))
		}
	}

	return nil
}
