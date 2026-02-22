package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alligatorO15/fin-tracker/internal/api/middleware"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/service"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService service.AnalyticsService
}

func NewAnalyticsHandler(analyticsService service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

func (h *AnalyticsHandler) GetSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)
	period := models.Period(c.DefaultQuery("period", "month"))

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = &t
		}
	}
	if e := c.Query("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endDate = &t
		}
	}

	summary, err := h.analyticsService.GetFinancialSummary(c.Request.Context(), userID, period, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *AnalyticsHandler) GetCashFlow(c *gin.Context) {
	userID := middleware.GetUserID(c)
	period := models.Period(c.DefaultQuery("period", "month"))

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = &t
		}
	}
	if e := c.Query("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endDate = &t
		}
	}

	report, err := h.analyticsService.GetCashFlowReport(c.Request.Context(), userID, period, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *AnalyticsHandler) GetSpendingTrends(c *gin.Context) {
	userID := middleware.GetUserID(c)

	months := 6
	if m := c.Query("months"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed > 0 {
			months = parsed
		}
	}

	trends, err := h.analyticsService.GetSpendingTrends(c.Request.Context(), userID, months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, trends)
}

func (h *AnalyticsHandler) GetNetWorth(c *gin.Context) {
	userID := middleware.GetUserID(c)

	report, err := h.analyticsService.GetNetWorthReport(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *AnalyticsHandler) GetFinancialHealth(c *gin.Context) {
	userID := middleware.GetUserID(c)

	health, err := h.analyticsService.GetFinancialHealth(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, health)
}

func (h *AnalyticsHandler) GetRecommendations(c *gin.Context) {
	userID := middleware.GetUserID(c)

	recommendations, err := h.analyticsService.GetRecommendations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recommendations)
}
