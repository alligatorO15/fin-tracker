package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BudgetPeriod string

const (
	BudgetPeriodWeekly    BudgetPeriod = "weekly"
	BudgetPeriodMonthly   BudgetPeriod = "monthly"
	BudgetPeriodQuarterly BudgetPeriod = "quarterly"
	BudgetPeriodYearly    BudgetPeriod = "yearly"
	BudgetPeriodCustom    BudgetPeriod = "custom" //кастомно как разница между StartDate и EndDate
)

type Budget struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	UserID       uuid.UUID       `json:"user_id" db:"user_id"`
	CategoryID   *uuid.UUID      `json:"category_id" db:"category_id"`
	Name         string          `json:"name" db:"name"`
	Amount       decimal.Decimal `json:"amount" db:"amount"`
	Currency     string          `json:"currency" db:"currency"`
	Period       BudgetPeriod    `json:"period" db:"period"`
	StartDate    time.Time       `json:"start_date" db:"start_date"`
	EndDate      *time.Time      `json:"end_date" db:"end_date"`
	IsActive     bool            `json:"is_active" db:"is_active"`
	AlertPercent int             `json:"alert_percent" db:"alert_percent"` // уведомляеь если достигло
	Notes        string          `json:"notes" db:"notes"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`

	// Вычисляются на лету
	Spent        decimal.Decimal `json:"spent" db:"-"`
	Remaining    decimal.Decimal `json:"remaining" db:"-"`
	SpentPercent float64         `json:"spent_percent" db:"-"`
	Category     *Category       `json:"category,omitempty"`
}

type BudgetCreate struct {
	CategoryID   *uuid.UUID      `json:"category_id"`
	Name         string          `json:"name" binding:"required"`
	Amount       decimal.Decimal `json:"amount" binding:"required"`
	Currency     string          `json:"currency" binding:"required"`
	Period       BudgetPeriod    `json:"period" binding:"required"`
	StartDate    time.Time       `json:"start_date" binding:"required"`
	EndDate      *time.Time      `json:"end_date"`
	AlertPercent int             `json:"alert_percent"`
	Notes        string          `json:"notes"`
}

type BudgetUpdate struct {
	CategoryID   *uuid.UUID       `json:"category_id"`
	Name         *string          `json:"name"`
	Amount       *decimal.Decimal `json:"amount"`
	Period       *BudgetPeriod    `json:"period"`
	StartDate    *time.Time       `json:"start_date"`
	EndDate      *time.Time       `json:"end_date"`
	IsActive     *bool            `json:"is_active"`
	AlertPercent *int             `json:"alert_percent"`
	Notes        *string          `json:"notes"`
}

type BudgetSummary struct {
	TotalBudgeted   decimal.Decimal `json:"total_budgeted"`
	TotalSpent      decimal.Decimal `json:"total_spent"`
	TotalRemaining  decimal.Decimal `json:"total_remaining"`
	OverBudgetCount int             `json:"over_budget_count"`
	Budgets         []Budget        `json:"budgets"`
}

type BudgetAlert struct {
	BudgetID   uuid.UUID       `json:"budget_id"`
	BudgetName string          `json:"budget_name"`
	Amount     decimal.Decimal `json:"amount"`
	Spent      decimal.Decimal `json:"spent"`
	Percent    float64         `json:"percent"`
	AlertType  string          `json:"alert_type"`
}
