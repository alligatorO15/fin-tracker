package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CryptoProvider реализует интерфейс MarketProvider для криптовалютных рынков
// использует CoinGecko API (бесплатный тариф)
type CryptoProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewCryptoProvider создаёт новый экземпляр крипто-провайдера
func NewCryptoProvider() *CryptoProvider {
	return &CryptoProvider{
		baseURL: "https://api.coingecko.com/api/v3",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *CryptoProvider) GetName() string {
	return "Crypto"
}

func (p *CryptoProvider) GetSupportedExchanges() []models.Exchange {
	return []models.Exchange{models.ExchangeCRYPTO}
}

func (p *CryptoProvider) IsEnabled() bool {
	return true
}

// Структуры ответов CoinGecko
type CGCoinMarket struct {
	ID                       string  `json:"id"`
	Symbol                   string  `json:"symbol"`
	Name                     string  `json:"name"`
	CurrentPrice             float64 `json:"current_price"`
	MarketCap                float64 `json:"market_cap"`
	TotalVolume              float64 `json:"total_volume"`
	High24h                  float64 `json:"high_24h"`
	Low24h                   float64 `json:"low_24h"`
	PriceChange24h           float64 `json:"price_change_24h"`
	PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
}

type CGCoinDetail struct {
	ID          string `json:"id"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Description struct {
		En string `json:"en"`
	} `json:"description"`
	MarketData struct {
		CurrentPrice             map[string]float64 `json:"current_price"`
		High24h                  map[string]float64 `json:"high_24h"`
		Low24h                   map[string]float64 `json:"low_24h"`
		PriceChange24h           map[string]float64 `json:"price_change_24h"`
		PriceChangePercentage24h float64            `json:"price_change_percentage_24h"`
		TotalVolume              map[string]float64 `json:"total_volume"`
	} `json:"market_data"`
}

type CGSearchResult struct {
	Coins []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
	} `json:"coins"`
}

type CGMarketChart struct {
	Prices       [][]float64 `json:"prices"`
	MarketCaps   [][]float64 `json:"market_caps"`
	TotalVolumes [][]float64 `json:"total_volumes"`
}

// маппинг популярных тикеров на ID CoinGecko (позже можно добавить свои)
var cryptoIDMap = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"USDT":  "tether",
	"BNB":   "binancecoin",
	"SOL":   "solana",
	"XRP":   "ripple",
	"DOGE":  "dogecoin",
	"ADA":   "cardano",
	"TRX":   "tron",
	"TON":   "the-open-network",
	"AVAX":  "avalanche-2",
	"DOT":   "polkadot",
	"MATIC": "matic-network",
	"LINK":  "chainlink",
	"UNI":   "uniswap",
	"ATOM":  "cosmos",
	"LTC":   "litecoin",
	"XLM":   "stellar",
}

func (p *CryptoProvider) GetQuote(ctx context.Context, ticker string, exchange models.Exchange) (*models.MarketQuote, error) {
	coinID := p.tickerToCoinID(ticker)

	url := fmt.Sprintf("%s/coins/%s?localization=false&tickers=false&community_data=false&developer_data=false",
		p.baseURL, coinID)

	var coin CGCoinDetail
	if err := p.makeRequest(ctx, url, &coin); err != nil {
		return nil, err
	}

	quote := &models.MarketQuote{
		Ticker:        strings.ToUpper(coin.Symbol),
		Exchange:      models.ExchangeCRYPTO,
		Timestamp:     time.Now(),
		LastPrice:     decimal.NewFromFloat(p.getFloat(coin.MarketData.CurrentPrice, "usd")),
		High:          decimal.NewFromFloat(p.getFloat(coin.MarketData.High24h, "usd")),
		Low:           decimal.NewFromFloat(p.getFloat(coin.MarketData.Low24h, "usd")),
		Change:        decimal.NewFromFloat(p.getFloat(coin.MarketData.PriceChange24h, "usd")),
		ChangePercent: decimal.NewFromFloat(coin.MarketData.PriceChangePercentage24h),
		Volume:        int64(p.getFloat(coin.MarketData.TotalVolume, "usd")),
	}

	return quote, nil
}

func (p *CryptoProvider) GetQuotes(ctx context.Context, tickers []string, exchange models.Exchange) (map[string]*models.MarketQuote, error) {
	// конвертируем тикеры в ID CoinGecko
	var coinIDs []string
	tickerToID := make(map[string]string) // и запоминаем обратное, чтобы потом время поиска O(1)

	for _, ticker := range tickers {
		coinID := p.tickerToCoinID(ticker)
		coinIDs = append(coinIDs, coinID)
		tickerToID[coinID] = strings.ToUpper(ticker)
	}

	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&ids=%s&sparkline=false",
		p.baseURL, strings.Join(coinIDs, ","))

	var coins []CGCoinMarket
	if err := p.makeRequest(ctx, url, &coins); err != nil {
		return nil, err
	}

	result := make(map[string]*models.MarketQuote)
	for _, coin := range coins {
		ticker := tickerToID[coin.ID]
		if ticker == "" {
			ticker = strings.ToUpper(coin.Symbol)
		}

		result[ticker] = &models.MarketQuote{
			Ticker:        ticker,
			Exchange:      models.ExchangeCRYPTO,
			Timestamp:     time.Now(),
			LastPrice:     decimal.NewFromFloat(coin.CurrentPrice),
			High:          decimal.NewFromFloat(coin.High24h),
			Low:           decimal.NewFromFloat(coin.Low24h),
			Change:        decimal.NewFromFloat(coin.PriceChange24h),
			ChangePercent: decimal.NewFromFloat(coin.PriceChangePercentage24h),
			Volume:        int64(coin.TotalVolume),
		}
	}

	return result, nil
}

func (p *CryptoProvider) SearchSecurities(ctx context.Context, query string, securityType *models.SecurityType, exchange models.Exchange) ([]models.Security, error) {
	// фильтруем по типу — возвращаем результаты только если запрошены крипто или без фильтра
	if securityType != nil && *securityType != models.SecurityTypeCrypto {
		return nil, nil
	}

	encodedQuery := url.QueryEscape(query)
	url := fmt.Sprintf("%s/search?query=%s", p.baseURL, encodedQuery)

	var searchResult CGSearchResult
	if err := p.makeRequest(ctx, url, &searchResult); err != nil {
		return nil, err
	}

	var securities []models.Security
	for _, coin := range searchResult.Coins {
		if len(securities) >= 50 {
			break
		}

		securities = append(securities, models.Security{
			ID:       uuid.New(),
			Ticker:   strings.ToUpper(coin.Symbol),
			Name:     coin.Name,
			Type:     models.SecurityTypeCrypto,
			Exchange: exchange,
			Currency: "USD",
			IsActive: true,
			LotSize:  1,
		})
	}

	return securities, nil
}

func (p *CryptoProvider) GetSecurityInfo(ctx context.Context, ticker string, exchange models.Exchange) (*models.Security, error) {
	coinID := p.tickerToCoinID(ticker)

	url := fmt.Sprintf("%s/coins/%s?localization=false&tickers=false&market_data=true&community_data=false&developer_data=false",
		p.baseURL, coinID)

	var coin CGCoinDetail
	if err := p.makeRequest(ctx, url, &coin); err != nil {
		return nil, fmt.Errorf("криптовалюта не найдена: %s", ticker)
	}

	security := &models.Security{
		ID:                 uuid.New(),
		Ticker:             strings.ToUpper(coin.Symbol),
		Name:               coin.Name,
		ShortName:          strings.ToUpper(coin.Symbol), // используем символ как короткое имя
		Type:               models.SecurityTypeCrypto,
		Exchange:           models.ExchangeCRYPTO,
		Currency:           "USD",
		IsActive:           true,
		LotSize:            1,
		MinPriceIncrement:  decimal.NewFromFloat(0.00000001), // Минимальная единица для крипты (satoshi)
		LastPrice:          decimal.NewFromFloat(p.getFloat(coin.MarketData.CurrentPrice, "usd")),
		PriceChange:        decimal.NewFromFloat(p.getFloat(coin.MarketData.PriceChange24h, "usd")),
		PriceChangePercent: decimal.NewFromFloat(coin.MarketData.PriceChangePercentage24h),
		Volume:             int64(p.getFloat(coin.MarketData.TotalVolume, "usd")),
	}

	return security, nil
}

func (p *CryptoProvider) GetPriceHistory(ctx context.Context, ticker string, exchange models.Exchange, from, to time.Time) ([]PriceBar, error) {
	coinID := p.tickerToCoinID(ticker)

	// CoinGecko использует Unix timestamps
	fromTS := from.Unix()
	toTS := to.Unix()

	url := fmt.Sprintf("%s/coins/%s/market_chart/range?vs_currency=usd&from=%d&to=%d",
		p.baseURL, coinID, fromTS, toTS)

	var chart CGMarketChart
	if err := p.makeRequest(ctx, url, &chart); err != nil {
		return nil, err
	}

	// CoinGecko возвращает точки данных, а не OHLCV бары
	// создаём дневные бары группируя точки данных
	dailyPrices := make(map[string][]float64)
	dailyVolumes := make(map[string]float64)

	for i, pricePoint := range chart.Prices {
		if len(pricePoint) < 2 {
			continue
		}

		timestamp := time.Unix(int64(pricePoint[0])/1000, 0)
		dateKey := timestamp.Format("2006-01-02")
		price := pricePoint[1]

		dailyPrices[dateKey] = append(dailyPrices[dateKey], price)

		if i < len(chart.TotalVolumes) && len(chart.TotalVolumes[i]) >= 2 {
			dailyVolumes[dateKey] += chart.TotalVolumes[i][1]
		}
	}

	var bars []PriceBar
	for dateKey, prices := range dailyPrices {
		if len(prices) == 0 {
			continue
		}

		date, _ := time.Parse("2006-01-02", dateKey)

		// вычисляем OHLC из точек данных
		open := prices[0]
		close := prices[len(prices)-1]
		high := prices[0]
		low := prices[0]

		for _, p := range prices {
			if p > high {
				high = p
			}
			if p < low {
				low = p
			}
		}

		bars = append(bars, PriceBar{
			Date:   date,
			Open:   decimal.NewFromFloat(open),
			High:   decimal.NewFromFloat(high),
			Low:    decimal.NewFromFloat(low),
			Close:  decimal.NewFromFloat(close),
			Volume: int64(dailyVolumes[dateKey]),
		})
	}

	return bars, nil
}

func (p *CryptoProvider) GetDividends(ctx context.Context, ticker string, exchange models.Exchange) ([]models.Dividend, error) {
	// криптовалюты не имеют традиционных дивидендов
	return nil, nil
}

func (p *CryptoProvider) GetCurrencyRate(ctx context.Context, from, to string) (decimal.Decimal, error) {
	// обрабатываем конвертацию крипто в фиат
	if from == to {
		return decimal.NewFromInt(1), nil
	}

	// используем простой эндпоинт для цен
	fromLower := strings.ToLower(from)
	toLower := strings.ToLower(to)

	// проверяем, является ли 'from' криптовалютой
	coinID := p.tickerToCoinID(fromLower)

	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=%s", p.baseURL, coinID, toLower)

	var result map[string]map[string]float64
	if err := p.makeRequest(ctx, url, &result); err != nil {
		return decimal.Zero, err
	}

	if prices, ok := result[coinID]; ok {
		if rate, ok := prices[toLower]; ok {
			return decimal.NewFromFloat(rate), nil
		}
	}

	return decimal.Zero, fmt.Errorf("не удалось получить курс для %s/%s", from, to)
}

// вспомогательные методы
// метод запроса
func (p *CryptoProvider) makeRequest(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка CoinGecko API: статус %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// getFloat безопасно извлекает float из map[string]float64
func (p *CryptoProvider) getFloat(m map[string]float64, key string) float64 {
	if val, ok := m[key]; ok {
		return val
	}
	return 0
}

// привычный формат тикера крипты в тикер ожидаемый геко
func (p *CryptoProvider) tickerToCoinID(ticker string) string {
	ticker = strings.ToUpper(ticker)
	if id, ok := cryptoIDMap[ticker]; ok {
		return id
	}
	return strings.ToLower(ticker)
}
