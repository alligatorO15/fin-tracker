package service

import (
	"context"
	"sort"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/ai"
	"github.com/alligatorO15/fin-tracker/internal/config"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AnalyticsService interface {
	GetFinancialSummary(ctx context.Context, userID uuid.UUID, period models.Period, startDate, endDate *time.Time) (*models.FinancialSummary, error)
	GetCashFlowReport(ctx context.Context, userID uuid.UUID, period models.Period, startDate, endDate *time.Time) (*models.CashFlowReport, error)
	GetSpendingTrends(ctx context.Context, userID uuid.UUID, months int) ([]models.SpendingTrend, error)
	GetNetWorthReport(ctx context.Context, userID uuid.UUID) (*models.NetWorthReport, error)
	GetFinancialHealth(ctx context.Context, userID uuid.UUID) (*models.FinancialHealth, error)
	GetRecommendations(ctx context.Context, userID uuid.UUID) ([]models.Recommendation, error)
}

type analyticsService struct {
	repos  *repository.Repositories
	config *config.Config
	ai     *ai.OllamaClient
}

func NewAnalyticsService(repos *repository.Repositories, cfg *config.Config, aiClient *ai.OllamaClient) AnalyticsService {
	return &analyticsService{
		repos:  repos,
		config: cfg,
		ai:     aiClient,
	}
}

func (s *analyticsService) GetFinancialSummary(ctx context.Context, userID uuid.UUID, period models.Period, startDate, endDate *time.Time) (*models.FinancialSummary, error) {
	start, end := s.calculatePeriodDates(period, startDate, endDate)

	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	summary := &models.FinancialSummary{
		Period:    period,
		StartDate: start,
		EndDate:   end,
		Currency:  user.DefaultCurrency,
	}

	// get income/expenses by category
	incomeByCategory, _ := s.repos.Transaction.GetSumByCategory(ctx, userID, start, end, models.TransactionTypeIncome)
	expensesByCategory, _ := s.repos.Transaction.GetSumByCategory(ctx, userID, start, end, models.TransactionTypeExpense)

	categories, _ := s.repos.Category.GetByUserID(ctx, userID)
	categoryMap := make(map[uuid.UUID]models.Category)
	for _, c := range categories {
		categoryMap[c.ID] = c
	}

	for categoryID, amount := range incomeByCategory {
		summary.TotalIncome = summary.TotalIncome.Add(amount)
		cat := categoryMap[categoryID]
		summary.IncomeByCategory = append(summary.IncomeByCategory, models.CategoryAmount{
			CategoryID:   categoryID,
			CategoryName: cat.Name,
			CategoryIcon: cat.Icon,
			Amount:       amount,
		})
	}

	for categoryID, amount := range expensesByCategory {
		summary.TotalExpenses = summary.TotalExpenses.Add(amount)
		cat := categoryMap[categoryID]
		summary.ExpenseByCategory = append(summary.ExpenseByCategory, models.CategoryAmount{
			CategoryID:   categoryID,
			CategoryName: cat.Name,
			CategoryIcon: cat.Icon,
			Amount:       amount,
		})
	}

	for i := range summary.IncomeByCategory {
		if summary.TotalIncome.GreaterThan(decimal.Zero) {
			summary.IncomeByCategory[i].Percentage = summary.IncomeByCategory[i].Amount.Div(summary.TotalIncome).Mul(decimal.NewFromInt(100))
		}
	}
	for i := range summary.ExpenseByCategory {
		if summary.TotalExpenses.GreaterThan(decimal.Zero) {
			summary.ExpenseByCategory[i].Percentage = summary.ExpenseByCategory[i].Amount.Div(summary.TotalExpenses).Mul(decimal.NewFromInt(100))
		}
	}

	// сортируем по сумме в порядке убываняи
	sort.Slice(summary.IncomeByCategory, func(i, j int) bool {
		return summary.IncomeByCategory[i].Amount.GreaterThan(summary.IncomeByCategory[j].Amount)
	})
	sort.Slice(summary.ExpenseByCategory, func(i, j int) bool {
		return summary.ExpenseByCategory[i].Amount.GreaterThan(summary.ExpenseByCategory[j].Amount)
	})

	// вычисляем сбережения
	summary.NetSavings = summary.TotalIncome.Sub(summary.TotalExpenses)
	if summary.TotalIncome.GreaterThan(decimal.Zero) {
		summary.SavingsRate = summary.NetSavings.Div(summary.TotalIncome).Mul(decimal.NewFromInt(100))
	}

	accountSummary, _ := s.repos.Account.GetSummary(ctx, userID)
	if accountSummary != nil {
		summary.TotalBalance = accountSummary.TotalBalance
	}

	// сравннение с предыд периодом
	prevStart, prevEnd := s.calculatePreviousPeriod(start, end)
	prevIncome, _ := s.repos.Transaction.GetSumByCategory(ctx, userID, prevStart, prevEnd, models.TransactionTypeIncome)
	prevExpenses, _ := s.repos.Transaction.GetSumByCategory(ctx, userID, prevStart, prevEnd, models.TransactionTypeExpense)

	var prevTotalIncome, prevTotalExpenses decimal.Decimal
	for _, amount := range prevIncome {
		prevTotalIncome = prevTotalIncome.Add(amount)
	}
	for _, amount := range prevExpenses {
		prevTotalExpenses = prevTotalExpenses.Add(amount)
	}

	summary.IncomeChange = summary.TotalIncome.Sub(prevTotalIncome)
	summary.ExpenseChange = summary.TotalExpenses.Sub(prevTotalExpenses)
	if prevTotalIncome.GreaterThan(decimal.Zero) {
		summary.IncomeChangePct = summary.IncomeChange.Div(prevTotalIncome).Mul(decimal.NewFromInt(100))
	}
	if prevTotalExpenses.GreaterThan(decimal.Zero) {
		summary.ExpenseChangePct = summary.ExpenseChange.Div(prevTotalExpenses).Mul(decimal.NewFromInt(100))
	}

	return summary, nil
}

func (s *analyticsService) GetCashFlowReport(ctx context.Context, userID uuid.UUID, period models.Period, startDate, endDate *time.Time) (*models.CashFlowReport, error) {
	start, end := s.calculatePeriodDates(period, startDate, endDate)

	groupBy := "month"
	switch period {
	case models.PeriodWeek, models.PeriodMonth:
		groupBy = "day"
	case models.PeriodQuarter:
		groupBy = "week"
	}

	data, err := s.repos.Transaction.GetSumByPeriod(ctx, userID, start, end, groupBy)
	if err != nil {
		return nil, err
	}

	user, _ := s.repos.User.GetByID(ctx, userID)
	currency := s.config.DefaultCurrency
	if user != nil {
		currency = user.DefaultCurrency
	}

	report := &models.CashFlowReport{
		Period:   period,
		Currency: currency,
		Data:     data,
	}

	for _, cf := range data {
		report.TotalIn = report.TotalIn.Add(cf.Income)
		report.TotalOut = report.TotalOut.Add(cf.Expenses)
	}
	report.NetFlow = report.TotalIn.Sub(report.TotalOut)

	return report, nil
}

func (s *analyticsService) GetSpendingTrends(ctx context.Context, userID uuid.UUID, months int) ([]models.SpendingTrend, error) {
	if months <= 0 {
		months = 6
	}

	end := time.Now()
	start := end.AddDate(0, -months, 0)

	categories, err := s.repos.Category.GetByType(ctx, userID, models.CategoryTypeExpense)
	if err != nil {
		return nil, err
	}

	var trends []models.SpendingTrend

	for _, category := range categories {
		var total decimal.Decimal
		var points []models.TrendPoint

		for m := 0; m < months; m++ {
			monthStart := start.AddDate(0, m, 0)
			monthEnd := monthStart.AddDate(0, 1, -1)

			sums, _ := s.repos.Transaction.GetSumByCategory(ctx, userID, monthStart, monthEnd, models.TransactionTypeExpense)
			amount := sums[category.ID]

			points = append(points, models.TrendPoint{
				Period: monthStart.Format("2006-01"),
				Amount: amount,
			})
			total = total.Add(amount)
		}

		if total.IsZero() {
			continue
		}

		trend := models.SpendingTrend{
			CategoryID:   category.ID,
			CategoryName: category.Name,
			Data:         points,
			Average:      total.Div(decimal.NewFromInt(int64(months))),
		}

		// вычисляем тренды (сравниваем вторую половину периода с первым)
		if len(points) >= 2 {
			mid := len(points) / 2
			var firstHalf, secondHalf decimal.Decimal
			for i, p := range points {
				if i < mid {
					firstHalf = firstHalf.Add(p.Amount)
				} else {
					secondHalf = secondHalf.Add(p.Amount)
				}
			}

			if !firstHalf.IsZero() {
				change := secondHalf.Sub(firstHalf).Div(firstHalf).Mul(decimal.NewFromInt(100))
				trend.TrendPercent = change
				if change.GreaterThan(decimal.NewFromInt(10)) {
					trend.Trend = "increasing"
				} else if change.LessThan(decimal.NewFromInt(-10)) {
					trend.Trend = "decreasing"
				} else {
					trend.Trend = "stable"
				}
			}
		}

		trends = append(trends, trend)
	}

	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Average.GreaterThan(trends[j].Average)
	})

	return trends, nil
}

func (s *analyticsService) GetNetWorthReport(ctx context.Context, userID uuid.UUID) (*models.NetWorthReport, error) {
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	report := &models.NetWorthReport{
		Date:              time.Now(),
		Currency:          user.DefaultCurrency,
		AssetsByType:      make(map[string]decimal.Decimal),
		LiabilitiesByType: make(map[string]decimal.Decimal),
	}

	accounts, _ := s.repos.Account.GetByUserID(ctx, userID)
	for _, acc := range accounts {
		if !acc.IsActive {
			continue
		}
		if acc.Type == models.AccountTypeDebt || acc.Type == models.AccountTypeCredit {
			report.TotalLiabilities = report.TotalLiabilities.Add(acc.Balance.Abs())
			report.LiabilitiesByType[string(acc.Type)] = report.LiabilitiesByType[string(acc.Type)].Add(acc.Balance.Abs())
		} else {
			report.TotalAssets = report.TotalAssets.Add(acc.Balance)
			report.AssetsByType[string(acc.Type)] = report.AssetsByType[string(acc.Type)].Add(acc.Balance)
		}
	}

	portfolios, _ := s.repos.Portfolio.GetByUserID(ctx, userID)
	for _, p := range portfolios {
		holdings, _ := s.repos.Holding.GetByPortfolioID(ctx, p.ID)
		for _, h := range holdings {
			report.TotalAssets = report.TotalAssets.Add(h.CurrentValue)
			report.AssetsByType["investment"] = report.AssetsByType["investment"].Add(h.CurrentValue)
		}
	}

	report.NetWorth = report.TotalAssets.Sub(report.TotalLiabilities)
	return report, nil
}

func (s *analyticsService) GetFinancialHealth(ctx context.Context, userID uuid.UUID) (*models.FinancialHealth, error) {
	health := &models.FinancialHealth{}

	summary, _ := s.GetFinancialSummary(ctx, userID, models.PeriodMonth, nil, nil)

	// выставляем баллы по сбережениям
	if summary != nil && summary.TotalIncome.GreaterThan(decimal.Zero) {
		savingsRate := summary.NetSavings.Div(summary.TotalIncome).InexactFloat64()
		health.SavingsRate = decimal.NewFromFloat(savingsRate * 100)
		switch {
		case savingsRate >= 0.20:
			health.SavingsScore = 100
		case savingsRate >= 0.10:
			health.SavingsScore = 80
		case savingsRate >= 0.05:
			health.SavingsScore = 60
		case savingsRate > 0:
			health.SavingsScore = 40
		default:
			health.SavingsScore = 20
		}
	}

	// то же самое по бюджетам
	budgets, _ := s.repos.Budget.GetByUserID(ctx, userID, true)
	if len(budgets) > 0 {
		onTrack := 0
		for _, b := range budgets {
			if b.SpentPercent <= 100 {
				onTrack++
			}
		}
		health.BudgetScore = onTrack * 100 / len(budgets)
	} else {
		health.BudgetScore = 50
	}

	// по обязательствам
	netWorth, _ := s.GetNetWorthReport(ctx, userID)
	if netWorth != nil && summary != nil && summary.TotalIncome.GreaterThan(decimal.Zero) {
		monthlyDebt := netWorth.TotalLiabilities.Div(decimal.NewFromInt(12))
		debtToIncome := monthlyDebt.Div(summary.TotalIncome).InexactFloat64()
		health.DebtToIncomeRatio = decimal.NewFromFloat(debtToIncome * 100)
		switch {
		case debtToIncome <= 0.20:
			health.DebtScore = 100
		case debtToIncome <= 0.35:
			health.DebtScore = 80
		case debtToIncome <= 0.50:
			health.DebtScore = 60
		default:
			health.DebtScore = 40
		}
	} else {
		health.DebtScore = 80
	}

	// вычисление ликвидных активов (только кэш и счета)
	accounts, _ := s.repos.Account.GetByUserID(ctx, userID)
	var liquidAssets decimal.Decimal
	for _, acc := range accounts {
		if acc.Type == models.AccountTypeCash || acc.Type == models.AccountTypeBank {
			liquidAssets = liquidAssets.Add(acc.Balance)
		}
	}
	if summary != nil && summary.TotalExpenses.GreaterThan(decimal.Zero) {
		monthsCovered := liquidAssets.Div(summary.TotalExpenses).InexactFloat64()
		health.EmergencyFundMonths = decimal.NewFromFloat(monthsCovered)
		switch {
		case monthsCovered >= 6:
			health.EmergencyFundScore = 100
		case monthsCovered >= 3:
			health.EmergencyFundScore = 80
		case monthsCovered >= 1:
			health.EmergencyFundScore = 60
		default:
			health.EmergencyFundScore = 30
		}
	}

	// общая оценка (буквенная)
	health.OverallScore = (health.SavingsScore + health.BudgetScore + health.DebtScore + health.EmergencyFundScore) / 4

	switch {
	case health.OverallScore >= 90:
		health.Grade = "A"
	case health.OverallScore >= 80:
		health.Grade = "B"
	case health.OverallScore >= 70:
		health.Grade = "C"
	case health.OverallScore >= 60:
		health.Grade = "D"
	default:
		health.Grade = "F"
	}

	health.TopRecommendations, _ = s.GetRecommendations(ctx, userID)
	if len(health.TopRecommendations) > 3 {
		health.TopRecommendations = health.TopRecommendations[:3]
	}

	return health, nil
}

func (s *analyticsService) GetRecommendations(ctx context.Context, userID uuid.UUID) ([]models.Recommendation, error) {
	summary, _ := s.GetFinancialSummary(ctx, userID, models.PeriodMonth, nil, nil)
	budgets, _ := s.repos.Budget.GetByUserID(ctx, userID, true)
	user, _ := s.repos.User.GetByID(ctx, userID)

	currency := s.config.DefaultCurrency
	if user != nil {
		currency = user.DefaultCurrency
	}

	// пробуем получить ai рекомендации
	if s.ai != nil && s.ai.IsAvailable(ctx) {
		aiSummary := s.buildAISummary(summary, budgets, currency)
		if advice, err := s.ai.GetFinancialAdvice(ctx, aiSummary); err == nil && advice != "" {
			return []models.Recommendation{{
				ID:          uuid.New(),
				Type:        "ai",
				Priority:    5,
				Title:       "Персональные рекомендации",
				Description: advice,
				Impact:      "high",
			}}, nil
		}
	}

	// fallback: простые правила если ai недоступен (если вернет пустой срез фронт покажет что нибуль типо круто)
	return s.getBasicRecommendations(summary, budgets)
}

func (s *analyticsService) buildAISummary(summary *models.FinancialSummary, budgets []models.Budget, currency string) ai.FinancialSummary {
	aiSummary := ai.FinancialSummary{Currency: currency}

	if summary != nil {
		aiSummary.TotalIncome = summary.TotalIncome
		aiSummary.TotalExpenses = summary.TotalExpenses
		aiSummary.Balance = summary.TotalBalance
		aiSummary.SavingsRate = summary.SavingsRate

		for _, cat := range summary.ExpenseByCategory {
			aiSummary.TopCategories = append(aiSummary.TopCategories, ai.CategorySpending{
				Name:   cat.CategoryName,
				Amount: cat.Amount,
			})
		}
	}

	for _, b := range budgets {
		aiSummary.BudgetStatus = append(aiSummary.BudgetStatus, ai.BudgetStatus{
			Category: b.Name,
			Limit:    b.Amount,
			Spent:    b.Spent,
			Percent:  decimal.NewFromFloat(b.SpentPercent),
		})
	}

	return aiSummary
}

func (s *analyticsService) getBasicRecommendations(summary *models.FinancialSummary, budgets []models.Budget) ([]models.Recommendation, error) {
	var recs []models.Recommendation

	// проверка нормы сбережений
	if summary != nil && summary.TotalIncome.GreaterThan(decimal.Zero) {
		if summary.SavingsRate.LessThan(decimal.NewFromInt(10)) {
			recs = append(recs, models.Recommendation{
				ID:          uuid.New(),
				Type:        "savings",
				Priority:    5,
				Title:       "Увеличьте норму сбережений",
				Description: "Рекомендуется откладывать минимум 10% дохода.",
				Impact:      "high",
			})
		}
	}

	// проверка бюджетов
	for _, b := range budgets {
		if b.SpentPercent >= 90 {
			recs = append(recs, models.Recommendation{
				ID:          uuid.New(),
				Type:        "budget",
				Priority:    4,
				Title:       "Бюджет «" + b.Name + "» близок к лимиту",
				Description: "Израсходовано более 90%.",
				Impact:      "medium",
			})
		}
	}

	return recs, nil
}

func (s *analyticsService) calculatePeriodDates(period models.Period, startDate, endDate *time.Time) (time.Time, time.Time) {
	now := time.Now()

	if startDate != nil && endDate != nil {
		return *startDate, *endDate
	}

	switch period {
	case models.PeriodDay:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, now
	case models.PeriodWeek:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := now.AddDate(0, 0, -weekday+1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		return start, now
	case models.PeriodMonth:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), now
	case models.PeriodQuarter:
		quarter := (int(now.Month()) - 1) / 3
		return time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location()), now
	case models.PeriodYear:
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()), now
	case models.PeriodAll:
		return time.Date(2000, 1, 1, 0, 0, 0, 0, now.Location()), now
	default:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), now
	}
}

func (s *analyticsService) calculatePreviousPeriod(currentStart, currentEnd time.Time) (time.Time, time.Time) {
	duration := currentEnd.Sub(currentStart)
	prevEnd := currentStart.Add(-time.Second)
	prevStart := prevEnd.Add(-duration)
	return prevStart, prevEnd
}
