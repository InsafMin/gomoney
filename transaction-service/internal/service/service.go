package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/InsafMin/gomoney/transaction-service/internal/kafka"
	"github.com/InsafMin/gomoney/transaction-service/internal/models"
	"github.com/InsafMin/gomoney/transaction-service/internal/repository"
)

type Service interface {
	// Transactions
	CreateTransaction(ctx context.Context, tx *models.Transaction) error
	GetTransaction(ctx context.Context, id, userID string) (*models.Transaction, error)
	GetTransactions(ctx context.Context, filter *models.TransactionFilter) (*models.TransactionSummary, error)
	UpdateTransaction(ctx context.Context, tx *models.Transaction) error
	DeleteTransaction(ctx context.Context, id, userID string) error
	GetBalance(ctx context.Context, userID, currency string) (*models.Balance, error)

	// Categories
	CreateCategory(ctx context.Context, cat *models.Category) error
	GetCategories(ctx context.Context, userID string, txType *models.TransactionType) ([]*models.Category, error)
	UpdateCategory(ctx context.Context, cat *models.Category) error
	DeleteCategory(ctx context.Context, id, userID string) error
}

type TransactionService struct {
	repo              repository.Repository
	kafkaProducer     *kafka.Producer
	allowedCurrencies map[string]bool
}

func NewService(repo repository.Repository, kafkaProducer *kafka.Producer, allowedCurrencies []string) *TransactionService {
	currencies := make(map[string]bool)
	for _, c := range allowedCurrencies {
		currencies[c] = true
	}

	return &TransactionService{
		repo:              repo,
		kafkaProducer:     kafkaProducer,
		allowedCurrencies: currencies,
	}
}

// CreateTransaction создает новую транзакцию
func (s *TransactionService) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	// Валидация
	if err := s.validateTransaction(tx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Проверка категории
	if tx.CategoryID != "" {
		_, err := s.repo.GetCategory(ctx, tx.CategoryID, tx.UserID)
		if err != nil {
			return fmt.Errorf("category not found: %w", err)
		}
	}

	// Создание транзакции
	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Публикация события в Kafka
	if s.kafkaProducer != nil {
		if err := s.kafkaProducer.PublishTransactionCreated(ctx, tx); err != nil {
			log.Printf("Warning: failed to publish transaction created event: %v", err)
		}
	}

	log.Printf("Transaction created: %s for user %s, amount: %.2f %s", tx.ID, tx.UserID, tx.Amount, tx.Currency)
	return nil
}

// GetTransaction получает транзакцию по ID
func (s *TransactionService) GetTransaction(ctx context.Context, id, userID string) (*models.Transaction, error) {
	tx, err := s.repo.GetTransaction(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return tx, nil
}

// GetTransactions получает список транзакций с фильтрацией
func (s *TransactionService) GetTransactions(ctx context.Context, filter *models.TransactionFilter) (*models.TransactionSummary, error) {
	summary, err := s.repo.GetTransactions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	return summary, nil
}

// UpdateTransaction обновляет транзакцию
func (s *TransactionService) UpdateTransaction(ctx context.Context, tx *models.Transaction) error {
	// Проверка существования
	existing, err := s.repo.GetTransaction(ctx, tx.ID, tx.UserID)
	if err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	// Валидация изменений
	if tx.Amount > 0 {
		existing.Amount = tx.Amount
	}
	if tx.CategoryID != "" {
		_, err := s.repo.GetCategory(ctx, tx.CategoryID, tx.UserID)
		if err != nil {
			return fmt.Errorf("category not found: %w", err)
		}
		existing.CategoryID = tx.CategoryID
	}
	if tx.Description != "" {
		existing.Description = tx.Description
	}
	if !tx.Date.IsZero() {
		existing.Date = tx.Date
	}
	if len(tx.Tags) > 0 {
		existing.Tags = tx.Tags
	}

	// Обновление
	if err := s.repo.UpdateTransaction(ctx, existing); err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	// Публикация события
	if s.kafkaProducer != nil {
		if err := s.kafkaProducer.PublishTransactionUpdated(ctx, existing); err != nil {
			log.Printf("Warning: failed to publish transaction updated event: %v", err)
		}
	}

	log.Printf("Transaction updated: %s", tx.ID)
	return nil
}

// DeleteTransaction удаляет транзакцию
func (s *TransactionService) DeleteTransaction(ctx context.Context, id, userID string) error {
	// Получаем транзакцию перед удалением для события
	tx, err := s.repo.GetTransaction(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	// Удаление
	if err := s.repo.DeleteTransaction(ctx, id, userID); err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	// Публикация события
	if s.kafkaProducer != nil {
		if err := s.kafkaProducer.PublishTransactionDeleted(ctx, tx); err != nil {
			log.Printf("Warning: failed to publish transaction deleted event: %v", err)
		}
	}

	log.Printf("Transaction deleted: %s", id)
	return nil
}

// GetBalance получает баланс пользователя
func (s *TransactionService) GetBalance(ctx context.Context, userID, currency string) (*models.Balance, error) {
	if currency == "" {
		currency = "RUB"
	}

	if !s.isCurrencyAllowed(currency) {
		return nil, fmt.Errorf("currency %s is not allowed", currency)
	}

	balance, err := s.repo.GetBalance(ctx, userID, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// Category methods

func (s *TransactionService) CreateCategory(ctx context.Context, cat *models.Category) error {
	if err := s.validateCategory(cat); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.CreateCategory(ctx, cat); err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	log.Printf("Category created: %s for user %s", cat.ID, cat.UserID)
	return nil
}

func (s *TransactionService) GetCategories(ctx context.Context, userID string, txType *models.TransactionType) ([]*models.Category, error) {
	categories, err := s.repo.GetCategories(ctx, userID, txType)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	return categories, nil
}

func (s *TransactionService) UpdateCategory(ctx context.Context, cat *models.Category) error {
	if cat.Name == "" {
		return fmt.Errorf("category name is required")
	}

	if err := s.repo.UpdateCategory(ctx, cat); err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	log.Printf("Category updated: %s", cat.ID)
	return nil
}

func (s *TransactionService) DeleteCategory(ctx context.Context, id, userID string) error {
	if err := s.repo.DeleteCategory(ctx, id, userID); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	log.Printf("Category deleted: %s", id)
	return nil
}

// Validation helpers

func (s *TransactionService) validateTransaction(tx *models.Transaction) error {
	if tx.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if tx.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	if tx.Currency == "" {
		tx.Currency = "RUB"
	}

	if !s.isCurrencyAllowed(tx.Currency) {
		return fmt.Errorf("currency %s is not allowed", tx.Currency)
	}

	if tx.Type != models.TransactionTypeIncome && tx.Type != models.TransactionTypeExpense {
		return fmt.Errorf("type must be 'income' or 'expense'")
	}

	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}

	if tx.Tags == nil {
		tx.Tags = []string{}
	}

	return nil
}

func (s *TransactionService) validateCategory(cat *models.Category) error {
	if cat.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if cat.Name == "" {
		return fmt.Errorf("name is required")
	}

	if cat.Type != models.TransactionTypeIncome && cat.Type != models.TransactionTypeExpense {
		return fmt.Errorf("type must be 'income' or 'expense'")
	}

	return nil
}

func (s *TransactionService) isCurrencyAllowed(currency string) bool {
	_, ok := s.allowedCurrencies[currency]
	return ok
}
