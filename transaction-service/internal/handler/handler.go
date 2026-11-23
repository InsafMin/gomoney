package handler

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/InsafMin/gomoney/shared/proto"
	"github.com/InsafMin/gomoney/transaction-service/internal/models"
	"github.com/InsafMin/gomoney/transaction-service/internal/service"
)

type Handler struct {
	pb.UnimplementedTransactionServiceServer
	service service.Service
}

func NewHandler(service service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	log.Printf("CreateTransaction: user=%s, amount=%.2f %s", req.UserId, req.Amount, req.Currency)

	tx := &models.Transaction{
		UserID:      req.UserId,
		Amount:      req.Amount,
		Currency:    req.Currency,
		CategoryID:  req.CategoryId,
		Description: req.Description,
		Date:        time.Unix(req.Date, 0),
		Type:        models.TransactionType(req.Type),
		Tags:        req.Tags,
	}

	log.Printf("Calling service.CreateTransaction")
	if err := h.service.CreateTransaction(ctx, tx); err != nil {
		log.Printf("CreateTransaction error: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
	}

	log.Printf("Transaction created successfully: %s", tx.ID)
	return &pb.CreateTransactionResponse{
		TransactionId: tx.ID,
		Transaction:   h.toProtoTransaction(tx),
	}, nil
}

func (h *Handler) GetTransactions(ctx context.Context, req *pb.GetTransactionsRequest) (*pb.GetTransactionsResponse, error) {
	filter := &models.TransactionFilter{
		UserID:   req.UserId,
		Limit:    int(req.GetLimit()),
		Offset:   int(req.GetOffset()),
		SortBy:   req.GetSortBy(),
		SortDesc: req.GetSortDesc(),
	}

	if req.StartDate != nil {
		t := time.Unix(*req.StartDate, 0)
		filter.StartDate = &t
	}
	if req.EndDate != nil {
		t := time.Unix(*req.EndDate, 0)
		filter.EndDate = &t
	}
	if req.CategoryId != nil {
		filter.CategoryID = req.CategoryId
	}
	if req.Type != nil {
		txType := models.TransactionType(*req.Type)
		filter.Type = &txType
	}

	summary, err := h.service.GetTransactions(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get transactions: %v", err)
	}

	var protoTxs []*pb.Transaction
	for _, tx := range summary.Transactions {
		protoTxs = append(protoTxs, h.toProtoTransaction(tx))
	}

	return &pb.GetTransactionsResponse{
		Transactions: protoTxs,
		Total:        int32(summary.Total),
		TotalIncome:  summary.TotalIncome,
		TotalExpense: summary.TotalExpense,
		Page:         int32(summary.Page),
		PageSize:     int32(summary.PageSize),
	}, nil
}

func (h *Handler) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	tx, err := h.service.GetTransaction(ctx, req.TransactionId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "transaction not found")
	}

	return &pb.GetTransactionResponse{
		Transaction: h.toProtoTransaction(tx),
	}, nil
}

func (h *Handler) UpdateTransaction(ctx context.Context, req *pb.UpdateTransactionRequest) (*pb.UpdateTransactionResponse, error) {
	tx := &models.Transaction{
		ID:     req.TransactionId,
		UserID: req.UserId,
	}

	if req.Amount != nil {
		tx.Amount = *req.Amount
	}
	if req.CategoryId != nil {
		tx.CategoryID = *req.CategoryId
	}
	if req.Description != nil {
		tx.Description = *req.Description
	}
	if req.Date != nil {
		tx.Date = time.Unix(*req.Date, 0)
	}
	if len(req.Tags) > 0 {
		tx.Tags = req.Tags
	}

	if err := h.service.UpdateTransaction(ctx, tx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update transaction: %v", err)
	}

	updated, _ := h.service.GetTransaction(ctx, tx.ID, tx.UserID)
	return &pb.UpdateTransactionResponse{
		Transaction: h.toProtoTransaction(updated),
	}, nil
}

func (h *Handler) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteTransactionResponse, error) {
	if err := h.service.DeleteTransaction(ctx, req.TransactionId, req.UserId); err != nil {
		return &pb.DeleteTransactionResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.DeleteTransactionResponse{Success: true, Message: "Transaction deleted successfully"}, nil
}

func (h *Handler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	currency := req.GetCurrency()
	if currency == "" {
		currency = "RUB"
	}

	balance, err := h.service.GetBalance(ctx, req.UserId, currency)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get balance: %v", err)
	}

	return &pb.GetBalanceResponse{
		Balance:      balance.Balance,
		Currency:     balance.Currency,
		TotalIncome:  balance.TotalIncome,
		TotalExpense: balance.TotalExpense,
		PeriodStart:  balance.PeriodStart.Unix(),
		PeriodEnd:    balance.PeriodEnd.Unix(),
	}, nil
}

func (h *Handler) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	cat := &models.Category{
		UserID: req.UserId,
		Name:   req.Name,
		Type:   models.TransactionType(req.Type),
		Color:  req.GetColor(),
		Icon:   req.GetIcon(),
	}

	if req.ParentId != nil {
		cat.ParentID = req.ParentId
	}

	if err := h.service.CreateCategory(ctx, cat); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create category: %v", err)
	}

	return &pb.CreateCategoryResponse{
		CategoryId: cat.ID,
		Category:   h.toProtoCategory(cat),
	}, nil
}

func (h *Handler) GetCategories(ctx context.Context, req *pb.GetCategoriesRequest) (*pb.GetCategoriesResponse, error) {
	var txType *models.TransactionType
	if req.Type != nil {
		t := models.TransactionType(*req.Type)
		txType = &t
	}

	cats, err := h.service.GetCategories(ctx, req.UserId, txType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get categories: %v", err)
	}

	var protoCats []*pb.Category
	for _, cat := range cats {
		protoCats = append(protoCats, h.toProtoCategory(cat))
	}

	return &pb.GetCategoriesResponse{Categories: protoCats}, nil
}

func (h *Handler) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.UpdateCategoryResponse, error) {
	cat := &models.Category{
		ID:     req.CategoryId,
		UserID: req.UserId,
	}

	if req.Name != nil {
		cat.Name = *req.Name
	}
	if req.Color != nil {
		cat.Color = *req.Color
	}
	if req.Icon != nil {
		cat.Icon = *req.Icon
	}

	if err := h.service.UpdateCategory(ctx, cat); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update category: %v", err)
	}

	return &pb.UpdateCategoryResponse{Category: h.toProtoCategory(cat)}, nil
}

func (h *Handler) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*pb.DeleteCategoryResponse, error) {
	if err := h.service.DeleteCategory(ctx, req.CategoryId, req.UserId); err != nil {
		return &pb.DeleteCategoryResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.DeleteCategoryResponse{Success: true, Message: "Category deleted successfully"}, nil
}

func (h *Handler) toProtoTransaction(tx *models.Transaction) *pb.Transaction {
	proto := &pb.Transaction{
		Id:          tx.ID,
		UserId:      tx.UserID,
		Amount:      tx.Amount,
		Currency:    tx.Currency,
		CategoryId:  tx.CategoryID,
		Description: tx.Description,
		Date:        tx.Date.Unix(),
		Type:        string(tx.Type),
		Tags:        tx.Tags,
		CreatedAt:   tx.CreatedAt.Unix(),
		UpdatedAt:   tx.UpdatedAt.Unix(),
	}

	if tx.Category != nil {
		proto.Category = h.toProtoCategory(tx.Category)
	}

	return proto
}

func (h *Handler) toProtoCategory(cat *models.Category) *pb.Category {
	proto := &pb.Category{
		Id:        cat.ID,
		UserId:    cat.UserID,
		Name:      cat.Name,
		Type:      string(cat.Type),
		Color:     cat.Color,
		Icon:      cat.Icon,
		CreatedAt: cat.CreatedAt.Unix(),
		UpdatedAt: cat.UpdatedAt.Unix(),
	}

	if cat.ParentID != nil {
		proto.ParentId = cat.ParentID
	}

	return proto
}
