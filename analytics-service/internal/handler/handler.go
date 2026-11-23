package handler

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/InsafMin/gomoney/analytics-service/internal/service"
	pb "github.com/InsafMin/gomoney/shared/proto"
)

// Handler обрабатывает gRPC запросы для аналитики
type Handler struct {
	pb.UnimplementedAnalyticsServiceServer
	service *service.Service
}

// NewHandler создает новый экземпляр gRPC обработчика
func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetDailyReport возвращает дневной отчет для пользователя
func (h *Handler) GetDailyReport(ctx context.Context, req *pb.DailyReportRequest) (*pb.DailyReportResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	date := time.Now()
	if req.Date != nil {
		date = req.Date.AsTime()
	}

	summary, stats, err := h.service.GetDailyReport(ctx, req.UserId, date)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get daily report: %v", err)
	}

	categoryStats := make([]*pb.CategoryStat, len(stats))
	for i, stat := range stats {
		categoryStats[i] = &pb.CategoryStat{
			CategoryId:       stat.CategoryID,
			CategoryName:     stat.CategoryName,
			Type:             stat.Type,
			Amount:           stat.Amount,
			TransactionCount: stat.TransactionCount,
			Percentage:       stat.Percentage,
		}
	}

	return &pb.DailyReportResponse{
		Date:    timestamppb.New(summary.Date),
		Income:  summary.Income,
		Expense: summary.Expense,
		Balance: summary.Balance,
		Stats:   categoryStats,
	}, nil
}

// GetPeriodReport возвращает отчет за период
func (h *Handler) GetPeriodReport(ctx context.Context, req *pb.PeriodReportRequest) (*pb.PeriodReportResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.StartDate == nil || req.EndDate == nil {
		return nil, status.Error(codes.InvalidArgument, "start_date and end_date are required")
	}

	startDate := req.StartDate.AsTime()
	endDate := req.EndDate.AsTime()

	if endDate.Before(startDate) {
		return nil, status.Error(codes.InvalidArgument, "end_date must be after start_date")
	}

	income, expense, err := h.service.GetPeriodReport(ctx, req.UserId, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get period report: %v", err)
	}

	return &pb.PeriodReportResponse{
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Income:    income,
		Expense:   expense,
		Balance:   income - expense,
	}, nil
}

// GetCategoryStats возвращает статистику по категориям
func (h *Handler) GetCategoryStats(ctx context.Context, req *pb.CategoryStatsRequest) (*pb.CategoryStatsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var txType *string
	if req.Type != "" {
		txType = &req.Type
	}

	var startDate, endDate *time.Time
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		startDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		endDate = &t
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	stats, err := h.service.GetCategoryStats(ctx, req.UserId, txType, startDate, endDate, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get category stats: %v", err)
	}

	categoryStats := make([]*pb.CategoryStat, len(stats))
	for i, stat := range stats {
		categoryStats[i] = &pb.CategoryStat{
			CategoryId:       stat.CategoryID,
			CategoryName:     stat.CategoryName,
			Type:             stat.Type,
			Amount:           stat.Amount,
			TransactionCount: stat.TransactionCount,
			Percentage:       stat.Percentage,
		}
	}

	return &pb.CategoryStatsResponse{
		Stats: categoryStats,
	}, nil
}

// GetTrend возвращает данные трендов для построения графиков
func (h *Handler) GetTrend(ctx context.Context, req *pb.TrendRequest) (*pb.TrendResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}
	if req.StartDate == nil || req.EndDate == nil {
		return nil, status.Error(codes.InvalidArgument, "start_date and end_date are required")
	}

	period := req.Period
	if period == "" {
		period = "daily"
	}

	if period != "daily" && period != "weekly" && period != "monthly" {
		return nil, status.Error(codes.InvalidArgument, "period must be daily, weekly, or monthly")
	}

	startDate := req.StartDate.AsTime()
	endDate := req.EndDate.AsTime()

	trendData, err := h.service.GetTrend(ctx, req.UserId, req.Type, period, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get trend: %v", err)
	}

	points := make([]*pb.TrendPoint, len(trendData.Points))
	for i, point := range trendData.Points {
		points[i] = &pb.TrendPoint{
			Date:   timestamppb.New(point.Date),
			Amount: point.Amount,
		}
	}

	return &pb.TrendResponse{
		Period:     trendData.Period,
		Type:       trendData.Type,
		TotalCount: int32(trendData.TotalCount),
		Points:     points,
	}, nil
}

// GetMonthlyComparison возвращает сравнение текущего и предыдущего месяца
func (h *Handler) GetMonthlyComparison(ctx context.Context, req *pb.MonthlyComparisonRequest) (*pb.MonthlyComparisonResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	month := time.Now()
	if req.Month != nil {
		month = req.Month.AsTime()
	}

	comparison, err := h.service.GetMonthlyComparison(ctx, req.UserId, month)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get monthly comparison: %v", err)
	}

	current := comparison["current"].(map[string]float64)
	previous := comparison["previous"].(map[string]float64)
	change := comparison["change"].(map[string]float64)

	return &pb.MonthlyComparisonResponse{
		CurrentIncome:   current["income"],
		CurrentExpense:  current["expense"],
		CurrentBalance:  current["balance"],
		PreviousIncome:  previous["income"],
		PreviousExpense: previous["expense"],
		PreviousBalance: previous["balance"],
		IncomeChange:    change["income"],
		ExpenseChange:   change["expense"],
	}, nil
}
