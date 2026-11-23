package service

import (
	"context"
	"testing"
	"time"

	"github.com/InsafMin/gomoney/analytics-service/internal/models"
)

// MockRepository имитирует repository для тестов
type MockRepository struct {
	DailySummary  *models.DailySummary
	CategoryStats []*models.CategoryStat
	Error         error
}

// GetDailyReport возвращает мок данные для дневного отчета
func (m *MockRepository) GetDailyReport(ctx context.Context, userID string, date time.Time) (*models.DailySummary, []*models.CategoryStat, error) {
	if m.Error != nil {
		return nil, nil, m.Error
	}
	return m.DailySummary, m.CategoryStats, nil
}

// GetPeriodReport возвращает мок данные для отчета за период
func (m *MockRepository) GetPeriodReport(ctx context.Context, userID string, startDate, endDate time.Time) (float64, float64, error) {
	if m.Error != nil {
		return 0, 0, m.Error
	}
	return 1000.0, 500.0, nil
}

// GetCategoryStats возвращает мок данные статистики по категориям
func (m *MockRepository) GetCategoryStats(ctx context.Context, userID string, txType *string, startDate, endDate *time.Time, limit int) ([]*models.CategoryStat, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.CategoryStats, nil
}

// GetTrendData возвращает мок данные трендов
func (m *MockRepository) GetTrendData(ctx context.Context, userID, txType string, startDate, endDate time.Time, period string) ([]time.Time, []float64, error) {
	if m.Error != nil {
		return nil, nil, m.Error
	}

	now := time.Now()
	dates := []time.Time{now.AddDate(0, 0, -2), now.AddDate(0, 0, -1), now}
	values := []float64{100.0, 150.0, 200.0}

	return dates, values, nil
}

// MockCache имитирует cache для тестов
type MockCache struct {
	Data  map[string]interface{}
	Error error
}

// NewMockCache создает новый mock cache
func NewMockCache() *MockCache {
	return &MockCache{
		Data: make(map[string]interface{}),
	}
}

// Get получает значение из мок кэша
func (m *MockCache) Get(ctx context.Context, key string, dest interface{}) error {
	if m.Error != nil {
		return m.Error
	}
	// Имитация cache miss - возвращаем ошибку, чтобы сервис пошел в репозиторий
	return context.DeadlineExceeded
}

// Set сохраняет значение в мок кэш
func (m *MockCache) Set(ctx context.Context, key string, value interface{}) error {
	if m.Error != nil {
		return m.Error
	}
	m.Data[key] = value
	return nil
}

// Delete удаляет значение из мок кэша
func (m *MockCache) Delete(ctx context.Context, keys ...string) error {
	return nil
}

// InvalidateUserCache инвалидирует кэш пользователя
func (m *MockCache) InvalidateUserCache(ctx context.Context, userID string) error {
	return nil
}

// TestGetDailyReport тестирует получение дневного отчета
func TestGetDailyReport(t *testing.T) {
	mockRepo := &MockRepository{
		DailySummary: &models.DailySummary{
			UserID:  "user-123",
			Date:    time.Now(),
			Income:  1000.0,
			Expense: 500.0,
			Balance: 500.0,
		},
		CategoryStats: []*models.CategoryStat{
			{
				CategoryID:       "cat-1",
				CategoryName:     "Food",
				Type:             "expense",
				Amount:           300.0,
				TransactionCount: 5,
			},
		},
	}

	mockCache := NewMockCache()
	service := NewServiceWithInterfaces(mockRepo, mockCache)

	ctx := context.Background()
	summary, stats, err := service.GetDailyReport(ctx, "user-123", time.Now())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary, got nil")
	}

	if summary.Income != 1000.0 {
		t.Errorf("Expected income 1000.0, got %f", summary.Income)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 category stat, got %d", len(stats))
	}

	// Проверка расчета процентов
	if stats[0].Percentage == 0 {
		t.Error("Expected percentage to be calculated")
	}
}

// TestGetTrend тестирует получение данных трендов
func TestGetTrend(t *testing.T) {
	mockRepo := &MockRepository{}
	mockCache := NewMockCache()
	service := NewServiceWithInterfaces(mockRepo, mockCache)

	ctx := context.Background()
	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	trendData, err := service.GetTrend(ctx, "user-123", "expense", "daily", startDate, endDate)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if trendData == nil {
		t.Fatal("Expected trend data, got nil")
	}

	if trendData.Period != "daily" {
		t.Errorf("Expected period 'daily', got %s", trendData.Period)
	}

	if trendData.Type != "expense" {
		t.Errorf("Expected type 'expense', got %s", trendData.Type)
	}

	if len(trendData.Points) != 3 {
		t.Errorf("Expected 3 trend points, got %d", len(trendData.Points))
	}
}

// TestCalculatePercentages тестирует расчет процентов для категорий
func TestCalculatePercentages(t *testing.T) {
	service := &Service{}

	stats := []*models.CategoryStat{
		{Amount: 600.0},
		{Amount: 300.0},
		{Amount: 100.0},
	}

	service.calculatePercentages(stats)

	// Общая сумма = 1000, проверяем проценты
	expectedPercentages := []float64{60.0, 30.0, 10.0}

	for i, stat := range stats {
		if stat.Percentage != expectedPercentages[i] {
			t.Errorf("Expected percentage %f, got %f", expectedPercentages[i], stat.Percentage)
		}
	}
}

// TestCalculatePercentageChange тестирует расчет процентного изменения
func TestCalculatePercentageChange(t *testing.T) {
	service := &Service{}

	testCases := []struct {
		name     string
		old      float64
		new      float64
		expected float64
	}{
		{"Increase by 50%", 100.0, 150.0, 50.0},
		{"Decrease by 50%", 100.0, 50.0, -50.0},
		{"No change", 100.0, 100.0, 0.0},
		{"From zero", 0.0, 100.0, 100.0},
		{"To zero", 100.0, 0.0, -100.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.calculatePercentageChange(tc.old, tc.new)
			if result != tc.expected {
				t.Errorf("Expected %f, got %f", tc.expected, result)
			}
		})
	}
}
