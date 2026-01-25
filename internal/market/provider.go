package market

import (
	"context"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/shopspring/decimal"
)

// MarketProvider определяет интерфейс для поставщиков рыночных данных
//
//	чтобы получать актуальные финансовые данные из внешних источников (биржи, API, провайдеры).
type MarketProvider interface {
	// GetName возвращает название поставщика данных(провайдера)
	GetName() string

	// GetSupportedExchanges возвращает список поддерживаемых бирж
	GetSupportedExchanges() []models.Exchange

	// IsEnabled проверяет, включен ли поставщик данных
	IsEnabled() bool

	// GetQuote получает текущую котировку ценной бумаги
	GetQuote(ctx context.Context, ticker string, exchange models.Exchange) (*models.MarketQuote, error)

	// то же только для нескольких бумаг
	GetQuotes(ctx context.Context, tickers []string, exchange models.Exchange) (map[string]*models.MarketQuote, error)

	// SearchSecurities ищет ценные бумаги по названию или тикеру(query)
	SearchSecurities(ctx context.Context, query string, securityType *models.SecurityType) ([]models.Security, error)

	// GetSecurityInfo получает подробную информацию о ценной бумаге
	GetSecurityInfo(ctx context.Context, ticker string, exchange models.Exchange) (*models.Security, error)

	// GetPriceHistory получает исторические данные цен
	//  для построения графиков и технического анализа.
	GetPriceHistory(ctx context.Context, ticker string, exchange models.Exchange, from, to time.Time) ([]PriceBar, error)

	// GetDividends получает историю дивидендных выплат
	GetDividends(ctx context.Context, ticker string, exchange models.Exchange) ([]models.Dividend, error)

	// GetCurrencyRate получает курс валюты
	GetCurrencyRate(ctx context.Context, from, to string) (decimal.Decimal, error)
}

// PriceBar представляет данные свечи OHLCV (цена открытия, максимум, минимум, закрытия, объем)
type PriceBar struct {
	Date   time.Time       `json:"date"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume int64           `json:"volume"`
}

// BrokerStatementParser определяет интерфейс для парсинга выписок брокеров
type BrokerStatementParser interface {
	// GetBrokerName возвращает название брокера
	GetBrokerName() string

	// GetSupportedFormats возвращает поддерживаемые форматы файлов
	GetSupportedFormats() []string

	// Parse разбирает файл выписки брокера и возвращает транзакции
	Parse(ctx context.Context, fileContent []byte, filename string) (*BrokerStatementResult, error)
}

// структура, содержащая ВСЕ данные, извлеченные из файла выписки брокера.
type BrokerStatementResult struct {
	BrokerName    string                         `json:"broker_name"`
	AccountNumber string                         `json:"account_number"`
	PeriodStart   time.Time                      `json:"period_start"`
	PeriodEnd     time.Time                      `json:"period_end"`
	Transactions  []models.InvestmentTransaction `json:"transactions"`
	CashFlows     []CashFlowEntry                `json:"cash_flows"`
	Errors        []string                       `json:"errors"`
	Warnings      []string                       `json:"warnings"`
}

// Пополнение/снятие денег, комиссии, налоги, проценты. Операции на брок счете, не связаные с инвестиционными транзакциями
type CashFlowEntry struct {
	Date        time.Time       `json:"date"`
	Type        string          `json:"type"`
	Amount      decimal.Decimal `json:"amount"`
	Currency    string          `json:"currency"`
	Description string          `json:"description"`
}
