package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountType string

const (
	AccountTypeCash       AccountType = "cash"
	AccountTypeBank       AccountType = "bank"
	AccountTypeCredit     AccountType = "credit"
	AccountTypeInvestment AccountType = "investment"
	AccountTypeCrypto     AccountType = "crypto"
	AccountTypeDebt       AccountType = "debt"
)

type Account struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	UserID         uuid.UUID       `json:"user_id" db:"user_id"`
	Name           string          `json:"name" db:"name"`
	Type           AccountType     `json:"type" db:"type"`
	Currency       string          `json:"currency" db:"currency"`
	Balance        decimal.Decimal `json:"balance" db:"balance"`
	InitialBalance decimal.Decimal `json:"initial_balance" db:"initial_balance"`
	Icon           string          `json:"icon" db:"icon"`
	Color          string          `json:"color" db:"color"`
	IsActive       bool            `json:"is_active" db:"is_active"`
	Institution    string          `json:"institution" db:"institution"`
	AccountNumber  string          `json:"account_number" db:"account_number"`
	Notes          string          `json:"notes" db:"notes"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time      `json:"-" db:"deleted_at"`
}

type AccountCreate struct {
	Name           string          `json:"name" binding:"required"`
	Type           AccountType     `json:"type" binding:"required"`
	Currency       string          `json:"currency" binding:"required"`
	InitialBalance decimal.Decimal `json:"initial_balance"`
	Icon           string          `json:"icon"`
	Color          string          `json:"color"`
	Institution    string          `json:"institution"`
	AccountNumber  string          `json:"account_number"`
	Notes          string          `json:"notes"`
}

type AccountUpdate struct {
	Name          *string `json:"name"`
	Icon          *string `json:"icon"`
	Color         *string `json:"color"`
	IsActive      *bool   `json:"is_active"`
	Institution   *string `json:"institution"`
	AccountNumber *string `json:"account_number"`
	Notes         *string `json:"notes"`
}

type AccountSummary struct {
	TotalBalance      decimal.Decimal            `json:"total_balance"`
	BalanceByCurrency map[string]decimal.Decimal `json:"balance_by_currency"`
	AccountsByType    map[AccountType]int        `json:"accounts_by_type"`
	Accounts          []Account                  `json:"accounts"`
}
