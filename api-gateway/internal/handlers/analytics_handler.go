package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/InsafMin/gomoney/api-gateway/internal/clients"
	"github.com/InsafMin/gomoney/api-gateway/internal/middleware"
	pb "github.com/InsafMin/gomoney/shared/proto"
)

type AnalyticsHandler struct {
	analyticsClient *clients.AnalyticsClient
}

func NewAnalyticsHandler(analyticsClient *clients.AnalyticsClient) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsClient: analyticsClient}
}

// GetDailyReport получает дневной отчет
func (h *AnalyticsHandler) GetDailyReport(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	dateStr := c.Query("date")
	var date time.Time
	if dateStr != "" {
		ts, err := strconv.ParseInt(dateStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
			return
		}
		date = time.Unix(ts, 0)
	} else {
		date = time.Now()
	}

	resp, err := h.analyticsClient.GetDailyReport(c.Request.Context(), &pb.DailyReportRequest{
		UserId: userID,
		Date:   timestamppb.New(date),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get daily report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"date":    resp.Date.AsTime().Unix(),
		"income":  resp.Income,
		"expense": resp.Expense,
		"balance": resp.Balance,
		"stats":   resp.Stats,
	})
}

// GetPeriodReport получает отчет за период
func (h *AnalyticsHandler) GetPeriodReport(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}

	startTs, err := strconv.ParseInt(startDateStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format"})
		return
	}

	endTs, err := strconv.ParseInt(endDateStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
		return
	}

	resp, err := h.analyticsClient.GetPeriodReport(c.Request.Context(), &pb.PeriodReportRequest{
		UserId:    userID,
		StartDate: timestamppb.New(time.Unix(startTs, 0)),
		EndDate:   timestamppb.New(time.Unix(endTs, 0)),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get period report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start_date": resp.StartDate.AsTime().Unix(),
		"end_date":   resp.EndDate.AsTime().Unix(),
		"income":     resp.Income,
		"expense":    resp.Expense,
		"balance":    resp.Balance,
	})
}

// GetCategoryStats получает статистику по категориям
func (h *AnalyticsHandler) GetCategoryStats(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	txType := c.Query("type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var startDate, endDate *timestamppb.Timestamp
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if ts, err := strconv.ParseInt(startDateStr, 10, 64); err == nil {
			startDate = timestamppb.New(time.Unix(ts, 0))
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if ts, err := strconv.ParseInt(endDateStr, 10, 64); err == nil {
			endDate = timestamppb.New(time.Unix(ts, 0))
		}
	}

	resp, err := h.analyticsClient.GetCategoryStats(c.Request.Context(), &pb.CategoryStatsRequest{
		UserId:    userID,
		Type:      txType,
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     int32(limit),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get category stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": resp.Stats,
	})
}

// GetTrend получает данные трендов
func (h *AnalyticsHandler) GetTrend(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	txType := c.DefaultQuery("type", "expense")
	period := c.DefaultQuery("period", "daily")

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}

	startTs, _ := strconv.ParseInt(startDateStr, 10, 64)
	endTs, _ := strconv.ParseInt(endDateStr, 10, 64)

	resp, err := h.analyticsClient.GetTrend(c.Request.Context(), &pb.TrendRequest{
		UserId:    userID,
		Type:      txType,
		Period:    period,
		StartDate: timestamppb.New(time.Unix(startTs, 0)),
		EndDate:   timestamppb.New(time.Unix(endTs, 0)),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trend data"})
		return
	}

	points := make([]map[string]interface{}, len(resp.Points))
	for i, p := range resp.Points {
		points[i] = map[string]interface{}{
			"date":   p.Date.AsTime().Unix(),
			"amount": p.Amount,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"period":      resp.Period,
		"type":        resp.Type,
		"total_count": resp.TotalCount,
		"points":      points,
	})
}

// GetMonthlyComparison получает сравнение месяцев
func (h *AnalyticsHandler) GetMonthlyComparison(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	monthStr := c.Query("month")
	var month time.Time
	if monthStr != "" {
		ts, err := strconv.ParseInt(monthStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format"})
			return
		}
		month = time.Unix(ts, 0)
	} else {
		month = time.Now()
	}

	resp, err := h.analyticsClient.GetMonthlyComparison(c.Request.Context(), &pb.MonthlyComparisonRequest{
		UserId: userID,
		Month:  timestamppb.New(month),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monthly comparison"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current": gin.H{
			"income":  resp.CurrentIncome,
			"expense": resp.CurrentExpense,
			"balance": resp.CurrentBalance,
		},
		"previous": gin.H{
			"income":  resp.PreviousIncome,
			"expense": resp.PreviousExpense,
			"balance": resp.PreviousBalance,
		},
		"change": gin.H{
			"income":  resp.IncomeChange,
			"expense": resp.ExpenseChange,
		},
	})
}
