package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Exchange string

// биржи
const (
	//российские
	ExchangeMOEX   Exchange = "MOEX"
	ExchangeCRYPTO Exchange = "CRYPTO"
)

// типы ценных бумаг
type SecurityType string

const (
	SecurityTypeStock      SecurityType = "stock"
	SecurityTypeBond       SecurityType = "bond"
	SecurityTypeETF        SecurityType = "etf"
	SecurityTypeMutualFund SecurityType = "mutual_fund" //пифы
	SecurityTypeCrypto     SecurityType = "crypto"
	SecurityTypeCurrency   SecurityType = "currency"   //валютные пары
	SecurityTypeDerivative SecurityType = "derivative" //производные бумаги(фьючерсы, опционы)
)

type Security struct {
	ID                uuid.UUID       `json:"id" db:"id"`
	Ticker            string          `json:"ticker" db:"ticker"` // биржевой тикер(уникален в рамках биржи): "GAZP" (Газпром), "SBER" (Сбербанк)
	ISIN              string          `json:"isin" db:"isin"`     //международный код ценной бумаги(уникален глобально): US0378331005 (Apple)
	Name              string          `json:"name" db:"name"`
	ShortName         string          `json:"short_name" db:"short_name"`
	Type              SecurityType    `json:"type" db:"type"`
	Exchange          Exchange        `json:"exchange" db:"exchange"`
	Country           string          `json:"country" db:"country"`
	Currency          string          `json:"currency" db:"currency"`
	Sector            string          `json:"sector" db:"sector"`
	Industry          string          `json:"industry" db:"industry"`
	LotSize           int             `json:"lot_size" db:"lot_size"`                       //мин кол-во с которого можно купить
	MinPriceIncrement decimal.Decimal `json:"min_price_increment" db:"min_price_increment"` //шаг изменения цены бумаги(устанаваливает биржа)
	IsActive          bool            `json:"is_active" db:"is_active"`
	//для обигаций bond
	FaceValue    *decimal.Decimal `json:"face_value" db:"face_value"`       //ном стоимость для облигаций, nil для других фин инструментов
	CouponRate   *decimal.Decimal `json:"coupon_rate" db:"coupon_rate"`     //ставка купона
	MaturityDate *time.Time       `json:"maturity_date" db:"maturity_date"` //дата погашения
	CouponFreq   *int             `json:"coupon_freq" db:"coupon_freq"`     //частота выплата купонов в год
	// для etf
	ExpenseRatio *decimal.Decimal `json:"expense_ration" db:"expense_ration"` //комиссия фонда в %
	//Рыночные данные
	LastPrice          decimal.Decimal `json:"last_price" db:"last_price"`                     //последняя цена сделки
	PriceChange        decimal.Decimal `json:"price_change" db:"price_change"`                 //изменение цены с пред закрытия
	PriceChangePercent decimal.Decimal `json:"price_change_percent" db:"price_change_percent"` // изменение в %
	Volume             int64           `json:"volume" db:"volume"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

// Portfolio представляет инвестиционный портфель пользователя
// Может быть несколько портфелей у одного пользователя
type Portfolio struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	AccountID     *uuid.UUID `json:"account_id" db:"account_id"` //привязка к счету если nil виртуальный портфель
	Name          string     `json:"name" db:"name"`             //наше навзание портфеля
	Description   string     `json:"description" db:"description"`
	Currency      string     `json:"currency" db:"currency"`             //базовая валюта портфеля(в котором ведется учет)
	BrokerName    string     `json:"broker_name" db:"broker_name"`       //брокер
	BrokerAccount string     `json:"broker_account" db:"broker_account"` //счет у брокера
	IsActive      bool       `json:"is_active" db:"is_active"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	//вычисляются на лету
	TotalValue    decimal.Decimal `json:"total_value" db:"-"`        //полная стоимость портфеля
	TotalInvested decimal.Decimal `json:"total_invested" db:"-"`     // стоимость вложений
	TotalProfit   decimal.Decimal `json:"total_profit" db:"-"`       // totalvalue-totalinvested
	ProfitPercent decimal.Decimal `json:"profit_percent" db:"-"`     //прибыль в процентах
	Holdings      []Holding       `json:"holdings,omitempty" db:"-"` //позиции портфеля(заполняется при join)
}

type PortfolioCreate struct {
	AccountID     *uuid.UUID `json:"account_id"`
	Name          string     `json:"name" binding:"required"`
	Description   string     `json:"description"`
	Currency      string     `json:"currency" binding:"required"` //обязательное(для конвертаации)
	BrokerName    string     `json:"broker_name"`
	BrokerAccount string     `json:"broker_account"`
}

type PortfolioUpdate struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	BrokerName    *string `json:"broker_name"`
	BrokerAccount *string `json:"broker_account"`
	IsActive      *bool   `json:"is_active"`
}

// представляет позицию в портфеле
type Holding struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	PortfolioID  uuid.UUID       `json:"portfolio_id" db:"portfolio_id"`
	SecurityID   uuid.UUID       `json:"security_id" db:"security_id"`
	Quantity     decimal.Decimal `json:"quantity" db:"quantity"`
	AveragePrice decimal.Decimal `json:"average_price" db:"average_price"` //средняя цена
	TotalCost    decimal.Decimal `json:"total_cost" db:"total_cost"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`

	// Вычисляемые поля
	CurrentPrice  decimal.Decimal `json:"current_price" db:"-"`  // подгружается из Security.LastPrice
	CurrentValue  decimal.Decimal `json:"current_value" db:"-"`  // = Quantity × CurrentPrice
	Profit        decimal.Decimal `json:"profit" db:"-"`         // = CurrentValue - TotalCost
	ProfitPercent decimal.Decimal `json:"profit_percent" db:"-"` // в % = (Profit / TotalCost) × 100
	Weight        decimal.Decimal `json:"weight" db:"-"`         // доля в портфеле (CurrentValue / PortfolioTotalValue) × 100
	Security      *Security       `json:"security,omitempty"`    // полные данные по каждой бумаге
}

func (h *Holding) CalculateValues() {
	if h.Security != nil {
		h.CurrentPrice = h.Security.LastPrice
		h.CurrentValue = h.Quantity.Mul(h.CurrentPrice)
		h.Profit = h.CurrentValue.Sub(h.TotalCost)

		if h.TotalCost.GreaterThan(decimal.Zero) {
			h.ProfitPercent = h.Profit.Div(h.TotalCost).Mul(decimal.NewFromInt(100))
		}
	}
}

// InvestmentTransaction представляет операцию на бирже
type InvestmentTransactionType string

const (
	InvestmentTransactionTypeBuy         InvestmentTransactionType = "buy"          // покупка бумаг
	InvestmentTransactionTypeSell        InvestmentTransactionType = "sell"         // продажа бумаг
	InvestmentTransactionTypeDividend    InvestmentTransactionType = "dividend"     // получение дивидендов
	InvestmentTransactionTypeCoupon      InvestmentTransactionType = "coupon"       // получение купона по облигации
	InvestmentTransactionTypeSplit       InvestmentTransactionType = "split"        // сплит (дробление) акций
	InvestmentTransactionTypeTransferIn  InvestmentTransactionType = "transfer_in"  // ввод бумаг со счета другого брокера
	InvestmentTransactionTypeTransferOut InvestmentTransactionType = "transfer_out" // вывод бумаг на счет другого брокера
	InvestmentTransactionTypeFee         InvestmentTransactionType = "fee"          // комиссия брокера/биржи
	InvestmentTransactionTypeTax         InvestmentTransactionType = "tax"          // удержание налога (например, налог на дивиденды)
)

// представляет биржевую сделку
type InvestmentTransaction struct {
	ID           uuid.UUID                 `json:"id" db:"id"`
	PortfolioID  uuid.UUID                 `json:"portfolio_id" db:"portfolio_id"` //порфтель к которому относится сделка
	SecurityID   uuid.UUID                 `json:"security_id" db:"security_id"`
	Type         InvestmentTransactionType `json:"type" db:"type"`
	Date         time.Time                 `json:"date" db:"date"`         // дата и время сделки (по биржевому времени)
	Quantity     decimal.Decimal           `json:"quantity" db:"quantity"` // количество бумаг
	Price        decimal.Decimal           `json:"price" db:"price"`
	Amount       decimal.Decimal           `json:"amount" db:"amount"`               // сумма операции = Quantity × Price (+/- комиссии)(для дивидендов/купонов - сумма выплаты)
	Commission   decimal.Decimal           `json:"commission" db:"commission"`       // комиссия брокера
	Currency     string                    `json:"currency" db:"currency"`           // валюта операции
	ExchangeRate decimal.Decimal           `json:"exchange_rate" db:"exchange_rate"` // курс конвертации в валюту портфеля
	Notes        string                    `json:"notes" db:"notes"`                 // заметки пользователя
	BrokerRef    string                    `json:"broker_ref" db:"broker_ref"`       // референс из выписки брокера(ункальный идентификатор)(для сверки)
	CreatedAt    time.Time                 `json:"created_at" db:"created_at"`
	Security     *Security                 `json:"security,omitempty"`
}

type InvestmentTransactionCreate struct {
	PortfolioID  uuid.UUID                 `json:"portfolio_id" binding:"required"`
	SecurityID   uuid.UUID                 `json:"security_id" binding:"required"`
	Type         InvestmentTransactionType `json:"type" binding:"required"`
	Date         time.Time                 `json:"date" binding:"required"`
	Quantity     decimal.Decimal           `json:"quantity" binding:"required"`
	Price        decimal.Decimal           `json:"price" binding:"required"`
	Commission   decimal.Decimal           `json:"commission"`
	Currency     string                    `json:"currency"`
	ExchangeRate decimal.Decimal           `json:"exchange_rate"`
	Notes        string                    `json:"notes"`
}

// представляет дивидендную выплату
type Dividend struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	SecurityID   uuid.UUID       `json:"security_id" db:"security_id"`
	ExDate       time.Time       `json:"ex_date" db:"ex_date"`             // кто купил акции после этой даты, не получит дивиденды
	PaymentDate  time.Time       `json:"payment_date" db:"payment_date"`   // дата фактической выплаты дивидендов на счет
	RecordDate   time.Time       `json:"record_date" db:"record_date"`     // дата фиксации(закрытия) реестра (после exdate фиксирует всех кто получит дивиденды)
	Amount       decimal.Decimal `json:"amount" db:"amount"`               // ден сумма дивидендов на одну акцию
	Currency     string          `json:"currency" db:"currency"`           // валюта выплаты
	DividendType string          `json:"dividend_type" db:"dividend_type"` // тип: regular (регулярные), special (специальные)
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	Security     *Security       `json:"security,omitempty"`
}

// PortfolioAnalytics содержит аналитику по портфелю
// рассчитывается на основе данных портфеля и рыночной информации. Это не аналитика личных финансов поэтому здесь оставил.
type PortfolioAnalytics struct {
	PortfolioID    uuid.UUID       `json:"portfolio_id"`
	TotalReturn    decimal.Decimal `json:"total_return"`         // совокупная доходность за все время
	TotalReturnPct decimal.Decimal `json:"total_return_percent"` // совокупная доходность в %
	// доходности = (доходность за сегодня - доходность_n_период_назад)/ доходность_n_период_назад
	DailyReturn   decimal.Decimal `json:"daily_return"` // доходность за сегодня
	WeeklyReturn  decimal.Decimal `json:"weekly_return"`
	MonthlyReturn decimal.Decimal `json:"monthly_return"`
	YearlyReturn  decimal.Decimal `json:"yearly_return"`

	// --- Метрики риска ---
	Volatility  decimal.Decimal `json:"volatility"`   //насколько сильно "скачет" стоимость портфеля. Чем выше, тем рискованнее портфель
	SharpeRatio decimal.Decimal `json:"sharpe_ratio"` // коэффициент Шарпа = (доходность - безрисковая ставка) / волатильность
	//  >1 хорошо, >2 отлично
	//например: sharpe = 0.5 = за каждый 1% риска получаете 0.5% дополнительной доходности
	MaxDrawdown decimal.Decimal `json:"max_drawdown"` // максимальная просадка (максимальная потеря от пика до дна)
	// например: -15% означает что портфель падал на 15% от максимума
	Beta decimal.Decimal `json:"beta"` // бета - мера корреляции с рынком
	// 1 = движется как рынок, <1 = менее волатильный, >1 = более волатильный

	// --- Аллокация (распределение активов) ---
	AllocationByType     map[SecurityType]decimal.Decimal `json:"allocation_by_type"`     // распределение по типам: 60% акции, 30% облигации, 10% ETF
	AllocationBySector   map[string]decimal.Decimal       `json:"allocation_by_sector"`   // распределение по секторам: 30% IT, 25% Финансы, 20% Энергетика
	AllocationByCurrency map[string]decimal.Decimal       `json:"allocation_by_currency"` // валютная диверсификация: 70% RUB, 20% USD, 10% EUR

	// --- Доходность ---
	DividendYield decimal.Decimal `json:"dividend_yield"` // дивидендная доходность портфеля в %
	//DividendYield = (Сумма всех дивидендов за год) / (Текущая стоимость портфеля) × 100%
	ExpectedDividends []Dividend `json:"expected_dividends"` // ожидаемые дивидендные выплаты

	// --- История стоимости ---
	ValueHistory []PortfolioValuePoint `json:"value_history"`
	// История изменения стоимости портфеля во времени
	// Для построения графиков
}

// Точка на графике стоимости портфеля
type PortfolioValuePoint struct {
	Date  time.Time       `json:"date"`
	Value decimal.Decimal `json:"value"`
}

// представляет налоговый отчет
// важно для декларации 3-НДФЛ в России
type TaxReport struct {
	Year           int             `json:"year"` // Налоговый год
	PortfolioID    uuid.UUID       `json:"portfolio_id"`
	TotalDividends decimal.Decimal `json:"total_dividends"` // cумма всех полученных дивидендов
	TotalCoupons   decimal.Decimal `json:"total_coupons"`   // cумма всех полученных купонов по облигациям
	RealizedGains  decimal.Decimal `json:"realized_gains"`  // реализованная прибыль (от продажи бумаг)
	RealizedLosses decimal.Decimal `json:"realized_losses"` // реализованные убытки (от продажи бумаг)
	NetGain        decimal.Decimal `json:"net_gain"`        // чистый финансовый результат = RealizedGains - RealizedLosses
	TaxableAmount  decimal.Decimal `json:"taxable_amount"`  // налогооблагаемая сумма. В РФ: дивиденды + купоны + прибыль от продаж (TaxableAmount = TotalDividends + TotalCoupons + NetGain)
	EstimatedTax   decimal.Decimal `json:"estimated_tax"`   // это уже рассчитанная сумма налога к уплате.

	//Доп детали
	Transactions     []InvestmentTransaction `json:"transactions"`      // сделки за год (для проверки)
	DividendPayments []Dividend              `json:"dividend_payments"` // дивидендные выплаты за год
}

// представляет импорт выписки от брокера
// автоматизация загрузки сделок из CSV/Excel файлов
type BrokerStatementImport struct {
	ID          uuid.UUID `json:"id" db:"id"`
	PortfolioID uuid.UUID `json:"portfolio_id" db:"portfolio_id"`
	BrokerType  string    `json:"broker_type" db:"broker_type"` // тип брокера: "sber", "tinkoff", "vtb", "alfa"(определяет парсер для файла)
	FileName    string    `json:"file_name" db:"file_name"`     // имя импортированного файла
	ImportDate  time.Time `json:"import_date" db:"import_date"`

	//Даты которые покрывает ввыпсика
	PeriodStart          time.Time `json:"period_start" db:"period_start"`
	PeriodEnd            time.Time `json:"period_end" db:"period_end"`
	Status               string    `json:"status" db:"status"`                               // статус: "pending", "processing", "completed", "failed"
	ErrorMessage         string    `json:"error_message" db:"error_message"`                 // cообщение об ошибке если импорт не удался
	TransactionsImported int       `json:"transactions_imported" db:"transactions_imported"` // cколько сделок импортировано из файла выписки
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

// представляет рыночные котировки в реальном времени
// получаются из внешних источников (MOEX API, Yahoo Finance и т.д.)
type MarketQuote struct {
	SecurityID    uuid.UUID       `json:"security_id"`
	Ticker        string          `json:"ticker"`
	Exchange      Exchange        `json:"exchange"`
	LastPrice     decimal.Decimal `json:"last_price"`     // последняя цена сделки
	Open          decimal.Decimal `json:"open"`           // цена открытия сессии
	High          decimal.Decimal `json:"high"`           // максимальная цена за сессию
	Low           decimal.Decimal `json:"low"`            // минимальная цена за сессию
	Close         decimal.Decimal `json:"close"`          // цена закрытия предыдущей сессии
	Volume        int64           `json:"volume"`         // объем торгов в штуках
	Change        decimal.Decimal `json:"change"`         // абсолютное изменение = LastPrice - Close
	ChangePercent decimal.Decimal `json:"change_percent"` // относительное изменение в %
	Bid           decimal.Decimal `json:"bid"`            // лучшая цена покупки (сколько покупатели готовы заплатить(макс))
	Ask           decimal.Decimal `json:"ask"`            // лучшая цена продажи (сколько продавцы просят(мин))
	// Spread = Ask - Bid (спред)
	Timestamp time.Time `json:"timestamp"` // время получения котировки
}
