package models

import (
	"time"

	"github.com/google/uuid"
)

type CategoryType string

const (
	CategoryTypeIncome   CategoryType = "income"
	CategoryTypeExpense  CategoryType = "expense"
	CategoryTypeTransfer CategoryType = "transfer"
)

type Category struct {
	ID        uuid.UUID    `json:"id" db:"id"`
	UserID    *uuid.UUID   `json:"usr_id" db:"user_id"` //nil будет если это системная категория
	Name      string       `json:"name" db:"name"`
	Type      CategoryType `json:"type" db:"type"`
	Icon      string       `json:"icon" db:"icon"`
	Color     string       `json:"color" db:"color"`
	ParentID  *uuid.UUID   `json:"parent_id" db:"parent_id"`
	IsSystem  bool         `json:"is_system" db:"is_system"`
	SortOrder int          `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`

	Children []Category `json:"children,omitempty"`
}

type CategoryCreate struct {
	Name     string       `json:"name" binding:"required"`
	Type     CategoryType `json:"type" binding:"required"`
	Icon     string       `json:"icon"`
	Color    string       `json:"color"`
	ParentID *uuid.UUID   `json:"parent_id"`
}

type CategoryUpdate struct {
	Name      *string    `json:"name"`
	Icon      *string    `json:"icon"`
	Color     *string    `json:"color"`
	ParentID  *uuid.UUID `json:"parent_id"`
	SortOrder *int       `json:"sort_order"`
}

// дефолтные системные категориии
var DefaultCategories = []Category{
	{Name: "Зарплата", Type: CategoryTypeIncome, Icon: "💵", Color: "#4CAF50", IsSystem: true},
	{Name: "Фриланс", Type: CategoryTypeIncome, Icon: "💻", Color: "#8BC34A", IsSystem: true},
	{Name: "Инвестиции", Type: CategoryTypeIncome, Icon: "📈", Color: "#009688", IsSystem: true},
	{Name: "Дивиденды", Type: CategoryTypeIncome, Icon: "💸", Color: "#00BCD4", IsSystem: true},
	{Name: "Подарки", Type: CategoryTypeIncome, Icon: "🎁", Color: "#03A9F4", IsSystem: true},
	{Name: "Другой доход", Type: CategoryTypeIncome, Icon: "💰", Color: "#2196F3", IsSystem: true},
	{Name: "Продукты", Type: CategoryTypeExpense, Icon: "🛒", Color: "#FF5722", IsSystem: true},
	{Name: "Рестораны", Type: CategoryTypeExpense, Icon: "🍽️", Color: "#FF9800", IsSystem: true},
	{Name: "Транспорт", Type: CategoryTypeExpense, Icon: "🚗", Color: "#FFC107", IsSystem: true},
	{Name: "Жилье", Type: CategoryTypeExpense, Icon: "🏠", Color: "#795548", IsSystem: true},
	{Name: "Коммунальные услуги", Type: CategoryTypeExpense, Icon: "💡", Color: "#607D8B", IsSystem: true},
	{Name: "Здоровье", Type: CategoryTypeExpense, Icon: "🏥", Color: "#E91E63", IsSystem: true},
	{Name: "Развлечения", Type: CategoryTypeExpense, Icon: "🎬", Color: "#9C27B0", IsSystem: true},
	{Name: "Покупки", Type: CategoryTypeExpense, Icon: "🛍️", Color: "#673AB7", IsSystem: true},
	{Name: "Образование", Type: CategoryTypeExpense, Icon: "📚", Color: "#3F51B5", IsSystem: true},
	{Name: "Путешествия", Type: CategoryTypeExpense, Icon: "✈️", Color: "#2196F3", IsSystem: true},
	{Name: "Подписки", Type: CategoryTypeExpense, Icon: "📱", Color: "#00BCD4", IsSystem: true},
	{Name: "Связь", Type: CategoryTypeExpense, Icon: "📞", Color: "#009688", IsSystem: true},
	{Name: "Домашние животные", Type: CategoryTypeExpense, Icon: "🐕", Color: "#4CAF50", IsSystem: true},
	{Name: "Другие расходы", Type: CategoryTypeExpense, Icon: "📋", Color: "#9E9E9E", IsSystem: true},
	{Name: "Перевод", Type: CategoryTypeTransfer, Icon: "💳", Color: "#607D8B", IsSystem: true},
}
