package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/InsafMin/gomoney/api-gateway/internal/clients"
	"github.com/InsafMin/gomoney/api-gateway/internal/middleware"
	pb "github.com/InsafMin/gomoney/shared/proto"
)

type TransactionHandler struct {
	transactionClient *clients.TransactionClient
}

func NewTransactionHandler(transactionClient *clients.TransactionClient) *TransactionHandler {
	return &TransactionHandler{transactionClient: transactionClient}
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Amount      float64  `json:"amount" binding:"required,gt=0"`
		Currency    string   `json:"currency" binding:"required"`
		CategoryID  string   `json:"category_id"`
		Description string   `json:"description"`
		Date        int64    `json:"date"`
		Type        string   `json:"type" binding:"required,oneof=income expense"`
		Tags        []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.transactionClient.CreateTransaction(c.Request.Context(), &pb.CreateTransactionRequest{
		UserId:      userID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		CategoryId:  req.CategoryID,
		Description: req.Description,
		Date:        req.Date,
		Type:        req.Type,
		Tags:        req.Tags,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"transaction_id": resp.TransactionId,
		"transaction":    resp.Transaction,
	})
}

func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	limit32 := int32(limit)
	offset32 := int32(offset)

	req := &pb.GetTransactionsRequest{
		UserId: userID,
		Limit:  &limit32,
		Offset: &offset32,
	}

	if startDate := c.Query("start_date"); startDate != "" {
		if ts, err := strconv.ParseInt(startDate, 10, 64); err == nil {
			req.StartDate = &ts
		}
	}

	if endDate := c.Query("end_date"); endDate != "" {
		if ts, err := strconv.ParseInt(endDate, 10, 64); err == nil {
			req.EndDate = &ts
		}
	}

	if categoryID := c.Query("category_id"); categoryID != "" {
		req.CategoryId = &categoryID
	}

	if txType := c.Query("type"); txType != "" {
		req.Type = &txType
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		req.SortBy = &sortBy
	}

	sortDesc := c.DefaultQuery("sort_desc", "true") == "true"
	req.SortDesc = &sortDesc

	resp, err := h.transactionClient.GetTransactions(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions":  resp.Transactions,
		"total":         resp.Total,
		"total_income":  resp.TotalIncome,
		"total_expense": resp.TotalExpense,
		"page":          resp.Page,
		"page_size":     resp.PageSize,
	})
}

func (h *TransactionHandler) GetBalance(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	currency := c.DefaultQuery("currency", "RUB")

	resp, err := h.transactionClient.GetBalance(c.Request.Context(), &pb.GetBalanceRequest{
		UserId:   userID,
		Currency: &currency,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance":       resp.Balance,
		"currency":      resp.Currency,
		"total_income":  resp.TotalIncome,
		"total_expense": resp.TotalExpense,
		"period_start":  resp.PeriodStart,
		"period_end":    resp.PeriodEnd,
	})
}

func (h *TransactionHandler) GetCategories(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	req := &pb.GetCategoriesRequest{
		UserId: userID,
	}

	if txType := c.Query("type"); txType != "" {
		req.Type = &txType
	}

	resp, err := h.transactionClient.GetCategories(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": resp.Categories,
	})
}

func (h *TransactionHandler) CreateCategory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Name  string `json:"name" binding:"required"`
		Type  string `json:"type" binding:"required,oneof=income expense"`
		Color string `json:"color"`
		Icon  string `json:"icon"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.transactionClient.CreateCategory(c.Request.Context(), &pb.CreateCategoryRequest{
		UserId: userID,
		Name:   req.Name,
		Type:   req.Type,
		Color:  &req.Color,
		Icon:   &req.Icon,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"category_id": resp.CategoryId,
		"category":    resp.Category,
	})
}
