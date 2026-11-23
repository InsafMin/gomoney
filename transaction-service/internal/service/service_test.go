package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/InsafMin/gomoney/transaction-service/internal/models"
)

// Mock Repository
type mockRepository struct {
	transactions map[string]*models.Transaction
	categories   map[string]*models.Category
	nextTxID     int
	nextCatID    int
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		transactions: make(map[string]*models.Transaction),
		categories:   make(map[string]*models.Category),
		nextTxID:     1,
		nextCatID:    1,
	}
}

func (m *mockRepository) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	tx.ID = fmt.Sprintf("mock-tx-id-%d", m.nextTxID)
	m.nextTxID++
	tx.CreatedAt = time.Now()
	tx.UpdatedAt = time.Now()
	m.transactions[tx.ID] = tx
	return nil
}

func (m *mockRepository) GetTransaction(ctx context.Context, id, userID string) (*models.Transaction, error) {
	tx, exists := m.transactions[id]
	if !exists || tx.UserID != userID {
		return nil, errors.New("transaction not found")
	}
	return tx, nil
}

func (m *mockRepository) GetTransactions(ctx context.Context, filter *models.TransactionFilter) (*models.TransactionSummary, error) {
	var txs []*models.Transaction
	var totalIncome, totalExpense float64

	for _, tx := range m.transactions {
		if tx.UserID == filter.UserID {
			txs = append(txs, tx)
			if tx.Type == models.TransactionTypeIncome {
				totalIncome += tx.Amount
			} else {
				totalExpense += tx.Amount
			}
		}
	}

	return &models.TransactionSummary{
		Transactions: txs,
		Total:        len(txs),
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
		Page:         1,
		PageSize:     len(txs),
	}, nil
}

func (m *mockRepository) UpdateTransaction(ctx context.Context, tx *models.Transaction) error {
	existing, exists := m.transactions[tx.ID]
	if !exists || existing.UserID != tx.UserID {
		return errors.New("transaction not found")
	}
	m.transactions[tx.ID] = tx
	return nil
}

func (m *mockRepository) DeleteTransaction(ctx context.Context, id, userID string) error {
	tx, exists := m.transactions[id]
	if !exists || tx.UserID != userID {
		return errors.New("transaction not found")
	}
	delete(m.transactions, id)
	return nil
}

func (m *mockRepository) GetBalance(ctx context.Context, userID, currency string) (*models.Balance, error) {
	var totalIncome, totalExpense float64

	for _, tx := range m.transactions {
		if tx.UserID == userID && tx.Currency == currency {
			if tx.Type == models.TransactionTypeIncome {
				totalIncome += tx.Amount
			} else {
				totalExpense += tx.Amount
			}
		}
	}

	return &models.Balance{
		Balance:      totalIncome - totalExpense,
		Currency:     currency,
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
	}, nil
}

func (m *mockRepository) CreateCategory(ctx context.Context, cat *models.Category) error {
	cat.ID = fmt.Sprintf("mock-cat-id-%d", m.nextCatID)
	m.nextCatID++
	cat.CreatedAt = time.Now()
	cat.UpdatedAt = time.Now()
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockRepository) GetCategory(ctx context.Context, id, userID string) (*models.Category, error) {
	cat, exists := m.categories[id]
	if !exists {
		return nil, errors.New("category not found")
	}
	return cat, nil
}

func (m *mockRepository) GetCategories(ctx context.Context, userID string, txType *models.TransactionType) ([]*models.Category, error) {
	var cats []*models.Category
	for _, cat := range m.categories {
		if cat.UserID == userID {
			if txType == nil || cat.Type == *txType {
				cats = append(cats, cat)
			}
		}
	}
	return cats, nil
}

func (m *mockRepository) UpdateCategory(ctx context.Context, cat *models.Category) error {
	existing, exists := m.categories[cat.ID]
	if !exists || existing.UserID != cat.UserID {
		return errors.New("category not found")
	}
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockRepository) DeleteCategory(ctx context.Context, id, userID string) error {
	cat, exists := m.categories[id]
	if !exists || cat.UserID != userID {
		return errors.New("category not found")
	}
	delete(m.categories, id)
	return nil
}

// Tests

func TestTransactionService_CreateTransaction(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, []string{"RUB", "USD", "EUR"})

	tests := []struct {
		name      string
		tx        *models.Transaction
		wantError bool
	}{
		{
			name: "valid income transaction",
			tx: &models.Transaction{
				UserID:   "user-1",
				Amount:   1000.00,
				Currency: "RUB",
				Type:     models.TransactionTypeIncome,
				Date:     time.Now(),
			},
			wantError: false,
		},
		{
			name: "valid expense transaction",
			tx: &models.Transaction{
				UserID:      "user-1",
				Amount:      500.00,
				Currency:    "USD",
				Type:        models.TransactionTypeExpense,
				Description: "Test expense",
				Date:        time.Now(),
			},
			wantError: false,
		},
		{
			name: "invalid amount",
			tx: &models.Transaction{
				UserID:   "user-1",
				Amount:   -100.00,
				Currency: "RUB",
				Type:     models.TransactionTypeIncome,
			},
			wantError: true,
		},
		{
			name: "invalid currency",
			tx: &models.Transaction{
				UserID:   "user-1",
				Amount:   100.00,
				Currency: "GBP",
				Type:     models.TransactionTypeIncome,
			},
			wantError: true,
		},
		{
			name: "missing user_id",
			tx: &models.Transaction{
				Amount:   100.00,
				Currency: "RUB",
				Type:     models.TransactionTypeIncome,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := svc.CreateTransaction(ctx, tt.tx)

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantError && tt.tx.ID == "" {
				t.Error("transaction ID should not be empty")
			}
		})
	}
}

func TestTransactionService_GetBalance(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, []string{"RUB", "USD", "EUR"})

	ctx := context.Background()

	// Создаем несколько транзакций
	_ = svc.CreateTransaction(ctx, &models.Transaction{
		UserID:   "user-1",
		Amount:   1000.00,
		Currency: "RUB",
		Type:     models.TransactionTypeIncome,
		Date:     time.Now(),
	})

	_ = svc.CreateTransaction(ctx, &models.Transaction{
		UserID:   "user-1",
		Amount:   300.00,
		Currency: "RUB",
		Type:     models.TransactionTypeExpense,
		Date:     time.Now(),
	})

	_ = svc.CreateTransaction(ctx, &models.Transaction{
		UserID:   "user-1",
		Amount:   200.00,
		Currency: "RUB",
		Type:     models.TransactionTypeExpense,
		Date:     time.Now(),
	})

	// Проверяем баланс
	balance, err := svc.GetBalance(ctx, "user-1", "RUB")
	if err != nil {
		t.Fatalf("failed to get balance: %v", err)
	}

	expectedBalance := 500.00 // 1000 - 300 - 200
	if balance.Balance != expectedBalance {
		t.Errorf("expected balance %.2f, got %.2f", expectedBalance, balance.Balance)
	}

	if balance.TotalIncome != 1000.00 {
		t.Errorf("expected total income 1000.00, got %.2f", balance.TotalIncome)
	}

	if balance.TotalExpense != 500.00 {
		t.Errorf("expected total expense 500.00, got %.2f", balance.TotalExpense)
	}
}

func TestTransactionService_CreateCategory(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, []string{"RUB"})

	tests := []struct {
		name      string
		cat       *models.Category
		wantError bool
	}{
		{
			name: "valid category",
			cat: &models.Category{
				UserID: "user-1",
				Name:   "Food",
				Type:   models.TransactionTypeExpense,
				Color:  "#FF5722",
				Icon:   "🍔",
			},
			wantError: false,
		},
		{
			name: "missing name",
			cat: &models.Category{
				UserID: "user-1",
				Type:   models.TransactionTypeExpense,
			},
			wantError: true,
		},
		{
			name: "invalid type",
			cat: &models.Category{
				UserID: "user-1",
				Name:   "Test",
				Type:   "invalid",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := svc.CreateCategory(ctx, tt.cat)

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTransactionService_GetTransactions(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, []string{"RUB"})

	ctx := context.Background()

	// Создаем транзакции
	for i := 0; i < 5; i++ {
		_ = svc.CreateTransaction(ctx, &models.Transaction{
			UserID:   "user-1",
			Amount:   float64(100 * (i + 1)),
			Currency: "RUB",
			Type:     models.TransactionTypeExpense,
			Date:     time.Now(),
		})
	}

	// Получаем транзакции
	filter := &models.TransactionFilter{
		UserID: "user-1",
		Limit:  10,
		Offset: 0,
	}

	summary, err := svc.GetTransactions(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get transactions: %v", err)
	}

	if summary.Total != 5 {
		t.Errorf("expected 5 transactions, got %d", summary.Total)
	}

	if len(summary.Transactions) != 5 {
		t.Errorf("expected 5 transactions in list, got %d", len(summary.Transactions))
	}
}
