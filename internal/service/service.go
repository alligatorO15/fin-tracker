package service

import (
	"github.com/alligatorO15/fin-tracker/internal/ai"
	"github.com/alligatorO15/fin-tracker/internal/config"
	"github.com/alligatorO15/fin-tracker/internal/market"
	"github.com/alligatorO15/fin-tracker/internal/repository"
)

type Services struct {
	Auth         AuthService
	User         UserService
	Account      AccountService
	Category     CategoryService
	Transaction  TransactionService
	Budget       BudgetService
	Goal         GoalService
	Portfolio    PortfolioService
	Investment   InvestmentService
	Analytics    AnalyticsService
	BrokerImport BrokerImportService
}

func NewServices(repos *repository.Repositories, marketProvider *market.MultiProvider, cfg *config.Config) *Services {
	var aiClient *ai.OllamaClient
	if cfg.OllamaURL != "" {
		aiClient = ai.NewOllamaClient(cfg.OllamaClient, cfg.OllamaModel)
	}

	return &Services{
		Auth:         NewAuthService(repos.User, repos.RefreshToken, cfg),
		User:         NewUserService(repos.User),
		Account:      NewAccountService(repos.Account, repos.User, marketProvider),
		Category:     NewCategoryService(repos.Category),
		Transaction:  NewTransactionService(repos.Transaction, repos.Account),
		Budget:       NewBudgetService(repos.Budget, repos.Transaction, repos.Category),
		Goal:         NewGoalService(repos.Goal),
		Portfolio:    NewPortfolioService(repos.Portfolio, repos.Holding, repos.Security, marketProvider),
		Investment:   NewInvestmentService(repos.Portfolio, repos.Holding, repos.Security, repos.Investment, marketProvider),
		Analytics:    NewAnalyticsService(repos, cfg, aiClient), // передаем весь repos так как хз какие но там много repos будут использоваться
		BrokerImport: NewBrokerImportService(repos),
	}

}
