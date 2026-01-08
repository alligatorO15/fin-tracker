package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type GoalStatus string

const (
	GoalStatusActive    GoalStatus = "active"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusCancelled GoalStatus = "cancelled"
	GoalStatusPaused    GoalStatus = "paused"
)

type Goal struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	UserID           uuid.UUID       `json:"user_id" db:"user_id"`
	AccountID        *uuid.UUID      `json:"account_id" db:"account_id"`
	Name             string          `json:"name" db:"name"`
	Description      string          `json:"description" db:"description"`
	TargetAmount     decimal.Decimal `json:"target_amount" db:"target_amount"`
	CurrentAmount    decimal.Decimal `json:"current_amount" db:"current_amount"`
	Currency         string          `json:"currency" db:"currency"`
	TargetDate       *time.Time      `json:"target_date" db:"target_date"`
	Icon             string          `json:"icon" db:"icon"`
	Color            string          `json:"color" db:"color"`
	Status           GoalStatus      `json:"status" db:"status"`
	Priority         int             `json:"priority" db:"priority"`
	AutoContribute   bool            `json:"auto_contribute" db:"auto_contribute"`
	ContributeAmount decimal.Decimal `json:"contribute_amount" db:"contribute_amount"`
	ContributeFreq   string          `json:"contribute_freq" db:"contribute_freq"` // daily, weekly, monthly
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
	CompletedAt      *time.Time      `json:"completed_at" db:"completed_at"`

	// Вычисляются на лету
	Progress        float64         `json:"progress" db:"-"`
	DaysRemaining   int             `json:"days_remaining" db:"-"`
	RequiredMonthly decimal.Decimal `json:"required_monthly" db:"-"`
	Account         *Account        `json:"account,omitempty"`
}

type GoalCreate struct {
	AccountID        *uuid.UUID      `json:"account_id"`
	Name             string          `json:"name" binding:"required"`
	Description      string          `json:"description"`
	TargetAmount     decimal.Decimal `json:"target_amount" binding:"required"`
	CurrentAmount    decimal.Decimal `json:"current_amount"`
	Currency         string          `json:"currency" binding:"required"`
	TargetDate       *time.Time      `json:"target_date"`
	Icon             string          `json:"icon"`
	Color            string          `json:"color"`
	Priority         int             `json:"priority"`
	AutoContribute   bool            `json:"auto_contribute"`
	ContributeAmount decimal.Decimal `json:"contribute_amount"`
	ContributeFreq   string          `json:"contribute_freq"`
}

type GoalUpdate struct {
	AccountID        *uuid.UUID       `json:"account_id"`
	Name             *string          `json:"name"`
	Description      *string          `json:"description"`
	TargetAmount     *decimal.Decimal `json:"target_amount"`
	CurrentAmount    *decimal.Decimal `json:"current_amount"`
	TargetDate       *time.Time       `json:"target_date"`
	Icon             *string          `json:"icon"`
	Color            *string          `json:"color"`
	Status           *GoalStatus      `json:"status"`
	Priority         *int             `json:"priority"`
	AutoContribute   *bool            `json:"auto_contribute"`
	ContributeAmount *decimal.Decimal `json:"contribute_amount"`
	ContributeFreq   *string          `json:"contribute_freq"`
}

type GoalContribution struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	GoalID    uuid.UUID       `json:"goal_id" db:"goal_id"`
	Amount    decimal.Decimal `json:"amount" db:"amount"`
	Date      time.Time       `json:"date" db:"date"`
	Notes     string          `json:"notes" db:"notes"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

type GoalContributionCreate struct {
	Amount decimal.Decimal `json:"amount" binding:"required"`
	Date   time.Time       `json:"date"`
	Notes  string          `json:"notes"`
}
