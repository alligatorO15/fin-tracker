package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(pool *pgxpool.Pool) error {
	log.Println("Running database migrations...")

	ctx := context.Background()

	migrations := []string{
		migrationCreateExtensions,
		migrationCreateUsers,
		migrationCreateRefreshTokens,
		migrationCreateAccounts,
		migrationCreateCategories,
		migrationCreateTransactions,
		migrationCreateBudgets,
		migrationCreateGoals,
		migrationCreateSecurities,
		migrationCreatePortfolios,
		migrationCreateHoldings,
		migrationCreateInvestmentTransactions,
		migrationCreateDividends,
		migrationCreateIndexes,
		migrationInsertDefaultCategories,
	}

	for i, migration := range migrations {
		if _, err := pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	log.Println("Migrations completed successfully")
	return nil
}

const migrationCreateExtensions = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
`

const migrationCreateUsers = `
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100),
    default_currency VARCHAR(3) DEFAULT 'RUB',
    timezone VARCHAR(50) DEFAULT 'Europe/Moscow',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
`

const migrationCreateRefreshTokens = `
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
`

const migrationCreateAccounts = `
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    balance DECIMAL(18, 2) DEFAULT 0,
    initial_balance DECIMAL(18, 2) DEFAULT 0,
    icon VARCHAR(10),
    color VARCHAR(7),
    is_active BOOLEAN DEFAULT true,
    institution VARCHAR(100),
    account_number VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
`

const migrationCreateCategories = `
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    icon VARCHAR(10),
    color VARCHAR(7),
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    is_system BOOLEAN DEFAULT false,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

const migrationCreateTransactions = `
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id),
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    description VARCHAR(500),
    date DATE NOT NULL,
    to_account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    to_amount DECIMAL(18, 2),
    is_recurring BOOLEAN DEFAULT false,
    recurrence_rule VARCHAR(100),
    parent_transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    location VARCHAR(200),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS transaction_tags (
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tag VARCHAR(50) NOT NULL,
    PRIMARY KEY (transaction_id, tag)
);

`

const migrationCreateBudgets = `
CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    period VARCHAR(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    is_active BOOLEAN DEFAULT true,
    alert_percent INTEGER DEFAULT 80,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

const migrationCreateGoals = `
CREATE TABLE IF NOT EXISTS goals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    target_amount DECIMAL(18, 2) NOT NULL,
    current_amount DECIMAL(18, 2) DEFAULT 0,
    currency VARCHAR(3) NOT NULL,
    target_date DATE,
    icon VARCHAR(10),
    color VARCHAR(7),
    status VARCHAR(20) DEFAULT 'active',
    priority INTEGER DEFAULT 0,
    auto_contribute BOOLEAN DEFAULT false,
    contribute_amount DECIMAL(18, 2) DEFAULT 0,
    contribute_freq VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS goal_contributions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    amount DECIMAL(18, 2) NOT NULL,
    date DATE NOT NULL,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

const migrationCreateSecurities = `
CREATE TABLE IF NOT EXISTS securities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticker VARCHAR(20) NOT NULL,
    isin VARCHAR(12),
    name VARCHAR(200) NOT NULL,
    short_name VARCHAR(50),
    type VARCHAR(20) NOT NULL,
    exchange VARCHAR(10) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    country VARCHAR(2),
    sector VARCHAR(100),
    industry VARCHAR(100),
    lot_size INTEGER DEFAULT 1,
    min_price_increment DECIMAL(18, 8) DEFAULT 0.01,
    is_active BOOLEAN DEFAULT true,
    face_value DECIMAL(18, 2),
    coupon_rate DECIMAL(8, 4),
    maturity_date DATE,
    coupon_freq INTEGER,
    expense_ratio DECIMAL(8, 4),
    last_price DECIMAL(18, 6) DEFAULT 0,
    price_change DECIMAL(18, 6) DEFAULT 0,
    price_change_percent DECIMAL(8, 4) DEFAULT 0,
    volume BIGINT DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(ticker, exchange)
);
`

const migrationCreatePortfolios = `
CREATE TABLE IF NOT EXISTS portfolios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    currency VARCHAR(3) NOT NULL,
    broker_name VARCHAR(100),
    broker_account VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

const migrationCreateHoldings = `
CREATE TABLE IF NOT EXISTS holdings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    security_id UUID NOT NULL REFERENCES securities(id),
    quantity DECIMAL(18, 8) NOT NULL,
    average_price DECIMAL(18, 6) NOT NULL,
    total_cost DECIMAL(18, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(portfolio_id, security_id)
);
`

const migrationCreateInvestmentTransactions = `
CREATE TABLE IF NOT EXISTS investment_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    security_id UUID NOT NULL REFERENCES securities(id),
    type VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    quantity DECIMAL(18, 8) NOT NULL,
    price DECIMAL(18, 6) NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    commission DECIMAL(18, 2) DEFAULT 0,
    currency VARCHAR(3) NOT NULL,
    exchange_rate DECIMAL(18, 6) DEFAULT 1,
    notes TEXT,
    broker_ref VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS broker_imports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    broker_type VARCHAR(50) NOT NULL,
    file_name VARCHAR(200) NOT NULL,
    import_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    period_start DATE,
    period_end DATE,
    status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,
    transactions_imported INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

const migrationCreateDividends = ``

const migrationCreateIndexes = `
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_budgets_user_id ON budgets(user_id);
CREATE INDEX IF NOT EXISTS idx_goals_user_id ON goals(user_id);
CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id);
CREATE INDEX IF NOT EXISTS idx_categories_type ON categories(type);
CREATE INDEX IF NOT EXISTS idx_portfolios_user_id ON portfolios(user_id);
CREATE INDEX IF NOT EXISTS idx_holdings_portfolio_id ON holdings(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_investment_transactions_portfolio_id ON investment_transactions(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_investment_transactions_date ON investment_transactions(date);
CREATE INDEX IF NOT EXISTS idx_securities_ticker ON securities(ticker);
CREATE INDEX IF NOT EXISTS idx_securities_exchange ON securities(exchange);
`

const migrationInsertDefaultCategories = `
INSERT INTO categories (id, name, type, icon, color, is_system, sort_order) VALUES
    -- Income
    (uuid_generate_v4(), 'Зарплата', 'income', '💰', '#4CAF50', true, 1),
    (uuid_generate_v4(), 'Фриланс', 'income', '💻', '#8BC34A', true, 2),
    (uuid_generate_v4(), 'Инвестиции', 'income', '📈', '#009688', true, 3),
    (uuid_generate_v4(), 'Дивиденды', 'income', '💵', '#00BCD4', true, 4),
    (uuid_generate_v4(), 'Подарки', 'income', '🎁', '#03A9F4', true, 5),
    (uuid_generate_v4(), 'Другой доход', 'income', '💸', '#2196F3', true, 6),
    -- Expenses
    (uuid_generate_v4(), 'Продукты', 'expense', '🛒', '#FF5722', true, 7),
    (uuid_generate_v4(), 'Рестораны', 'expense', '🍽️', '#FF9800', true, 8),
    (uuid_generate_v4(), 'Транспорт', 'expense', '🚗', '#FFC107', true, 9),
    (uuid_generate_v4(), 'Жилье', 'expense', '🏠', '#795548', true, 10),
    (uuid_generate_v4(), 'Коммунальные услуги', 'expense', '💡', '#607D8B', true, 11),
    (uuid_generate_v4(), 'Здоровье', 'expense', '🏥', '#E91E63', true, 12),
    (uuid_generate_v4(), 'Развлечения', 'expense', '🎬', '#9C27B0', true, 13),
    (uuid_generate_v4(), 'Покупки', 'expense', '🛍️', '#673AB7', true, 14),
    (uuid_generate_v4(), 'Образование', 'expense', '📚', '#3F51B5', true, 15),
    (uuid_generate_v4(), 'Путешествия', 'expense', '✈️', '#2196F3', true, 16),
    (uuid_generate_v4(), 'Подписки', 'expense', '📱', '#00BCD4', true, 17),
    (uuid_generate_v4(), 'Связь', 'expense', '📞', '#009688', true, 18),
    (uuid_generate_v4(), 'Домашние животные', 'expense', '🐕', '#4CAF50', true, 19),
    (uuid_generate_v4(), 'Другие расходы', 'expense', '📋', '#9E9E9E', true, 20),
    -- Transfer
    (uuid_generate_v4(), 'Перевод', 'transfer', '🔄', '#607D8B', true, 21)
ON CONFLICT DO NOTHING;
`
