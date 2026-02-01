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

type MOEXProvider struct {
	baseURL    string
	httpClient *http.Client
}

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
	return []models.Exchange{models.ExchangeMOEX}
}

func (p *MOEXProvider) IsEnabled() bool {
	return true
}

// MOEXResponse представляет стандартную структуру ответа MOEX ISS API
type MOEXResponse struct {
	Securities struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"secutiries"`
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
	// определяем торговую систему, рынок и редим торгов для url
	engine, market, board := p.detectMarket(ticker)

	url := fmt.Sprintf("%s/engines/%s/markets/%s/boards/%s/securities/%s.json&iss.meta=off", p.baseURL, engine, market, board, ticker)

	resp, err := p.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	if len(resp.Marketdata.Data) == 0 {
		return nil, fmt.Errorf("нет рыночных данных для %s", ticker)
	}

	mdCols := makeColumnIndex(resp.Marketdata.Columns)
	data := resp.Marketdata.Data[0] // для одной бумаги один срез данных

	quote := &models.MarketQuote{
		Ticker:    ticker,
		Exchange:  exchange,
		Timestamp: time.Now(),
	}

	// безопасно извлекам значения
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

	// группируем тикеры по engine/market (акции, облигации, валюты — разные эндпоинты)
	type marketKey struct {
		engine string
		market string
	}
	grouped := make(map[marketKey][]string)

	for _, ticker := range tickers {
		engine, market, _ := p.detectMarket(ticker)
		key := marketKey{engine, market}
		grouped[key] = append(grouped[key], ticker)
	}

	// делаем отдельный запрос для каждой группы
	var lastErr error
	for key, groupTickers := range grouped {
		tickerList := strings.Join(groupTickers, ",")

		url := fmt.Sprintf("%s/engines/%s/markets/%s/securities.json?iss.meta=off&securities=%s",
			p.baseURL, key.engine, key.market, tickerList)

		resp, err := p.makeRequest(ctx, url)
		if err != nil {
			// Запоминаем последнюю ошибку, продолжаем с другими группами
			lastErr = fmt.Errorf("ошибка получения котировок %s/%s: %w", key.engine, key.market, err)
			continue
		}

		mdCols := makeColumnIndex(resp.Marketdata.Columns)
		secCols := makeColumnIndex(resp.Securities.Columns)

		for _, data := range resp.Marketdata.Data {
			var ticker string
			if secIdx, ok := mdCols["SECID"]; ok && secIdx < len(data) {
				ticker, _ = data[secIdx].(string)
			}
			if ticker == "" {
				continue // пропускаем записи без тикера(защита от битых данных)
			}

			quote := &models.MarketQuote{
				Ticker:    ticker,
				Exchange:  exchange,
				Timestamp: time.Now(),
			}

			quote.LastPrice = p.getDecimal(data, mdCols, "LAST", "CURRENTVALUE")
			quote.Change = p.getDecimal(data, mdCols, "CHANGE")
			quote.ChangePercent = p.getDecimal(data, mdCols, "LASTTOPREVPRICE")
			quote.Open = p.getDecimal(data, mdCols, "OPEN", "OPENPERIODPRICE")
			quote.High = p.getDecimal(data, mdCols, "HIGH")
			quote.Low = p.getDecimal(data, mdCols, "LOW")
			quote.Close = p.getDecimal(data, mdCols, "CLOSE", "CLOSEPRICE", "LCLOSEPRICE")
			quote.Bid = p.getDecimal(data, mdCols, "BID")
			quote.Ask = p.getDecimal(data, mdCols, "OFFER")

			if v, ok := mdCols["VOLTODAY"]; ok && v < len(data) {
				if vol, ok := data[v].(float64); ok {
					quote.Volume = int64(vol)
				}
			}

			result[ticker] = quote
		}

		// дополняем информацией из данных о ценных бумагах
		for _, data := range resp.Securities.Data {
			var ticker string
			if secIdx, ok := secCols["SECID"]; ok && secIdx < len(data) {
				ticker, _ = data[secIdx].(string)
			}
			if ticker == "" {
				continue // пропускаем записи без тикера(защита от битых данных)
			}

			if quote, exists := result[ticker]; exists {
				if quote.LastPrice.IsZero() {
					quote.LastPrice = p.getDecimal(data, secCols, "PREVPRICE", "PREVADMITTEDQUOTE") // фоллбэк: используем цену закрытия если тек котирвоки нет
				}
			}
		}
	}

	// если ничего не получили и была ошибка — возвращаем её
	if len(result) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return result, nil
}

func (p *MOEXProvider) SearchSecurities(ctx context.Context, query string, securityType *models.SecurityType, exchange models.Exchange) ([]models.Security, error) {
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
			Exchange: exchange,
			IsActive: true,
			Country:  "RU",
			LotSize:  1,
		}

		security.Ticker = p.getString(data, cols, "secid")
		if security.Ticker == "" {
			continue // пропускаем записи без тикера(защита от битых данных)
		}

		security.Name = p.getString(data, cols, "name")
		security.ShortName = p.getString(data, cols, "shortname")
		security.ISIN = p.getString(data, cols, "isin")

		group := p.getString(data, cols, "group")
		security.Type = p.mapSecurityType(group)

		// если не соответствует указанному фильтру пропускаем
		if securityType != nil && security.Type != *securityType {
			continue
		}

		// валюта из MOEX (если нет — по умолчанию RUB)
		security.Currency = p.getString(data, cols, "CURRENCYID", "CURRENCY", "currencyid", "currency")
		if security.Currency == "" {
			security.Currency = "RUB"
		}

		// размер лота (если доступен)
		if v := p.getFloat(data, cols, "lotsize"); v > 0 {
			security.LotSize = int(v)
		}

		// Минимальный шаг цены
		security.MinPriceIncrement = decimal.NewFromFloat(p.getFloat(data, cols, "minstep"))

		// сектор и индустрия (если доступны)
		security.Sector = p.getString(data, cols, "sector")
		security.Industry = p.getString(data, cols, "industry")

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
		return nil, fmt.Errorf("ценная бумага не найдена: %s", ticker)
	}

	cols := makeColumnIndex(resp.Securities.Columns)
	data := resp.Securities.Data[0]

	currency := p.getString(data, cols, "CURRENCYID", "CURRENCY", "currencyid", "currency")
	if currency == "" {
		currency = "RUB"
	}

	security := &models.Security{
		ID:       uuid.New(),
		Ticker:   ticker,
		Exchange: exchange,
		IsActive: true,
		Country:  "RU",
		Currency: currency,
		LotSize:  1,
	}

	security.Name = p.getString(data, cols, "name")
	security.ShortName = p.getString(data, cols, "shortname")
	security.ISIN = p.getString(data, cols, "isin")

	group := p.getString(data, cols, "group")
	security.Type = p.mapSecurityType(group)

	// сектор и индустрия
	security.Sector = p.getString(data, cols, "sector")
	security.Industry = p.getString(data, cols, "industry")

	// получаем размер лота
	if v := p.getFloat(data, cols, "lotsize"); v > 0 {
		security.LotSize = int(v)
	}

	// получаем минимальный шаг цены
	security.MinPriceIncrement = decimal.NewFromFloat(p.getFloat(data, cols, "minstep"))

	// специфичные поля для облигаций
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

		// частота выплат купонов
		if freq := p.getFloat(data, cols, "couponfrequency"); freq > 0 {
			f := int(freq)
			security.CouponFreq = &f
		}
	}

	// для ETF
	if security.Type == models.SecurityTypeETF {
		if ratio := p.getFloat(data, cols, "expenseratio"); ratio > 0 {
			er := decimal.NewFromFloat(ratio)
			security.ExpenseRatio = &er
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
			ID:       uuid.New(),
			Currency: "RUB",
		}

		// cумма дивиденда на акцию
		dividend.Amount = decimal.NewFromFloat(p.getFloat(data, cols, "value", "dividendvalue"))

		// дата закрытия реестра (record date)
		if dateStr := p.getString(data, cols, "registryclosedate", "recorddate"); dateStr != "" {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dividend.RecordDate = t
			}
		}

		// экс-дивидендная дата (обычно за 2 дня до закрытия реестра по российским правилам)
		if dateStr := p.getString(data, cols, "exdividenddate"); dateStr != "" {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dividend.ExDate = t
			}
		} else if !dividend.RecordDate.IsZero() {
			// если нет exdate — вычисляем минус 2 дня (упрощенно) (там на самом деле надо учитываь праздничные и выходнфе дни но мы не запариваемся)
			dividend.ExDate = dividend.RecordDate.AddDate(0, 0, -2)
		}

		// дата выплаты
		if dateStr := p.getString(data, cols, "paymentdate"); dateStr != "" {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dividend.PaymentDate = t
			}
		}

		// тип дивиденда
		divType := p.getString(data, cols, "dividendtype", "type")
		if divType != "" {
			dividend.DividendType = divType
		} else {
			dividend.DividendType = "regular"
		}

		// валюта (если есть)
		if curr := p.getString(data, cols, "currencyid", "currency"); curr != "" {
			dividend.Currency = curr
		}

		dividends = append(dividends, dividend)
	}

	return dividends, nil
}

func (p *MOEXProvider) GetCurrencyRate(ctx context.Context, from, to string) (decimal.Decimal, error) {
	// обрабатываем пары с рублём через валютный рынок MOEX (тоже упрощенно)
	var ticker string
	var invert bool // moex api предоставляем currency к рублю, обратно прилется самому

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
		return decimal.Zero, fmt.Errorf("неподдерживаемая валютная пара: %s/%s", from, to)
	}

	url := fmt.Sprintf("%s/engines/currency/markets/selt/boards/CETS/securities/%s.json?iss.meta=off", p.baseURL, ticker)

	resp, err := p.makeRequest(ctx, url)
	if err != nil {
		return decimal.Zero, err
	}

	if len(resp.Marketdata.Data) == 0 {
		return decimal.Zero, fmt.Errorf("нет данных по курсу для %s/%s", from, to)
	}

	cols := makeColumnIndex(resp.Marketdata.Columns)
	data := resp.Marketdata.Data[0]

	rate := p.getDecimal(data, cols, "LAST", "WAPRICE")
	if rate.IsZero() {
		return decimal.Zero, fmt.Errorf("не удалось получить курс для %s/%s", from, to)
	}

	if invert {
		rate = decimal.NewFromInt(1).Div(rate)
	}

	return rate, nil
}

// вспомогаттельные методы

// метод запроса
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
		return nil, fmt.Errorf("ошибка MOEX API: %d", resp.StatusCode)
	}

	var result MOEXResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// упрощенно определяем параметры для url ISS API по тикеру
func (p *MOEXProvider) detectMarket(ticker string) (engine, market, board string) {
	upperTicker := strings.ToUpper(ticker)

	// облигации
	if strings.HasPrefix(upperTicker, "SU") || strings.HasPrefix(upperTicker, "RU") {
		return "stock", "bonds", "TQOB"
	}

	// валюта
	if strings.HasPrefix(upperTicker, "RUB") || strings.Contains(upperTicker, "USD000") || strings.Contains(upperTicker, "EUR_RUB") {
		return "currency", "selt", "CETS"
	}

	//ETF
	if len(ticker) == 4 && strings.HasSuffix(upperTicker, "F") {
		return "stock", "shares", "TQTF"
	}

	// по умолч рынок акций
	return "stock", "shares", "TQBR"
}

// приводит строку из iss moex к типам models.SecurityType
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

// безопасное приведение к строке
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

// безопасное приведение к float
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

// безопасное приведение к decimal
func (p *MOEXProvider) getDecimal(data []interface{}, cols map[string]int, keys ...string) decimal.Decimal {
	v := p.getFloat(data, cols, keys...)
	return decimal.NewFromFloat(v)
}

// создаем индексы колонок, чтобы время поиска было O(1)
func makeColumnIndex(columns []string) map[string]int {
	idx := make(map[string]int)
	for i, col := range columns {
		idx[col] = i
	}
	return idx
}
