package market

import (
	"context"
	"fmt"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/config"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/shopspring/decimal"
)

// MultiProvider агрегирует несколько провайдеров рыночных данных
type MultiProvider struct {
	providers map[models.Exchange]MarketProvider
	config    *config.Config
}

// NewMultiProvider создаёт новый экземпляр мульти-провайдера
func NewMultiProvider(cfg *config.Config) *MultiProvider {
	mp := &MultiProvider{
		providers: make(map[models.Exchange]MarketProvider),
		config:    cfg,
	}

	// Регистрация провайдера MOEX (российский рынок — основной)
	if cfg.MOEXEnabled {
		moexProvider := NewMOEXProvider(cfg.MOEXApiURL)
		for _, exchange := range moexProvider.GetSupportedExchanges() {
			mp.providers[exchange] = moexProvider
		}
	}

	// Регистрация крипто-провайдера (всегда доступен)
	cryptoProvider := NewCryptoProvider()
	mp.providers[models.ExchangeCRYPTO] = cryptoProvider

	return mp
}

// GetProvider возвращает подходящий провайдер для биржи
func (mp *MultiProvider) GetProvider(exchange models.Exchange) (MarketProvider, error) {
	provider, exists := mp.providers[exchange]
	if !exists {
		return nil, fmt.Errorf("нет провайдера для биржи %s", exchange)
	}
	return provider, nil
}

// GetQuote получает котировку от соответствующего провайдера
func (mp *MultiProvider) GetQuote(ctx context.Context, ticker string, exchange models.Exchange) (*models.MarketQuote, error) {
	provider, err := mp.GetProvider(exchange)
	if err != nil {
		return nil, err
	}
	return provider.GetQuote(ctx, ticker, exchange)
}

// GetQuotes получает несколько котировок
func (mp *MultiProvider) GetQuotes(ctx context.Context, tickers []string, exchange models.Exchange) (map[string]*models.MarketQuote, error) {
	provider, err := mp.GetProvider(exchange)
	if err != nil {
		return nil, err
	}
	return provider.GetQuotes(ctx, tickers, exchange)
}

// SearchSecurities ищет ценные бумаги по всем включённым провайдерам
func (mp *MultiProvider) SearchSecurities(ctx context.Context, query string, securityType *models.SecurityType, exchange *models.Exchange) ([]models.Security, error) {
	var results []models.Security

	if exchange != nil {
		// Поиск только на конкретной бирже
		provider, err := mp.GetProvider(*exchange)
		if err != nil {
			return nil, err
		}
		return provider.SearchSecurities(ctx, query, securityType, *exchange)
	}

	// Поиск по всем провайдерам
	seen := make(map[string]bool)
	for _, provider := range mp.providers {
		if seen[provider.GetName()] {
			continue
		}
		seen[provider.GetName()] = true

		securities, err := provider.SearchSecurities(ctx, query, securityType, *exchange)
		if err != nil {
			continue // Пропускаем провайдеры с ошибками
		}
		results = append(results, securities...)
	}

	return results, nil
}

// GetSecurityInfo получает детальную информацию о ценной бумаге
func (mp *MultiProvider) GetSecurityInfo(ctx context.Context, ticker string, exchange models.Exchange) (*models.Security, error) {
	provider, err := mp.GetProvider(exchange)
	if err != nil {
		return nil, err
	}
	return provider.GetSecurityInfo(ctx, ticker, exchange)
}

// GetPriceHistory получает историю цен
func (mp *MultiProvider) GetPriceHistory(ctx context.Context, ticker string, exchange models.Exchange, from, to time.Time) ([]PriceBar, error) {
	provider, err := mp.GetProvider(exchange)
	if err != nil {
		return nil, err
	}
	return provider.GetPriceHistory(ctx, ticker, exchange, from, to)
}

// GetDividends получает историю дивидендов
func (mp *MultiProvider) GetDividends(ctx context.Context, ticker string, exchange models.Exchange) ([]models.Dividend, error) {
	provider, err := mp.GetProvider(exchange)
	if err != nil {
		return nil, err
	}
	return provider.GetDividends(ctx, ticker, exchange)
}

// GetCurrencyRate получает курс обмена валют
func (mp *MultiProvider) GetCurrencyRate(ctx context.Context, from, to string) (decimal.Decimal, error) {
	// Сначала пробуем MOEX для пар с рублём
	if from == "RUB" || to == "RUB" {
		if provider, exists := mp.providers[models.ExchangeMOEX]; exists {
			rate, err := provider.GetCurrencyRate(ctx, from, to)
			if err == nil {
				return rate, nil
			}
		}
	}

	// Пробуем другие провайдеры для остальных валют
	for _, provider := range mp.providers {
		rate, err := provider.GetCurrencyRate(ctx, from, to)
		if err == nil {
			return rate, nil
		}
	}

	return decimal.Zero, fmt.Errorf("не удалось получить курс для %s/%s", from, to)
}

// GetSupportedExchanges возвращает все поддерживаемые биржи
func (mp *MultiProvider) GetSupportedExchanges() []models.Exchange {
	exchanges := make([]models.Exchange, 0, len(mp.providers))
	for exchange := range mp.providers {
		exchanges = append(exchanges, exchange)
	}
	return exchanges
}

// IsExchangeSupported проверяет, поддерживается ли биржа
func (mp *MultiProvider) IsExchangeSupported(exchange models.Exchange) bool {
	_, exists := mp.providers[exchange]
	return exists
}
