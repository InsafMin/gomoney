package service

import (
	"context"
	"fmt"
	"time"

	"github.com/InsafMin/gomoney/analytics-service/internal/cache"
	"github.com/InsafMin/gomoney/analytics-service/internal/models"
	"github.com/InsafMin/gomoney/analytics-service/internal/repository"
)

// RepositoryInterface определяет методы работы с БД
type RepositoryInterface interface {
	GetDailyReport(ctx context.Context, userID string, date time.Time) (*models.DailySummary, []*models.CategoryStat, error)
	GetPeriodReport(ctx context.Context, userID string, startDate, endDate time.Time) (float64, float64, error)
	GetCategoryStats(ctx context.Context, userID string, txType *string, startDate, endDate *time.Time, limit int) ([]*models.CategoryStat, error)
	GetTrendData(ctx context.Context, userID, txType string, startDate, endDate time.Time, period string) ([]time.Time, []float64, error)
}

// CacheInterface определяет методы работы с кэшем
type CacheInterface interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}) error
}

// Service предоставляет бизнес-логику для аналитики
type Service struct {
	repo  RepositoryInterface
	cache CacheInterface
}

// NewService создает новый экземпляр сервиса аналитики
func NewService(repo *repository.Repository, cache *cache.Cache) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
	}
}

// NewServiceWithInterfaces создает сервис с интерфейсами (для тестов)
func NewServiceWithInterfaces(repo RepositoryInterface, cache CacheInterface) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
	}
}

// GetDailyReport возвращает отчет за день с кэшированием
func (s *Service) GetDailyReport(ctx context.Context, userID string, date time.Time) (*models.DailySummary, []*models.CategoryStat, error) {
	cacheKey := cache.MakeDailyReportKey(userID, date)

	type cachedReport struct {
		Summary *models.DailySummary   `json:"summary"`
		Stats   []*models.CategoryStat `json:"stats"`
	}

	var cached cachedReport
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return cached.Summary, cached.Stats, nil
	}

	summary, stats, err := s.repo.GetDailyReport(ctx, userID, date)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get daily report: %w", err)
	}

	s.calculatePercentages(stats)

	cached = cachedReport{Summary: summary, Stats: stats}
	if err := s.cache.Set(ctx, cacheKey, cached); err != nil {
		// Ошибка кэширования не критична
	}

	return summary, stats, nil
}

// GetPeriodReport возвращает отчет за период
func (s *Service) GetPeriodReport(ctx context.Context, userID string, startDate, endDate time.Time) (float64, float64, error) {
	income, expense, err := s.repo.GetPeriodReport(ctx, userID, startDate, endDate)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get period report: %w", err)
	}
	return income, expense, nil
}

// GetCategoryStats возвращает статистику по категориям с кэшированием
func (s *Service) GetCategoryStats(ctx context.Context, userID string, txType *string, startDate, endDate *time.Time, limit int) ([]*models.CategoryStat, error) {
	// Для кэширования нужны конкретные даты
	var cacheKey string
	useCache := txType != nil && startDate != nil && endDate != nil

	if useCache {
		cacheKey = cache.MakeCategoryStatsKey(userID, *txType, *startDate, *endDate)
		var cached []*models.CategoryStat
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	stats, err := s.repo.GetCategoryStats(ctx, userID, txType, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get category stats: %w", err)
	}

	s.calculatePercentages(stats)

	if useCache && len(stats) > 0 {
		if err := s.cache.Set(ctx, cacheKey, stats); err != nil {
			// Ошибка кэширования не критична
		}
	}

	return stats, nil
}

// GetTrend возвращает данные трендов по периодам (daily, weekly, monthly)
func (s *Service) GetTrend(ctx context.Context, userID, txType, period string, startDate, endDate time.Time) (*models.TrendData, error) {
	cacheKey := cache.MakeTrendKey(userID, txType, period, startDate, endDate)

	var cached models.TrendData
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	dates, values, err := s.repo.GetTrendData(ctx, userID, txType, startDate, endDate, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}

	points := make([]models.TrendPoint, len(dates))
	for i := range dates {
		points[i] = models.TrendPoint{
			Date:   dates[i],
			Amount: values[i],
		}
	}

	trendData := &models.TrendData{
		Period:     period,
		Type:       txType,
		TotalCount: len(points),
		Points:     points,
	}

	if len(points) > 0 {
		if err := s.cache.Set(ctx, cacheKey, trendData); err != nil {
			// Ошибка кэширования не критична
		}
	}

	return trendData, nil
}

// GetMonthlyComparison возвращает сравнение текущего и предыдущего месяца
func (s *Service) GetMonthlyComparison(ctx context.Context, userID string, month time.Time) (map[string]interface{}, error) {
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	startOfPrevMonth := startOfMonth.AddDate(0, -1, 0)
	endOfPrevMonth := startOfMonth

	currentIncome, currentExpense, err := s.repo.GetPeriodReport(ctx, userID, startOfMonth, endOfMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to get current month report: %w", err)
	}

	prevIncome, prevExpense, err := s.repo.GetPeriodReport(ctx, userID, startOfPrevMonth, endOfPrevMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous month report: %w", err)
	}

	incomeChange := s.calculatePercentageChange(prevIncome, currentIncome)
	expenseChange := s.calculatePercentageChange(prevExpense, currentExpense)

	return map[string]interface{}{
		"current": map[string]float64{
			"income":  currentIncome,
			"expense": currentExpense,
			"balance": currentIncome - currentExpense,
		},
		"previous": map[string]float64{
			"income":  prevIncome,
			"expense": prevExpense,
			"balance": prevIncome - prevExpense,
		},
		"change": map[string]float64{
			"income":  incomeChange,
			"expense": expenseChange,
		},
	}, nil
}

// calculatePercentages вычисляет процентное соотношение для статистики категорий
func (s *Service) calculatePercentages(stats []*models.CategoryStat) {
	if len(stats) == 0 {
		return
	}

	var total float64
	for _, stat := range stats {
		total += stat.Amount
	}

	if total > 0 {
		for _, stat := range stats {
			stat.Percentage = (stat.Amount / total) * 100
		}
	}
}

// calculatePercentageChange вычисляет процентное изменение
func (s *Service) calculatePercentageChange(old, new float64) float64 {
	if old == 0 {
		if new == 0 {
			return 0
		}
		return 100
	}
	return ((new - old) / old) * 100
}
