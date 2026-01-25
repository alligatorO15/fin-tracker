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

// MOEXProvider implements MarketProvider for Moscow Exchange
type MOEXProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewMOEXProvider creates a new MOEX provider instance
func NewMOEXProvider(baseURL string) *MOEXProvider {
	if baseURL == "" {
		baseURL = "https://iss.moex.com/iss"
	}

	return &MOEXProvider{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *MOEXProvider) GetName() string {
	return "MOEX"
}

func (p *MOEXProvider) GetSupportedExchanges() []models.Exchange {
	return []models.Exchange{models.ExchangeMOEX, models.ExchangeSPB}
}

func (p *MOEXProvider) IsEnabled() bool {
	return true
}

// MOEXResponse represents the standard MOEX ISS API response structure
type MOEXResponse struct {
	Securities struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"securities"`
	Marketdata struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"marketdata"`
	History struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"history"`
	Dividends struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"dividends"`
	Coupons struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"coupons"`
}

func (p *MOEXProvider) GetQuote(ctx context.Context, ticker string, exchange models.Exchange) (*models.MarketQuote, error) {
	// Determine board and engine based on ticker format
	engine, market, board := p.detectMarket(ticker)

	url := fmt.Sprintf("%s/engines/%s/markets/%s/boards/%s/securities/%s.json?iss.meta=off",
		p.baseURL, engine, market, board, ticker)

	resp, err := p.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	if len(resp.Marketdata.Data) == 0 {
		return nil, fmt.Errorf("no market data for %s", ticker)
	}

	// Parse columns to find indices
	mdCols := makeColumnIndex(resp.Marketdata.Columns)

	data := resp.Marketdata.Data[0]

	quote := &models.MarketQuote{
		Ticker:    ticker,
		Exchange:  exchange,
		Timestamp: time.Now(),
	}

	// Extract values safely
	quote.LastPrice = p.getDecimal(data, mdCols, "LAST", "CURRENTVALUE")
	quote.Open = p.getDecimal(data, mdCols, "OPEN", "OPENVALUE")
	quote.High = p.getDecimal(data, mdCols, "HIGH", "HIGHVALUE")
	quote.Low = p.getDecimal(data, mdCols, "LOW", "LOWVALUE")
	quote.Close = p.getDecimal(data, mdCols, "CLOSE", "LEGALCLOSEPRICE")
	quote.Bid = p.getDecimal(data, mdCols, "BID")
	quote.Ask = p.getDecimal(data, mdCols, "OFFER")
	quote.Change = p.getDecimal(data, mdCols, "CHANGE")
	quote.ChangePercent = p.getDecimal(data, mdCols, "LASTTOPREVPRICE", "CHANGEPERCENT")

	if v, ok := mdCols["VOLTODAY"]; ok && v < len(data) {
		if vol, ok := data[v].(float64); ok {
			quote.Volume = int64(vol)
		}
	}

	return quote, nil
}

func (p *MOEXProvider) GetQuotes(ctx context.Context, tickers []string, exchange models.Exchange) (map[string]*models.MarketQuote, error) {
	result := make(map[string]*models.MarketQuote)

	// MOEX allows batch requests
	tickerList := strings.Join(tickers, ",")
	engine, market, _ := p.detectMarket(tickers[0])

	url := fmt.Sprintf("%s/engines/%s/markets/%s/securities.json?iss.meta=off&securities=%s",
		p.baseURL, engine, market, tickerList)

	resp, err := p.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	mdCols := makeColumnIndex(resp.Marketdata.Columns)
	secCols := makeColumnIndex(resp.Securities.Columns)

	for _, data := range resp.Marketdata.Data {
		var ticker string
		if secIdx, ok := mdCols["SECID"]; ok && secIdx < len(data) {
			ticker, _ = data[secIdx].(string)
		}
		if ticker == "" {
			continue
		}

		quote := &models.MarketQuote{
			Ticker:    ticker,
			Exchange:  exchange,
			Timestamp: time.Now(),
		}

		quote.LastPrice = p.getDecimal(data, mdCols, "LAST", "CURRENTVALUE")
		quote.Change = p.getDecimal(data, mdCols, "CHANGE")
		quote.ChangePercent = p.getDecimal(data, mdCols, "LASTTOPREVPRICE")

		if v, ok := mdCols["VOLTODAY"]; ok && v < len(data) {
			if vol, ok := data[v].(float64); ok {
				quote.Volume = int64(vol)
			}
		}

		result[ticker] = quote
	}

	// Fill in additional info from securities data
	for _, data := range resp.Securities.Data {
		var ticker string
		if secIdx, ok := secCols["SECID"]; ok && secIdx < len(data) {
			ticker, _ = data[secIdx].(string)
		}
		if ticker == "" {
			continue
		}

		if quote, exists := result[ticker]; exists {
			if quote.LastPrice.IsZero() {
				quote.LastPrice = p.getDecimal(data, secCols, "PREVPRICE", "PREVADMITTEDQUOTE")
			}
		}
	}

	return result, nil
}

func (p *MOEXProvider) SearchSecurities(ctx context.Context, query string, securityType *models.SecurityType) ([]models.Security, error) {
	encodedQuery := url.QueryEscape(query)

	url := fmt.Sprintf("%s/securities.json?iss.meta=off&q=%s&limit=50", p.baseURL, encodedQuery)

	resp, err := p.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	cols := makeColumnIndex(resp.Securities.Columns)

	var securities []models.Security
	for _, data := range resp.Securities.Data {
		security := models.Security{
			ID:       uuid.New(),
			Exchange: models.ExchangeMOEX,
			IsActive: true,
		}

		security.Ticker = p.getString(data, cols, "secid")
		security.Name = p.getString(data, cols, "name")
		security.ShortName = p.getString(data, cols, "shortname")
		security.ISIN = p.getString(data, cols, "isin")

		// Determine security type from group
		group := p.getString(data, cols, "group")
		security.Type = p.mapSecurityType(group)

		// Filter by security type if specified
		if securityType != nil && security.Type != *securityType {
			continue
		}

		// Determine currency
		security.Currency = "RUB"
		if strings.Contains(group, "foreign") {
			security.Currency = "USD"
		}

		security.Country = "RU"

		securities = append(securities, security)
	}

	return securities, nil
}

func (p *MOEXProvider) GetSecurityInfo(ctx context.Context, ticker string, exchange models.Exchange) (*models.Security, error) {
	url := fmt.Sprintf("%s/securities/%s.json?iss.meta=off", p.baseURL, ticker)

	resp, err := p.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	if len(resp.Securities.Data) == 0 {
		return nil, fmt.Errorf("security not found: %s", ticker)
	}

	cols := makeColumnIndex(resp.Securities.Columns)
	data := resp.Securities.Data[0]

	security := &models.Security{
		ID:       uuid.New(),
		Ticker:   ticker,
		Exchange: exchange,
		IsActive: true,
		Country:  "RU",
		Currency: "RUB",
		LotSize:  1,
	}

	security.Name = p.getString(data, cols, "name")
	security.ShortName = p.getString(data, cols, "shortname")
	security.ISIN = p.getString(data, cols, "isin")

	group := p.getString(data, cols, "group")
	security.Type = p.mapSecurityType(group)

	// Get lot size
	if v := p.getFloat(data, cols, "lotsize"); v > 0 {
		security.LotSize = int(v)
	}

	// Get min price increment
	security.MinPriceIncrement = decimal.NewFromFloat(p.getFloat(data, cols, "minstep"))

	// Bond specific fields
	if security.Type == models.SecurityTypeBond {
		faceValue := p.getFloat(data, cols, "facevalue")
		if faceValue > 0 {
			fv := decimal.NewFromFloat(faceValue)
			security.FaceValue = &fv
		}

		couponRate := p.getFloat(data, cols, "couponpercent")
		if couponRate > 0 {
			cr := decimal.NewFromFloat(couponRate)
			security.CouponRate = &cr
		}

		matDate := p.getString(data, cols, "matdate")
		if matDate != "" {
			if t, err := time.Parse("2006-01-02", matDate); err == nil {
				security.MaturityDate = &t
			}
		}
	}

	return security, nil
}

func (p *MOEXProvider) GetPriceHistory(ctx context.Context, ticker string, exchange models.Exchange, from, to time.Time) ([]PriceBar, error) {
	engine, market, board := p.detectMarket(ticker)

	var bars []PriceBar
	startDate := from.Format("2006-01-02")
	endDate := to.Format("2006-01-02")
	start := 0

	for {
		url := fmt.Sprintf("%s/history/engines/%s/markets/%s/boards/%s/securities/%s.json?iss.meta=off&from=%s&till=%s&start=%d",
			p.baseURL, engine, market, board, ticker, startDate, endDate, start)

		resp, err := p.makeRequest(ctx, url)
		if err != nil {
			return nil, err
		}

		if len(resp.History.Data) == 0 {
			break
		}

		cols := makeColumnIndex(resp.History.Columns)

		for _, data := range resp.History.Data {
			dateStr := p.getString(data, cols, "TRADEDATE")
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}

			bar := PriceBar{
				Date:   date,
				Open:   decimal.NewFromFloat(p.getFloat(data, cols, "OPEN")),
				High:   decimal.NewFromFloat(p.getFloat(data, cols, "HIGH")),
				Low:    decimal.NewFromFloat(p.getFloat(data, cols, "LOW")),
				Close:  decimal.NewFromFloat(p.getFloat(data, cols, "CLOSE", "LEGALCLOSEPRICE")),
				Volume: int64(p.getFloat(data, cols, "VOLUME")),
			}

			bars = append(bars, bar)
		}

		if len(resp.History.Data) < 100 {
			break
		}
		start += 100
	}

	return bars, nil
}

func (p *MOEXProvider) GetDividends(ctx context.Context, ticker string, exchange models.Exchange) ([]models.Dividend, error) {
	url := fmt.Sprintf("%s/securities/%s/dividends.json?iss.meta=off", p.baseURL, ticker)

	resp, err := p.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	cols := makeColumnIndex(resp.Dividends.Columns)

	var dividends []models.Dividend
	for _, data := range resp.Dividends.Data {
		dividend := models.Dividend{
			ID:           uuid.New(),
			Currency:     "RUB",
			DividendType: "regular",
		}

		dividend.Amount = decimal.NewFromFloat(p.getFloat(data, cols, "value"))

		if dateStr := p.getString(data, cols, "registryclosedate"); dateStr != "" {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dividend.RecordDate = t
				dividend.ExDate = t.AddDate(0, 0, -2) // Approximate ex-date
			}
		}

		dividends = append(dividends, dividend)
	}

	return dividends, nil
}

func (p *MOEXProvider) GetCurrencyRate(ctx context.Context, from, to string) (decimal.Decimal, error) {
	// Handle RUB pairs using MOEX currency market
	var ticker string
	var invert bool

	switch {
	case from == "USD" && to == "RUB":
		ticker = "USD000UTSTOM"
	case from == "RUB" && to == "USD":
		ticker = "USD000UTSTOM"
		invert = true
	case from == "EUR" && to == "RUB":
		ticker = "EUR_RUB__TOM"
	case from == "RUB" && to == "EUR":
		ticker = "EUR_RUB__TOM"
		invert = true
	case from == "CNY" && to == "RUB":
		ticker = "CNYRUB_TOM"
	case from == "RUB" && to == "CNY":
		ticker = "CNYRUB_TOM"
		invert = true
	default:
		return decimal.Zero, fmt.Errorf("unsupported currency pair: %s/%s", from, to)
	}

	url := fmt.Sprintf("%s/engines/currency/markets/selt/boards/CETS/securities/%s.json?iss.meta=off", p.baseURL, ticker)

	resp, err := p.makeRequest(ctx, url)
	if err != nil {
		return decimal.Zero, err
	}

	if len(resp.Marketdata.Data) == 0 {
		return decimal.Zero, fmt.Errorf("no rate data for %s/%s", from, to)
	}

	cols := makeColumnIndex(resp.Marketdata.Columns)
	data := resp.Marketdata.Data[0]

	rate := p.getDecimal(data, cols, "LAST", "WAPRICE")
	if rate.IsZero() {
		return decimal.Zero, fmt.Errorf("could not get rate for %s/%s", from, to)
	}

	if invert {
		rate = decimal.NewFromInt(1).Div(rate)
	}

	return rate, nil
}

// Helper methods

func (p *MOEXProvider) makeRequest(ctx context.Context, url string) (*MOEXResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MOEX API error: %d", resp.StatusCode)
	}

	var result MOEXResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (p *MOEXProvider) detectMarket(ticker string) (engine, market, board string) {
	upperTicker := strings.ToUpper(ticker)

	// 1. Фьючерсы и опционы (содержат дефис и месяц/год)
	// Примеры: SI-6.25, RTS-6.25, BR-6.25, GOLD-6.25
	if strings.Contains(upperTicker, "-") {
		parts := strings.Split(upperTicker, "-")
		if len(parts) == 2 {
			// Проверяем формат месяца/года: M.MM или M.MM
			if strings.Contains(parts[1], ".") {
				return "futures", "forts", "RFUD"
			}
		}
	}

	// 2. Облигации (государственные, корпоративные, муниципальные)
	// SU - государственные, RU - российские, XS - еврооблигации
	if strings.HasPrefix(upperTicker, "SU") || strings.HasPrefix(upperTicker, "RU") ||
		strings.HasPrefix(upperTicker, "XS") || strings.HasPrefix(upperTicker, "RU000A") {
		// Проверяем, это не пай фонда (у паев тоже может начинаться на RU)
		if strings.HasPrefix(upperTicker, "RU000A10") {
			// Паи ПИФов имеют специфичный префикс
			return "stock", "shares", "TQPI" // Режим для паев
		}
		return "stock", "bonds", "TQOB"
	}

	// 3. Валютные инструменты (спот и свопы)
	// USD000UTSTOM - доллар/рубль том
	// EUR_RUB__TOM - евро/рубль том
	// CNY000000TOD - юань/рубль tod
	if strings.Contains(upperTicker, "RUB") || strings.Contains(upperTicker, "USD") ||
		strings.Contains(upperTicker, "EUR") || strings.Contains(upperTicker, "CNY") ||
		strings.Contains(upperTicker, "GBP") || strings.Contains(upperTicker, "CHF") ||
		strings.Contains(upperTicker, "JPY") || strings.Contains(upperTicker, "TRY") ||
		strings.Contains(upperTicker, "HKD") || strings.Contains(upperTicker, "KZT") {
		// Определяем тип валютного инструмента
		if strings.Contains(upperTicker, "TOM") || strings.Contains(upperTicker, "TOD") {
			return "currency", "selt", "CETS"
		}
		// Для валютных свопов
		if strings.Contains(upperTicker, "SWAP") {
			return "currency", "swap", "CRTS"
		}
	}

	// 4. Иностранные акции (торгуемые на СПБ бирже с суффиксом)
	// AAPL-RM, TSLA-RM, BABA-RM
	if strings.HasSuffix(upperTicker, "-RM") || strings.HasSuffix(upperTicker, "-SPB") {
		return "stock", "foreignshares", "FQBR"
	}

	// 5. ETF и БПИФ
	// Проверяем по известным тикерам ETF на МБ
	etfTickers := map[string]bool{
		"FXGD": true, // FinEx Золото
		"FXRB": true, // FinEx ОФЗ
		"FXRL": true, // FinEx Акции
		"FXRU": true, // FinEx Корп облигации
		"FXUS": true, // FinEx США
		"FXDE": true, // FinEx Германия
		"FXCN": true, // FinEx Китай
		"TMOS": true, // Тинькофф iMOEX
		"TBIO": true, // Тинькофф Biotech
		"TECH": true, // Тинькофч Tech
		"TGLD": true, // Тинькофф Золото
		"TSPX": true, // Тинькофф S&P 500
		"SBGB": true, // Сбер БПИФ гос облигации
		"SBPR": true, // Сбер БПИФ привилегированные акции
		"VTBX": true, // ВТБ Акции
		"VTBH": true, // ВТБ Хедж-фонд
		"VTBE": true, // ВТБ Еврооблигации
		"VTBA": true, // ВТБ Американские акции
	}

	if etfTickers[upperTicker] {
		return "stock", "shares", "TQTF"
	}

	// 6. Паи ПИФов (фондов) - обычно начинаются с определенных префиксов
	if strings.HasPrefix(upperTicker, "PIF") || strings.HasPrefix(upperTicker, "ПИФ") ||
		strings.HasPrefix(upperTicker, "RU000A10") {
		return "stock", "shares", "TQPI"
	}

	// 7. Депозитарные расписки
	if strings.HasSuffix(upperTicker, "DR") {
		return "stock", "dr", "TQDR"
	}

	// 8. Индексные контракты
	if strings.HasPrefix(upperTicker, "IMOEX") || strings.HasPrefix(upperTicker, "RTS") {
		return "futures", "forts", "RFUD"
	}

	// 9. По умолчанию - российские акции основного режима
	return "stock", "shares", "TQBR"
}
func (p *MOEXProvider) mapSecurityType(group string) models.SecurityType {
	group = strings.ToLower(group)

	switch {
	case strings.Contains(group, "bond"):
		return models.SecurityTypeBond
	case strings.Contains(group, "etf") || strings.Contains(group, "ppif"):
		return models.SecurityTypeETF
	case strings.Contains(group, "currency"):
		return models.SecurityTypeCurrency
	case strings.Contains(group, "futures") || strings.Contains(group, "option"):
		return models.SecurityTypeDerivative
	default:
		return models.SecurityTypeStock
	}
}

func (p *MOEXProvider) getString(data []interface{}, cols map[string]int, keys ...string) string {
	for _, key := range keys {
		if idx, ok := cols[key]; ok && idx < len(data) {
			if s, ok := data[idx].(string); ok {
				return s
			}
		}
	}
	return ""
}

func (p *MOEXProvider) getFloat(data []interface{}, cols map[string]int, keys ...string) float64 {
	for _, key := range keys {
		if idx, ok := cols[key]; ok && idx < len(data) {
			if v, ok := data[idx].(float64); ok {
				return v
			}
		}
	}
	return 0
}

func (p *MOEXProvider) getDecimal(data []interface{}, cols map[string]int, keys ...string) decimal.Decimal {
	v := p.getFloat(data, cols, keys...)
	return decimal.NewFromFloat(v)
}

func makeColumnIndex(columns []string) map[string]int {
	idx := make(map[string]int)
	for i, col := range columns {
		idx[col] = i
	}
	return idx
}
