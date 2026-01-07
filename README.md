# FinTracker

FinTracker — комплексная платформа для управления личными финансами и инвестициями на Go.

## 🚀 Возможности

### Управление финансами
- **Счета** — поддержка нескольких счетов (наличные, банковские карты, кредиты, инвестиционные)
- **Транзакции** — учет доходов и расходов с категоризацией
- **Бюджеты** — планирование и контроль расходов по категориям
- **Цели** — постановка финансовых целей и отслеживание прогресса
- **Аналитика** — детальные отчеты и рекомендации

### Инвестиционный модуль
- **Портфели** — создание и управление инвестиционными портфелями
- **Ценные бумаги** — акции, облигации, ETF, криптовалюты
- **Московская биржа (MOEX)** — интеграция с российским рынком
- **Иностранные биржи** — поддержка NYSE, NASDAQ, LSE (при снятии санкций)
- **Котировки в реальном времени** — актуальные цены с бирж
- **Дивиденды** — отслеживание и уведомления
- **Налоговые отчеты** — расчет налогов по сделкам

## 📋 Требования

- Go 1.23+
- PostgreSQL 14+
- Docker и Docker Compose (рекомендуется)

## 🛠 Установка

### Вариант 1: Docker Compose (рекомендуется)

```bash
# Клонировать и запустить
git clone https://github.com/your-username/fin-tracker.git
cd fin-tracker
docker-compose up -d
```

API доступен на `http://localhost:8080`

#### Docker команды

```bash
# Запустить все сервисы
docker-compose up -d

# Остановить
docker-compose down

# Посмотреть логи
docker-compose logs -f

# Логи только API
docker-compose logs -f app

# Пересобрать после изменений
docker-compose up -d --build

# Подключиться к PostgreSQL
docker-compose exec postgres psql -U fintracker -d fintracker

# Запустить с pgAdmin (http://localhost:5050)
docker-compose --profile tools up -d

# Полный сброс (удаляет данные!)
docker-compose down -v
```

### Вариант 2: Локальная установка

```bash
# 1. Клонирование
git clone https://github.com/your-username/fin-tracker.git
cd fin-tracker

# 2. Настройка окружения
cp env.example .env
# Отредактируйте .env

# 3. Создание БД
createdb fintracker

# 4. Запуск
go mod download
go run cmd/server/main.go
```

Сервер запустится на `http://localhost:8080`

## 📚 API Документация

### Аутентификация

```bash
# Регистрация
POST /api/v1/auth/register
{
  "email": "user@example.com",
  "password": "securepassword",
  "first_name": "Иван",
  "last_name": "Иванов"
}

# Вход
POST /api/v1/auth/login
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

### Счета

```bash
# Создание счета
POST /api/v1/accounts
Authorization: Bearer <token>
{
  "name": "Основной счет",
  "type": "bank",
  "currency": "RUB",
  "initial_balance": 100000
}

# Список счетов
GET /api/v1/accounts

# Сводка по счетам
GET /api/v1/accounts/summary
```

### Транзакции

```bash
# Создание транзакции
POST /api/v1/transactions
{
  "account_id": "uuid",
  "category_id": "uuid",
  "type": "expense",
  "amount": 1500,
  "description": "Продукты в магазине",
  "date": "2024-01-15"
}

# Список транзакций с фильтрами
GET /api/v1/transactions?type=expense&date_from=2024-01-01&limit=50
```

### Бюджеты

```bash
# Создание бюджета
POST /api/v1/budgets
{
  "name": "Продукты",
  "category_id": "uuid",
  "amount": 30000,
  "currency": "RUB",
  "period": "monthly",
  "start_date": "2024-01-01"
}

# Сводка по бюджетам
GET /api/v1/budgets/summary

# Уведомления о превышении
GET /api/v1/budgets/alerts
```

### Инвестиции

```bash
# Поиск ценных бумаг
GET /api/v1/investments/securities/search?q=SBER&exchange=MOEX

# Получение котировки
GET /api/v1/investments/securities/SBER/quote?exchange=MOEX

# Создание портфеля
POST /api/v1/portfolios
{
  "name": "Мой портфель",
  "currency": "RUB",
  "broker_name": "Тинькофф"
}

# Добавление сделки
POST /api/v1/investments/transactions
{
  "portfolio_id": "uuid",
  "security_id": "uuid",
  "type": "buy",
  "date": "2024-01-15",
  "quantity": 10,
  "price": 250.50,
  "commission": 50
}

# Аналитика портфеля
GET /api/v1/investments/portfolios/{id}/analytics

# Налоговый отчет
GET /api/v1/investments/portfolios/{id}/tax-report?year=2024
```

### Аналитика

```bash
# Финансовая сводка
GET /api/v1/analytics/summary?period=month

# Денежный поток
GET /api/v1/analytics/cashflow?period=year

# Тренды расходов
GET /api/v1/analytics/trends?months=6

# Чистая стоимость активов
GET /api/v1/analytics/networth

# Финансовое здоровье
GET /api/v1/analytics/health

# Рекомендации
GET /api/v1/analytics/recommendations
```

## 🏗 Архитектура

```
fin-tracker/
├── cmd/
│   └── server/
│       └── main.go              # Точка входа
├── internal/
│   ├── api/
│   │   ├── handlers/            # HTTP обработчики
│   │   ├── middleware/          # Middleware (auth, cors, logging)
│   │   └── server.go            # Маршрутизация
│   ├── config/                  # Конфигурация
│   ├── database/                # Подключение к БД и миграции
│   ├── market/                  # Провайдеры рыночных данных
│   │   ├── moex.go              # Московская биржа
│   │   ├── foreign.go           # Иностранные биржи
│   │   └── crypto.go            # Криптовалюты
│   ├── models/                  # Модели данных
│   ├── repository/              # Слой работы с БД
│   └── service/                 # Бизнес-логика
├── Dockerfile                   # Сборка образа
├── docker-compose.yml           # Dev окружение
├── docker-compose.prod.yml      # Production настройки
├── env.example                  # Пример переменных окружения
├── go.mod
└── README.md
```

## 🌍 Поддержка рынков

### Российский рынок (активен)
- **MOEX** — Московская биржа (акции, облигации, ETF)
- **SPB** — Санкт-Петербургская биржа

### Иностранные рынки (при снятии санкций)
- **NYSE** — Нью-Йоркская фондовая биржа
- **NASDAQ** — Американская технологическая биржа
- **LSE** — Лондонская фондовая биржа
- **FRA** — Франкфуртская биржа
- **HKEX** — Гонконгская биржа

### Криптовалюты (всегда доступны)
- Bitcoin, Ethereum, и другие через CoinGecko API

## 🔧 Конфигурация

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `PORT` | Порт сервера | 8080 |
| `DATABASE_URL` | URL подключения к PostgreSQL | - |
| `JWT_SECRET` | Секретный ключ для JWT | - |
| `MOEX_ENABLED` | Включить интеграцию с MOEX | true |
| `FOREIGN_ENABLED` | Включить иностранные биржи | false |
| `DEFAULT_CURRENCY` | Валюта по умолчанию | RUB |

## 📊 Категории по умолчанию

### Доходы
- 💰 Зарплата
- 💻 Фриланс
- 📈 Инвестиции
- 💵 Дивиденды
- 🎁 Подарки

### Расходы
- 🛒 Продукты
- 🍽️ Рестораны
- 🚗 Транспорт
- 🏠 Жилье
- 💡 Коммунальные услуги
- 🏥 Здоровье
- 🎬 Развлечения
- 📚 Образование
- ✈️ Путешествия

## 🐳 Docker

### Структура контейнеров

```
┌─────────────────────────────────────────┐
│        Docker Network                    │
│                                          │
│  ┌────────────┐    ┌────────────┐       │
│  │ fintracker │───▶│ PostgreSQL │       │
│  │   :8080    │    │   :5432    │       │
│  └────────────┘    └────────────┘       │
│                          │               │
│                    ┌─────▼─────┐        │
│                    │  Volume   │        │
│                    │   data    │        │
│                    └───────────┘        │
└─────────────────────────────────────────┘
```

### Production деплой

```bash
# Установить секреты
export JWT_SECRET="ваш-секретный-ключ-минимум-32-символа"
export POSTGRES_PASSWORD="надёжный-пароль-для-бд"

# Запустить с production конфигом
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## 🔧 Технологии

- **Go 1.23+** — язык программирования
- **Gin** — HTTP веб-фреймворк
- **pgx** — PostgreSQL драйвер с пулом соединений
- **JWT** — аутентификация
- **shopspring/decimal** — точная арифметика для финансов
- **Docker** — контейнеризация

## 🔐 Безопасность

- JWT аутентификация
- Хеширование паролей (bcrypt)
- CORS защита
- Prepared statements для защиты от SQL-инъекций
- pgx — безопасный драйвер с защитой от SQL-инъекций

## 📝 Лицензия

MIT License

## 🤝 Контрибуция

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

 