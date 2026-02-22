package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

type OllamaClient struct {
	baseURL string
	model   string
	client  *http.Client
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetFinancialAdvice генерирует рекомендации на основе финансовых данных
func (c *OllamaClient) GetFinancialAdvice(ctx context.Context, data FinancialSummary) (string, error) {
	prompt := c.buildPrompt(data)
	return c.generate(ctx, prompt)
}

type FinancialSummary struct {
	TotalIncome   decimal.Decimal    `json:"total_income"`
	TotalExpenses decimal.Decimal    `json:"total_expenses"`
	Balance       decimal.Decimal    `json:"balance"`
	TopCategories []CategorySpending `json:"top_categories"`
	SavingsRate   decimal.Decimal    `json:"savings_rate"`
	BudgetStatus  []BudgetStatus     `json:"budget_status"`
	Currency      string             `json:"currency"`
}

type CategorySpending struct {
	Name   string          `json:"name"`
	Amount decimal.Decimal `json:"amount"`
}

type BudgetStatus struct {
	Category string          `json:"category"`
	Limit    decimal.Decimal `json:"limit"`
	Spent    decimal.Decimal `json:"spent"`
	Percent  decimal.Decimal `json:"percent"`
}

func (c *OllamaClient) buildPrompt(data FinancialSummary) string {
	return fmt.Sprintf(`Ты финансовый консультант. Проанализируй данные пользователя и дай 3-5 кратких рекомендаций на русском языке.

Финансовые данные за месяц:
- Доходы: %s %s
- Расходы: %s %s
- Баланс: %s %s
- Норма сбережений: %s%%

Топ категории расходов:
%s

Статус бюджетов:
%s

Дай конкретные, практичные рекомендации. Отвечай кратко, по делу.`,
		data.TotalIncome.StringFixed(2), data.Currency,
		data.TotalExpenses.StringFixed(2), data.Currency,
		data.Balance.StringFixed(2), data.Currency,
		data.SavingsRate.StringFixed(1),
		formatCategories(data.TopCategories, data.Currency),
		formatBudgets(data.BudgetStatus, data.Currency),
	)
}

func formatCategories(categories []CategorySpending, currency string) string {
	if len(categories) == 0 {
		return "Нет данных"
	}
	var result string
	for _, c := range categories {
		result += fmt.Sprintf("- %s: %s %s\n", c.Name, c.Amount.StringFixed(2), currency)
	}
	return result
}

func formatBudgets(budgets []BudgetStatus, currency string) string {
	if len(budgets) == 0 {
		return "Бюджеты не установлены"
	}
	hundred := decimal.NewFromInt(100)
	eighty := decimal.NewFromInt(80)
	var result string
	for _, b := range budgets {
		status := "✓"
		if b.Percent.GreaterThan(hundred) {
			status = "⚠️ превышен"
		} else if b.Percent.GreaterThan(eighty) {
			status = "⚡ близко к лимиту"
		}
		result += fmt.Sprintf("- %s: %s/%s %s (%s%%) %s\n",
			b.Category, b.Spent.StringFixed(2), b.Limit.StringFixed(2), currency, b.Percent.StringFixed(0), status)
	}
	return result
}

func (c *OllamaClient) generate(ctx context.Context, prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Response, nil
}

// IsAvailable проверяет доступность Ollama
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
