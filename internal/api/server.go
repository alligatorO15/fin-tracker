package api

import (
	"github.com/alligatorO15/fin-tracker/internal/api/handlers"
	"github.com/alligatorO15/fin-tracker/internal/api/middleware"
	"github.com/alligatorO15/fin-tracker/internal/config"
	"github.com/alligatorO15/fin-tracker/internal/service"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router   *gin.Engine
	config   *config.Config
	services *service.Services
}

func NewServer(cfg *config.Config, services *service.Services) *Server {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	server := &Server{
		router:   router,
		config:   cfg,
		services: services,
	}

	server.setupRoutes()

	return server
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) setupRoutes() {
	//middleware
	s.router.Use(middleware.CORS())
	s.router.Use(middleware.RequestLogger())

	// health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := s.router.Group("/api/v1")

	// подготавливаем хэндлеры
	authHandler := handlers.NewAuthHandler(s.services.Auth, s.config)
	userHandler := handlers.NewUserHandler(s.services.User)
	accountHandler := handlers.NewAccountHandler(s.services.Account)
	categoryHandler := handlers.NewCategoryHandler(s.services.Category)
	transactionHandler := handlers.NewTransactionHandler(s.services.Transaction)
	budgetHandler := handlers.NewBudgetHandler(s.services.Budget)
	goalHandler := handlers.NewGoalHandler(s.services.Goal)
	portfolioHandler := handlers.NewPortfolioHandler(s.services.Portfolio)
	investmentHandler := handlers.NewInvestmentHandler(s.services.Investment)
	analyticsHandler := handlers.NewAnalyticsHandler(s.services.Analytics)

	// эндпоинты аутентификации (публичные)
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	// непублчиные эндпоинты
	protected := api.Group("")
	protected.Use(middleware.Auth(s.services.Auth))
	{
		// auth (protected)
		protected.POST("/auth/logout-all", authHandler.LogoutAll)

		// user
		protected.GET("/user", userHandler.GetCurrent)
		protected.PUT("/user", userHandler.Update)
		protected.DELETE("/user", userHandler.Delete)

		// accounts
		accounts := protected.Group("/accounts")
		{
			accounts.POST("", accountHandler.Create)
			accounts.GET("", accountHandler.List)
			accounts.GET("/summary", accountHandler.GetSummary)
			accounts.GET("/:id", accountHandler.GetByID)
			accounts.PUT("/:id", accountHandler.Update)
			accounts.DELETE("/:id", accountHandler.Delete)
		}

		// categories
		categories := protected.Group("/categories")
		{
			categories.POST("", categoryHandler.Create)
			categories.GET("", categoryHandler.List)
			categories.GET("/:id", categoryHandler.GetByID)
			categories.PUT("/:id", categoryHandler.Update)
			categories.DELETE("/:id", categoryHandler.Delete)
		}

		// transactions
		transactions := protected.Group("/transactions")
		{
			transactions.POST("", transactionHandler.Create)
			transactions.GET("", transactionHandler.List)
			transactions.GET("/:id", transactionHandler.GetByID)
			transactions.PUT("/:id", transactionHandler.Update)
			transactions.DELETE("/:id", transactionHandler.Delete)
		}

		// budgets
		budgets := protected.Group("/budgets")
		{
			budgets.POST("", budgetHandler.Create)
			budgets.GET("", budgetHandler.List)
			budgets.GET("/summary", budgetHandler.GetSummary)
			budgets.GET("/alerts", budgetHandler.GetAlerts)
			budgets.GET("/:id", budgetHandler.GetByID)
			budgets.PUT("/:id", budgetHandler.Update)
			budgets.DELETE("/:id", budgetHandler.Delete)
		}

		// goals
		goals := protected.Group("/goals")
		{
			goals.POST("", goalHandler.Create)
			goals.GET("", goalHandler.List)
			goals.GET("/:id", goalHandler.GetByID)
			goals.PUT("/:id", goalHandler.Update)
			goals.DELETE("/:id", goalHandler.Delete)
			goals.POST("/:id/contributions", goalHandler.AddContribution)
			goals.GET("/:id/contributions", goalHandler.GetContributions)
		}

		// investment portfolios
		portfolios := protected.Group("/portfolios")
		{
			portfolios.POST("", portfolioHandler.Create)
			portfolios.GET("", portfolioHandler.List)
			portfolios.GET("/:id", portfolioHandler.GetByID)
			portfolios.GET("/:id/holdings", portfolioHandler.GetHoldings)
			portfolios.PUT("/:id", portfolioHandler.Update)
			portfolios.DELETE("/:id", portfolioHandler.Delete)
			portfolios.POST("/:id/refresh", portfolioHandler.RefreshPrices)
		}

		// investment operations
		investments := protected.Group("/investments")
		{
			investments.GET("/securities/search", investmentHandler.SearchSecurities)
			investments.GET("/securities/:id", investmentHandler.GetSecurity)
			investments.GET("/securities/:ticker/quote", investmentHandler.GetQuote)
			investments.POST("/transactions", investmentHandler.AddTransaction)
			investments.GET("/portfolios/:id/transactions", investmentHandler.GetTransactions)
			investments.DELETE("/transactions/:id", investmentHandler.DeleteTransaction)
			investments.GET("/portfolios/:id/analytics", investmentHandler.GetAnalytics)
			investments.GET("/portfolios/:id/tax-report", investmentHandler.GetTaxReport)
			investments.GET("/portfolios/:id/dividends", investmentHandler.GetDividends)
		}

		// analytics
		analytics := protected.Group("/analytics")
		{
			analytics.GET("/summary", analyticsHandler.GetSummary)
			analytics.GET("/cashflow", analyticsHandler.GetCashFlow)
			analytics.GET("/trends", analyticsHandler.GetSpendingTrends)
			analytics.GET("/networth", analyticsHandler.GetNetWorth)
			analytics.GET("/health", analyticsHandler.GetFinancialHealth)
			analytics.GET("/recommendations", analyticsHandler.GetRecommendations)
		}

	}
}
