package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Period определяет временные интервалы для отчетов
type Period string

const (
	PeriodDay     Period = "day"     // Дневной отчет
	PeriodWeek    Period = "week"    // Недельный отчет (понедельник-воскресенье)
	PeriodMonth   Period = "month"   // Месячный отчет (календарный месяц)
	PeriodQuarter Period = "quarter" // Квартальный отчет (3 месяца)
	PeriodYear    Period = "year"    // Годовой отчет (январь-декабрь)
	PeriodAll     Period = "all"     // За все время
)

// предоставляет общий финансовый обзор пользователя
type FinancialSummary struct {
	Period        Period          `json:"period"`         // какой период анализируем
	StartDate     time.Time       `json:"start_date"`     // начальная дата периода
	EndDate       time.Time       `json:"end_date"`       // конечная дата периода
	Currency      string          `json:"currency"`       // валюта отчета (может отличаться от валют транзакций)
	TotalIncome   decimal.Decimal `json:"total_income"`   // общая сумма доходов за период
	TotalExpenses decimal.Decimal `json:"total_expenses"` // общая сумма расходов за период
	NetSavings    decimal.Decimal `json:"net_savings"`    // чистые сбережения = TotalIncome - TotalExpenses
	SavingsRate   decimal.Decimal `json:"savings_rate"`   // норма сбережений в % = (NetSavings / TotalIncome) × 100
	// Финансовый индикатор: >20% хорошо, <10% плохо
	TotalBalance     decimal.Decimal  `json:"total_balance"`      // общий баланс по всем счетам на конец периода
	IncomeChange     decimal.Decimal  `json:"income_change"`      // абсолютное изменение доходов: текущий - предыдущий период
	ExpenseChange    decimal.Decimal  `json:"expense_change"`     // абсолютное изменение расходов
	IncomeChangePct  decimal.Decimal  `json:"income_change_pct"`  // относительное изменение доходов в %
	ExpenseChangePct decimal.Decimal  `json:"expense_change_pct"` // относительное изменение расходов в %
	IncomeByCategory []CategoryAmount `json:"income_by_category"`
	// Список категорий доходов с суммами
	// Пример: Зарплата (80%), Фриланс (15%), Дивиденды (5%)
	ExpenseByCategory []CategoryAmount `json:"expense_by_category"`
	// Список категорий расходов с суммами
	// Пример: Продукты (30%), Аренда (25%), Транспорт (15%)
}

// представляет сумму по категории
// Используется для построения круговых диаграмм и анализа структуры
type CategoryAmount struct {
	CategoryID   uuid.UUID       `json:"category_id"`   // id категории для навигации
	CategoryName string          `json:"category_name"` // название категории: "Продукты", "Транспорт", "Развлечения"
	CategoryIcon string          `json:"category_icon"` // иконка категории для UI
	Amount       decimal.Decimal `json:"amount"`        // cумма по этой категории
	Percentage   decimal.Decimal `json:"percentage"`    // доля в общем объеме в % = (Amount / Total) × 100
	Count        int             `json:"count"`         // rоличество транзакций в этой категории
}

// представляет данные о движении денежных средств
type CashFlow struct {
	Period   string          `json:"period"`   // Период (форматированный): "Янв 2024", "Неделя 1", "Q1"
	Income   decimal.Decimal `json:"income"`   // Приток денег за период
	Expenses decimal.Decimal `json:"expenses"` // Отток денег за период
	Net      decimal.Decimal `json:"net"`      // Чистый денежный поток = Income - Expenses
	// Положительный = деньги остаются, Отрицательный = убыток
}

// представляет полный отчет о денежных потоках
type CashFlowReport struct {
	Period   Period          `json:"period"`    // Базовый период (month, quarter, year)
	Currency string          `json:"currency"`  // Валюта отчета
	Data     []CashFlow      `json:"data"`      // Список потоков по подпериодам
	TotalIn  decimal.Decimal `json:"total_in"`  // Общий приток за весь период
	TotalOut decimal.Decimal `json:"total_out"` // Общий отток за весь период
	NetFlow  decimal.Decimal `json:"net_flow"`  // Общий чистый поток = TotalIn - TotalOut
}

// показывает динамику расходов по времени
type SpendingTrend struct {
	CategoryID   uuid.UUID       `json:"category_id"`
	CategoryName string          `json:"category_name"`
	Data         []TrendPoint    `json:"data"`          // Точки данных во времени
	Average      decimal.Decimal `json:"average"`       // Среднемесячные расходы по категории
	Trend        string          `json:"trend"`         // Направление тренда
	TrendPercent decimal.Decimal `json:"trend_percent"` // Изменение тренда в % за последний период
}

// представляет точку на графике тренда
type TrendPoint struct {
	Period string          `json:"period"`
	Amount decimal.Decimal `json:"amount"` // Сумма за этот период
}

// показывает отчет о чистом капитале
type NetWorthReport struct {
	Date              time.Time                  `json:"date"`                // Дата отчета (обычно на конец месяца)
	Currency          string                     `json:"currency"`            // Валюта отчета
	TotalAssets       decimal.Decimal            `json:"total_assets"`        // Общая стоимость активов
	TotalLiabilities  decimal.Decimal            `json:"total_liabilities"`   // Общая сумма обязательств (долгов)
	NetWorth          decimal.Decimal            `json:"net_worth"`           // Чистый капитал = TotalAssets - TotalLiabilities
	AssetsByType      map[string]decimal.Decimal `json:"assets_by_type"`      // Распределение активов по типам:
	LiabilitiesByType map[string]decimal.Decimal `json:"liabilities_by_type"` // Распределение долгов по типам
}

// представляет финансовую рекомендацию
type Recommendation struct {
	ID           uuid.UUID       `json:"id"`
	Type         string          `json:"type"` // Тип рекомендации:
	Priority     int             `json:"priority"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`             // Детальное описание: "Вы тратите 30% дохода на рестораны. Попробуйте готовить дома."
	CurrentValue decimal.Decimal `json:"current_value,omitempty"` // Текущее значение (если применимо)
	// Для рекомендации по бюджету: текущие траты 25000 руб
	TargetValue decimal.Decimal `json:"target_value,omitempty"` // Целевое значение (если применимо)
	// Для рекомендации по бюджету: рекомендуемые траты 15000 руб
	Impact string `json:"impact"` // Насколько сильно это повлияет на финансы
}

// FinancialHealth предоставляет общую оценку финансового здоровья
type FinancialHealth struct {
	OverallScore        int              `json:"overall_score"`         // Итоговый балл финансового здоровья
	Grade               string           `json:"grade"`                 // A, B, C, D, F
	SavingsScore        int              `json:"savings_score"`         // Оценка сбережений (норма сбережений, наличие "подушки")
	BudgetScore         int              `json:"budget_score"`          // Оценка управления бюджетом (соответствие плану, контроль расходов)
	DebtScore           int              `json:"debt_score"`            // Оценка долговой нагрузки (соотношение долг/доход, виды долгов)
	EmergencyFundScore  int              `json:"emergency_fund_score"`  // Оценка резервного фонда (наличие и достаточность)
	SavingsRate         decimal.Decimal  `json:"savings_rate"`          // норма сбережений в %
	DebtToIncomeRatio   decimal.Decimal  `json:"debt_to_income_ratio"`  // Коэффициент долговой нагрузки = (Ежемесячные платежи по долгам / Ежемесячный доход) × 100
	EmergencyFundMonths decimal.Decimal  `json:"emergency_fund_months"` // На сколько месяцев хватит резервного фонда = (Резервный фонд / Среднемесячные расходы)
	TopRecommendations  []Recommendation `json:"top_recommendations"`
}
