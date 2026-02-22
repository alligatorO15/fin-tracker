package handlers

import (
	"net/http"

	"github.com/alligatorO15/fin-tracker/internal/api/middleware"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PortfolioHandler struct {
	portfolioService service.PortfolioService
}

func NewPortfolioHandler(portfolioService service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{portfolioService: portfolioService}
}

func (h *PortfolioHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input models.PortfolioCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	portfolio, err := h.portfolioService.Create(c.Request.Context(), userID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, portfolio)
}

func (h *PortfolioHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	portfolios, err := h.portfolioService.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, portfolios)
}

func (h *PortfolioHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	portfolio, err := h.portfolioService.GetWithHoldings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "portfolio not found"})
		return
	}

	c.JSON(http.StatusOK, portfolio)
}

func (h *PortfolioHandler) GetHoldings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	portfolio, err := h.portfolioService.GetWithHoldings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "portfolio not found"})
		return
	}

	c.JSON(http.StatusOK, portfolio.Holdings)
}

func (h *PortfolioHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	var input models.PortfolioUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	portfolio, err := h.portfolioService.Update(c.Request.Context(), id, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, portfolio)
}

func (h *PortfolioHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	if err := h.portfolioService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "portfolio deleted"})
}

func (h *PortfolioHandler) RefreshPrices(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portfolio ID"})
		return
	}

	if err := h.portfolioService.RefreshPrices(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return updated portfolio
	portfolio, err := h.portfolioService.GetWithHoldings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, portfolio)
}
