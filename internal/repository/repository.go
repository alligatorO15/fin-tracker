package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repositories struct {
	TxManager    TxManager
	User         UserRepository
	RefreshToken RefreshTokenRepository
	Account      AccountRepository
	Category     CategoryRepository
	Transaction  TransactionRepository
	Budget       BudgetRepository
	Goal         GoalRepository
	Portfolio    PortfolioRepository
	Security     SecurityRepository
	Holding      HoldingRepository
	Investment   InvestmentTransactionRepository
}

func NewRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		TxManager:    NewTxManager(pool),
		User:         NewUserRepository(pool),
		RefreshToken: NewRefreshTokenRepository(pool),
		Account:      NewAccountRepository(pool),
		Category:     NewCategoryRepository(pool),
		Transaction:  NewTransactionRepository(pool),
		Budget:       NewBudgetRepository(pool),
		Goal:         NewGoalRepository(pool),
		Portfolio:    NewPortfolioRepository(pool),
		Security:     NewSecurityRepository(pool),
		Holding:      NewHoldingRepository(pool),
		Investment:   NewInvestmentTransactionRepository(pool),
	}
}
