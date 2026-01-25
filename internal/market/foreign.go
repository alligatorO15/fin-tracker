package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ForeignProvider implements MarketProvider for foreign exchanges
// Uses Alpha Vantage and Twelve Data APIs
// This provider is designed to be enabled when sanctions are lifted
type ForeignProvider struct {
	alphaVantageKey string
	twelveDataKey   string
	httpClient      *http.Client
	enabled         bool
}

// NewForeignProvider creates a new foreign market provider
func NewForeignProvider(alphaVantageKey, twelveDataKey string) *ForeignProvider {
	return &ForeignProvider{
		alphaVantageKey: alphaVantageKey,
		twelveDataKey:   twelveDataKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled: alphaVantageKey != "" || twelveDataKey != "",
	}
}

func (p *ForeignProvider) GetName() string {
	return "Foreign"
}

func (p *ForeignProvider) GetSupportedExchanges() []models.Exchange {
	return []models.Exchange{
		models.ExchangeNYSE,
		models.ExchangeNASDAQ,
		models.ExchangeLSE,
		models.ExchangeFRA,
		models.ExchangeHKEX,
	}
}

func (p *ForeignProvider) IsEnabled() bool {
	return p.enabled
}

// AlphaVantage response structures
type AVQuoteResponse struct {
	GlobalQuote struct {
		Symbol           string `json:"01. symbol"`
		Open             string `json:"02. open"`
		High             string `json:"03. high"`
		Low              string `json:"04. low"`
		Price            string `json:"05. price"`
		Volume           string `json:"06. volume"`
		LatestTradingDay string `json:"07. latest trading day"`
		PreviousClose    string `json:"08. previous close"`
		Change           string `json:"09. change"`
		ChangePercent    string `json:"10. change percent"`
	} `json:"Global Quote"`
}

type AVSearchResponse struct {
	BestMatches []struct {
		Symbol      string `json:"1. symbol"`
		Name        string `json:"2. name"`
		Type        string `json:"3. type"`
		Region      string `json:"4. region"`
		MarketOpen  string `json:"5. marketOpen"`
		MarketClose string `json:"6. marketClose"`
		Timezone    string `json:"7. timezone"`
		Currency    string `json:"8. currency"`
		MatchScore  string `json:"9. matchScore"`
	} `json:"bestMatches"`
}

type AVTimeSeriesResponse struct {
	MetaData struct {
		Symbol string `json:"2. Symbol"`
	} `json:"Meta Data"`
	TimeSeries map[string]struct {
		Open   string `json:"1. open"`
		High   string `json:"2. high"`
		Low    string `json:"3. low"`
		Close  string `json:"4. close"`
		Volume string `json:"5. volume"`
	} `json:"Time Series (Daily)"`
}

func (p *ForeignProvider) GetQuote(ctx context.Context, ticker string, exchange models.Exchange) (*models.MarketQuote, error) {
	if !p.enabled {
		return nil, fmt.Errorf("foreign market provider is disabled")
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s",
		ticker, p.alphaVantageKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var avResp AVQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&avResp); err != nil {
		return nil, err
	}

	if avResp.GlobalQuote.Symbol == "" {
		return nil, fmt.Errorf("no quote data for %s", ticker)
	}

	quote := &models.MarketQuote{
		Ticker:    ticker,
		Exchange:  exchange,
		Timestamp: time.Now(),
	}

	quote.LastPrice, _ = decimal.NewFromString(avResp.GlobalQuote.Price)
	quote.Open, _ = decimal.NewFromString(avResp.GlobalQuote.Open)
	quote.High, _ = decimal.NewFromString(avResp.GlobalQuote.High)
	quote.Low, _ = decimal.NewFromString(avResp.GlobalQuote.Low)
	quote.Close, _ = decimal.NewFromString(avResp.GlobalQuote.PreviousClose)
	quote.Change, _ = decimal.NewFromString(avResp.GlobalQuote.Change)

	// Parse change percent (remove % sign)
	changePercentStr := avResp.GlobalQuote.ChangePercent
	if len(changePercentStr) > 0 && changePercentStr[len(changePercentStr)-1] == '%' {
		changePercentStr = changePercentStr[:len(changePercentStr)-1]
	}
	quote.ChangePercent, _ = decimal.NewFromString(changePercentStr)

	return quote, nil
}

func (p *ForeignProvider) GetQuotes(ctx context.Context, tickers []string, exchange models.Exchange) (map[string]*models.MarketQuote, error) {
	result := make(map[string]*models.MarketQuote)

	// Alpha Vantage doesn't support batch quotes in free tier
	// Fetch one by one
	for _, ticker := range tickers {
		quote, err := p.GetQuote(ctx, ticker, exchange)
		if err != nil {
			continue
		}
		result[ticker] = quote
	}

	return result, nil
}

func (p *ForeignProvider) SearchSecurities(ctx context.Context, query string, securityType *models.SecurityType) ([]models.Security, error) {
	if !p.enabled {
		return nil, fmt.Errorf("foreign market provider is disabled")
	}

	encodedQuery := url.QueryEscape(query)
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=SYMBOL_SEARCH&keywords=%s&apikey=%s",
		encodedQuery, p.alphaVantageKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var avResp AVSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&avResp); err != nil {
		return nil, err
	}

	var securities []models.Security
	for _, match := range avResp.BestMatches {
		security := models.Security{
			ID:       uuid.New(),
			Ticker:   match.Symbol,
			Name:     match.Name,
			Currency: match.Currency,
			Country:  p.regionToCountry(match.Region),
			Exchange: p.regionToExchange(match.Region),
			Type:     p.typeToSecurityType(match.Type),
			IsActive: true,
			LotSize:  1,
		}

		// Filter by security type if specified
		if securityType != nil && security.Type != *securityType {
			continue
		}

		securities = append(securities, security)
	}

	return securities, nil
}

func (p *ForeignProvider) GetSecurityInfo(ctx context.Context, ticker string, exchange models.Exchange) (*models.Security, error) {
	if !p.enabled {
		return nil, fmt.Errorf("foreign market provider is disabled")
	}

	// Use overview endpoint for detailed info
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=OVERVIEW&symbol=%s&apikey=%s",
		ticker, p.alphaVantageKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var overview map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&overview); err != nil {
		return nil, err
	}

	if overview["Symbol"] == "" {
		return nil, fmt.Errorf("security not found: %s", ticker)
	}

	security := &models.Security{
		ID:       uuid.New(),
		Ticker:   overview["Symbol"],
		Name:     overview["Name"],
		Exchange: exchange,
		Currency: overview["Currency"],
		Country:  overview["Country"],
		Sector:   overview["Sector"],
		Industry: overview["Industry"],
		Type:     p.assetTypeToSecurityType(overview["AssetType"]),
		IsActive: true,
		LotSize:  1,
	}

	return security, nil
}

func (p *ForeignProvider) GetPriceHistory(ctx context.Context, ticker string, exchange models.Exchange, from, to time.Time) ([]PriceBar, error) {
	if !p.enabled {
		return nil, fmt.Errorf("foreign market provider is disabled")
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=full&apikey=%s",
		ticker, p.alphaVantageKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var avResp AVTimeSeriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&avResp); err != nil {
		return nil, err
	}

	var bars []PriceBar
	for dateStr, data := range avResp.TimeSeries {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Filter by date range
		if date.Before(from) || date.After(to) {
			continue
		}

		bar := PriceBar{
			Date: date,
		}
		bar.Open, _ = decimal.NewFromString(data.Open)
		bar.High, _ = decimal.NewFromString(data.High)
		bar.Low, _ = decimal.NewFromString(data.Low)
		bar.Close, _ = decimal.NewFromString(data.Close)

		bars = append(bars, bar)
	}

	return bars, nil
}

func (p *ForeignProvider) GetDividends(ctx context.Context, ticker string, exchange models.Exchange) ([]models.Dividend, error) {
	// Alpha Vantage doesn't provide comprehensive dividend data in free tier
	// Return empty list - premium feature
	return nil, fmt.Errorf("dividend data not available for foreign securities in free tier")
}

func (p *ForeignProvider) GetCurrencyRate(ctx context.Context, from, to string) (decimal.Decimal, error) {
	if !p.enabled {
		return decimal.Zero, fmt.Errorf("foreign market provider is disabled")
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=CURRENCY_EXCHANGE_RATE&from_currency=%s&to_currency=%s&apikey=%s",
		from, to, p.alphaVantageKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return decimal.Zero, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return decimal.Zero, err
	}
	defer resp.Body.Close()

	var result struct {
		RealtimeCurrencyExchangeRate struct {
			ExchangeRate string `json:"5. Exchange Rate"`
		} `json:"Realtime Currency Exchange Rate"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decimal.Zero, err
	}

	rate, err := decimal.NewFromString(result.RealtimeCurrencyExchangeRate.ExchangeRate)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid exchange rate")
	}

	return rate, nil
}

// Helper methods

func (p *ForeignProvider) regionToCountry(region string) string {
	switch region {
	case "United States":
		return "US"
	case "United Kingdom":
		return "GB"
	case "Germany", "Frankfurt":
		return "DE"
	case "Hong Kong":
		return "HK"
	case "Japan":
		return "JP"
	case "China":
		return "CN"
	default:
		return "US"
	}
}

func (p *ForeignProvider) regionToExchange(region string) models.Exchange {
	switch region {
	case "United States":
		return models.ExchangeNYSE
	case "United Kingdom":
		return models.ExchangeLSE
	case "Germany", "Frankfurt":
		return models.ExchangeFRA
	case "Hong Kong":
		return models.ExchangeHKEX
	default:
		return models.ExchangeNYSE
	}
}

func (p *ForeignProvider) typeToSecurityType(avType string) models.SecurityType {
	switch avType {
	case "Equity":
		return models.SecurityTypeStock
	case "ETF":
		return models.SecurityTypeETF
	case "Mutual Fund":
		return models.SecurityTypeMutualFund
	default:
		return models.SecurityTypeStock
	}
}

func (p *ForeignProvider) assetTypeToSecurityType(assetType string) models.SecurityType {
	switch assetType {
	case "Common Stock":
		return models.SecurityTypeStock
	case "ETF":
		return models.SecurityTypeETF
	case "Mutual Fund":
		return models.SecurityTypeMutualFund
	default:
		return models.SecurityTypeStock
	}
}
