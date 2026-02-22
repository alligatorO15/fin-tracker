package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InvestmentHandler struct {
	investmentService service.InvestmentService
}

func NewInvestmentHandler(investmentService service.InvestmentService) *InvestmentHandler {
	return &InvestmentHandler{investmentService: investmentService}
}

func (h *InvestmentHandler) SearchSecurities(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query required"})
		return
	}

	var securityType *models.SecurityType
	if t := c.Query("type"); t != "" {
		st := models.SecurityType(t)
		securityType = &st
	}

	var exchange *models.Exchange
	if e := c.Query("exchange"); e != "" {
		ex := models.Exchange(e)
		exchange = &ex
	}

	securities, err := h.investmentService.SearchSecurities(c.Request.Context(), query, securityType, exchange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, securities)
}

func (h *InvestmentHandler) GetSecurity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid security ID"})
		return
	}

	security, err := h.investmentService.GetSecurityByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "security not found"})
		return
	}

	c.JSON(http.StatusOK, security)
}

func (h *InvestmentHandler) GetQuote(c *gin.Context) {
	ticker := c.Param("ticker")
	exchangeStr := c.DefaultQuery("exchange", "MOEX")
	exchange := models.Exchange(exchangeStr)

	quote, err := h.investmentService.GetSecurityQuote(c.Request.Context(), ticker, exchange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quote)
}

func (h *InvestmentHandler) AddTransaction(c *gin.Context) {
	var input models.InvestmentTransactionCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction, err := h.investmentService.AddTransaction(c.Request.Context(), &input)
	if err != nil {
		if err == service.ErrSecurityNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err == service.ErrInsufficientShares {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

func (h *InvestmentHandler) GetTransactions(c *gin.Context) {
	portfolioID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	limit := 100
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	transactions, err := h.investmentService.GetTransactions(c.Request.Context(), portfolioID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

func (h *InvestmentHandler) DeleteTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction ID"})
		return
	}

	if err := h.investmentService.DeleteTransaction(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transaction deleted"})
}

func (h *InvestmentHandler) GetAnalytics(c *gin.Context) {
	portfolioID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	analytics, err := h.investmentService.GetPortfolioAnalytics(c.Request.Context(), portfolioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *InvestmentHandler) GetTaxReport(c *gin.Context) {
	portfolioID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	year := time.Now().Year()
	if y := c.Query("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}

	report, err := h.investmentService.GetTaxReport(c.Request.Context(), portfolioID, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *InvestmentHandler) GetDividends(c *gin.Context) {
	portfolioID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	dividends, err := h.investmentService.GetUpcomingDividends(c.Request.Context(), portfolioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dividends)
}
