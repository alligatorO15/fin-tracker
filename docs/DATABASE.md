# Структура базы данных FinTracker

## Обзор

База данных PostgreSQL 14+ с расширениями:
- `uuid-ossp` — генерация UUID
- `pgcrypto` — криптографические функции

## Таблицы

### Пользователи и авторизация

#### `users`
Пользователи системы.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `email` | VARCHAR(255) | Email (уникальный) |
| `password_hash` | VARCHAR(255) | Хеш пароля (bcrypt) |
| `first_name` | VARCHAR(100) | Имя |
| `last_name` | VARCHAR(100) | Фамилия |
| `default_currency` | VARCHAR(3) | Валюта по умолчанию (RUB) |
| `timezone` | VARCHAR(50) | Часовой пояс |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |
| `deleted_at` | TIMESTAMPTZ | Soft delete |

#### `refresh_tokens`
Токены для обновления JWT.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users |
| `token_hash` | VARCHAR(64) | Хеш токена (SHA-256) |
| `expires_at` | TIMESTAMPTZ | Срок действия |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `revoked_at` | TIMESTAMPTZ | Дата отзыва |

---

### Финансы

#### `accounts`
Счета пользователя (банк, наличные, кредит и т.д.).

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users |
| `name` | VARCHAR(100) | Название |
| `type` | VARCHAR(20) | Тип: cash, bank, credit, investment, crypto, debt |
| `currency` | VARCHAR(3) | Валюта |
| `balance` | DECIMAL(18,2) | Текущий баланс |
| `initial_balance` | DECIMAL(18,2) | Начальный баланс |
| `icon` | VARCHAR(10) | Иконка (emoji) |
| `color` | VARCHAR(7) | Цвет (#HEX) |
| `is_active` | BOOLEAN | Активен |
| `institution` | VARCHAR(100) | Банк/учреждение |
| `account_number` | VARCHAR(50) | Номер счёта |
| `notes` | TEXT | Заметки |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |
| `deleted_at` | TIMESTAMPTZ | Soft delete |

#### `categories`
Категории доходов/расходов.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users (NULL для системных) |
| `name` | VARCHAR(100) | Название |
| `type` | VARCHAR(20) | Тип: income, expense, transfer |
| `icon` | VARCHAR(10) | Иконка |
| `color` | VARCHAR(7) | Цвет |
| `parent_id` | UUID | FK → categories (для подкатегорий) |
| `is_system` | BOOLEAN | Системная категория |
| `sort_order` | INTEGER | Порядок сортировки |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |

#### `transactions`
Финансовые транзакции.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users |
| `account_id` | UUID | FK → accounts |
| `category_id` | UUID | FK → categories |
| `type` | VARCHAR(20) | Тип: income, expense, transfer |
| `amount` | DECIMAL(18,2) | Сумма |
| `currency` | VARCHAR(3) | Валюта |
| `description` | VARCHAR(500) | Описание |
| `date` | DATE | Дата |
| `to_account_id` | UUID | FK → accounts (для переводов) |
| `to_amount` | DECIMAL(18,2) | Сумма в валюте получателя |
| `is_recurring` | BOOLEAN | Повторяющаяся |
| `recurrence_rule` | VARCHAR(100) | Правило повторения |
| `parent_transaction_id` | UUID | FK → transactions (родительская транзакция) |
| `location` | VARCHAR(200) | Место |
| `notes` | TEXT | Заметки |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |
| `deleted_at` | TIMESTAMPTZ | Soft delete |

#### `transaction_tags`
Теги транзакций (many-to-many).

| Поле | Тип | Описание |
|------|-----|----------|
| `transaction_id` | UUID | FK → transactions |
| `tag` | VARCHAR(50) | Тег |

---

### Бюджеты и цели

#### `budgets`
Бюджеты по категориям.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users |
| `category_id` | UUID | FK → categories |
| `name` | VARCHAR(100) | Название |
| `amount` | DECIMAL(18,2) | Лимит |
| `currency` | VARCHAR(3) | Валюта |
| `period` | VARCHAR(20) | Период: weekly, monthly, quarterly, yearly, custom |
| `start_date` | DATE | Начало |
| `end_date` | DATE | Конец |
| `is_active` | BOOLEAN | Активен |
| `alert_percent` | INTEGER | Порог оповещения (%) |
| `notes` | TEXT | Заметки |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |

#### `goals`
Финансовые цели.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users |
| `account_id` | UUID | FK → accounts |
| `name` | VARCHAR(100) | Название |
| `description` | TEXT | Описание |
| `target_amount` | DECIMAL(18,2) | Целевая сумма |
| `current_amount` | DECIMAL(18,2) | Текущая сумма |
| `currency` | VARCHAR(3) | Валюта |
| `target_date` | DATE | Целевая дата |
| `icon` | VARCHAR(10) | Иконка |
| `color` | VARCHAR(7) | Цвет |
| `status` | VARCHAR(20) | Статус: active, completed, cancelled, paused |
| `priority` | INTEGER | Приоритет |
| `auto_contribute` | BOOLEAN | Автопополнение |
| `contribute_amount` | DECIMAL(18,2) | Сумма пополнения |
| `contribute_freq` | VARCHAR(20) | Частота: daily, weekly, monthly |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |
| `completed_at` | TIMESTAMPTZ | Дата завершения |

#### `goal_contributions`
История пополнений целей.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `goal_id` | UUID | FK → goals |
| `amount` | DECIMAL(18,2) | Сумма |
| `date` | DATE | Дата |
| `notes` | TEXT | Заметки |
| `created_at` | TIMESTAMPTZ | Дата создания |

---

### Инвестиции

#### `securities`
Ценные бумаги и криптовалюты.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `ticker` | VARCHAR(20) | Тикер |
| `isin` | VARCHAR(12) | ISIN код |
| `name` | VARCHAR(200) | Название |
| `short_name` | VARCHAR(50) | Короткое название |
| `type` | VARCHAR(20) | Тип: stock, bond, etf, mutual_fund, crypto, currency, derivative |
| `exchange` | VARCHAR(10) | Биржа: MOEX, CRYPTO |
| `currency` | VARCHAR(3) | Валюта |
| `country` | VARCHAR(2) | Страна |
| `sector` | VARCHAR(100) | Сектор |
| `industry` | VARCHAR(100) | Отрасль |
| `lot_size` | INTEGER | Размер лота |
| `min_price_increment` | DECIMAL(18,6) | Минимальный шаг цены |
| `is_active` | BOOLEAN | Активна |
| `face_value` | DECIMAL(18,2) | Номинал (для облигаций) |
| `coupon_rate` | DECIMAL(8,4) | Купонная ставка |
| `coupon_freq` | INTEGER | Выплат купона в год |
| `maturity_date` | DATE | Дата погашения |
| `expense_ratio` | DECIMAL(8,4) | Комиссия ETF |
| `last_price` | DECIMAL(18,6) | Последняя цена |
| `price_change` | DECIMAL(18,6) | Изменение цены |
| `price_change_percent` | DECIMAL(8,4) | Изменение в % |
| `volume` | BIGINT | Объём торгов |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |

#### `portfolios`
Инвестиционные портфели.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users |
| `account_id` | UUID | FK → accounts |
| `name` | VARCHAR(100) | Название |
| `description` | TEXT | Описание |
| `currency` | VARCHAR(3) | Валюта |
| `broker_name` | VARCHAR(100) | Брокер |
| `broker_account` | VARCHAR(50) | Номер счёта |
| `is_active` | BOOLEAN | Активен |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |

#### `holdings`
Позиции в портфеле.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `portfolio_id` | UUID | FK → portfolios |
| `security_id` | UUID | FK → securities |
| `quantity` | DECIMAL(18,8) | Количество |
| `average_price` | DECIMAL(18,6) | Средняя цена |
| `total_cost` | DECIMAL(18,2) | Общая стоимость |
| `created_at` | TIMESTAMPTZ | Дата создания |
| `updated_at` | TIMESTAMPTZ | Дата обновления |

#### `investment_transactions`
Инвестиционные операции.

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | PK |
| `portfolio_id` | UUID | FK → portfolios |
| `security_id` | UUID | FK → securities |
| `type` | VARCHAR(20) | Тип: buy, sell, dividend, coupon, split, transfer_in, transfer_out, fee, tax |
| `date` | DATE | Дата |
| `quantity` | DECIMAL(18,8) | Количество |
| `price` | DECIMAL(18,6) | Цена |
| `amount` | DECIMAL(18,2) | Сумма |
| `commission` | DECIMAL(18,2) | Комиссия |
| `currency` | VARCHAR(3) | Валюта |
| `exchange_rate` | DECIMAL(18,6) | Курс валюты |
| `notes` | TEXT | Заметки |
| `broker_ref` | VARCHAR(100) | Референс из отчёта брокера |
| `created_at` | TIMESTAMPTZ | Дата создания |

---

## Индексы

```sql
-- Основные
idx_accounts_user_id
idx_transactions_user_id
idx_transactions_account_id
idx_transactions_category_id
idx_transactions_date
idx_transactions_type
idx_budgets_user_id
idx_goals_user_id
idx_categories_user_id
idx_categories_type

-- Инвестиции
idx_portfolios_user_id
idx_holdings_portfolio_id
idx_holdings_security_id
idx_investment_transactions_portfolio_id
idx_investment_transactions_date
idx_securities_ticker
idx_securities_exchange

-- Токены
idx_refresh_tokens_user_id
idx_refresh_tokens_token_hash
```

## Ограничения

- `users.email` — UNIQUE
- `securities(ticker, exchange)` — UNIQUE
- `holdings(portfolio_id, security_id)` — UNIQUE
- `transaction_tags(transaction_id, tag)` — PK
- Все FK имеют `ON DELETE CASCADE` или `ON DELETE SET NULL`
