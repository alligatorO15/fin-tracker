package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionType string

const (
	TransactionTypeIncome   TransactionType = "income"
	TransactionTypeExpense  TransactionType = "expense"
	TransactionTypeTransfer TransactionType = "transfer"
)

type Transaction struct {
	ID                  uuid.UUID        `json:"id" db:"id"`
	UserID              uuid.UUID        `json:"user_id" db:"user_id"`
	AccountID           uuid.UUID        `json:"account_id" db:"account_id"` //счет владельца: income(счет зачисления), expense(счет списания),transfer(счет отправителя)
	CategoryID          uuid.UUID        `json:"category_id" db:"category_id"`
	Type                TransactionType  `json:"type" db:"type"`
	Amount              decimal.Decimal  `json:"amount" db:"amount"`
	Currency            string           `json:"currency" db:"currency"`
	Description         string           `json:"description" db:"description"`
	Date                time.Time        `json:"date" db:"date"`
	ToAccountID         *uuid.UUID       `json:"to_account_id,omitempty" db:"to_account_id"`                 //таргет счет                 //акк тому кому перевели
	ToAmount            *decimal.Decimal `json:"to_amount,omitempty" db:"to_amount"`                         // сума которая отображается у него на счете(может зависеть от валюты)
	IsRecurring         bool             `json:"is_recurring" db:"is_recurring"`                             //периодические платежи
	RecurrenceRule      string           `json:"recurrence_rule,omitempty" db:"recurrence_rule"`             // правило чтобы автоматизировать платтежи
	ParentTransactionID *uuid.UUID       `json:"parent_transaction_id,omitempty" db:"parent_transaction_id"` // ссылка на род транзакцию(оригинал) для повторяющихся
	//метаданные для сортировки и деталей (теги, и т.д.)
	Tags        []string `json:"tags" db:"-"` //теги для категоризации
	Location    string   `json:"location" db:"location"`
	Notes       string   `json:"notes" db:"notes"`
	Attachments []string `json:"attachments" db:"-"` //ссылки на прикрепленные файлы(отчётности и т.п.)
	//время аудит
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"-" db:"deleted_at"`
	//связанные данные(заполняются например при чтении из бд с join)
	Account   *Account  `json:"account,omitempty"`
	Category  *Category `json:"category,omitempty"`
	ToAccount *Account  `json:"to_account,omitempty"`
}

type TransactionCreate struct {
	AccountID      uuid.UUID        `json:"account_id" binding:"required"`
	CategoryID     uuid.UUID        `json:"category_id" binding:"required"`
	Type           TransactionType  `json:"type" binding:"required"`
	Amount         decimal.Decimal  `json:"amount" binding:"required"`
	Description    string           `json:"description"`
	Date           time.Time        `json:"date" binding:"required"`
	ToAccountID    *uuid.UUID       `json:"to_account_id"`
	ToAmount       *decimal.Decimal `json:"to_amount"`
	IsRecurring    bool             `json:"is_recurring"`
	RecurrenceRule string           `json:"recurrence_rule"`
	Tags           []string         `json:"tags"`
	Location       string           `json:"location"`
	Notes          string           `json:"notes"`
}

type TransactionUpdate struct {
	AccountID   *uuid.UUID       `json:"account_id"`
	CategoryID  *uuid.UUID       `json:"category_id"`
	Amount      *decimal.Decimal `json:"amount"`
	Description *string          `json:"description"`
	Date        *time.Time       `json:"date"`
	ToAccountID *uuid.UUID       `json:"to_account_id"`
	ToAmount    *decimal.Decimal `json:"to_amount"`
	Tags        []string         `json:"tags"`
	Location    *string          `json:"location"`
	Notes       *string          `json:"notes"`
}

type TransactionFilter struct {
	AccountID  *uuid.UUID       `form:"account_id"`
	CategoryID *uuid.UUID       `form:"category_id"`
	Type       *TransactionType `form:"type"`
	DateFrom   *time.Time       `form:"date_from"`  //транзакции с этой даты
	DateTo     *time.Time       `form:"date_to"`    // по эту дату
	AmountMin  *decimal.Decimal `form:"amount_min"` //мин сумма
	AmountMax  *decimal.Decimal `form:"amount_max"` //макс сумма
	Search     string           `form:"search"`     //по description или notes
	Tags       []string         `form:"tags"`
	Page       int              `form:"page"`       //пагинация номер стр
	Limit      int              `form:"limit"`      //пагинация кол-во на стр
	SortBy     string           `form:"sort_by"`    //?sort_by=date
	SortOrder  string           `form:"sort_order"` //?sort_order=desc
}

// структура пагинированного ответа
type TransactionList struct {
	Transactions []Transaction `json:"transactions"`
	Total        int64         `json:"total"` //всего тарнзакций
	Page         int           `json:"page"`
	Limit        int           `json:"limit"`
	TotalPages   int           `json:"total_pages"` //всего страниц
}
