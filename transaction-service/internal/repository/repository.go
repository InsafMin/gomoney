package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/InsafMin/gomoney/transaction-service/internal/models"
)

type Repository interface {
	// Transactions
	CreateTransaction(ctx context.Context, tx *models.Transaction) error
	GetTransaction(ctx context.Context, id, userID string) (*models.Transaction, error)
	GetTransactions(ctx context.Context, filter *models.TransactionFilter) (*models.TransactionSummary, error)
	UpdateTransaction(ctx context.Context, tx *models.Transaction) error
	DeleteTransaction(ctx context.Context, id, userID string) error
	GetBalance(ctx context.Context, userID, currency string) (*models.Balance, error)

	// Categories
	CreateCategory(ctx context.Context, cat *models.Category) error
	GetCategory(ctx context.Context, id, userID string) (*models.Category, error)
	GetCategories(ctx context.Context, userID string, txType *models.TransactionType) ([]*models.Category, error)
	UpdateCategory(ctx context.Context, cat *models.Category) error
	DeleteCategory(ctx context.Context, id, userID string) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateTransaction создает новую транзакцию
func (r *PostgresRepository) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (id, user_id, amount, currency, category_id, description, date, type, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	tx.ID = uuid.New().String()
	tx.CreatedAt = time.Now()
	tx.UpdatedAt = time.Now()

	// Преобразуем пустую строку в NULL для category_id
	var categoryID interface{}
	if tx.CategoryID == "" {
		categoryID = nil
	} else {
		categoryID = tx.CategoryID
	}

	_, err := r.db.ExecContext(ctx, query,
		tx.ID,
		tx.UserID,
		tx.Amount,
		tx.Currency,
		categoryID,
		tx.Description,
		tx.Date,
		string(tx.Type),
		pq.Array(tx.Tags),
		tx.CreatedAt,
		tx.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// GetTransaction получает транзакцию по ID
func (r *PostgresRepository) GetTransaction(ctx context.Context, id, userID string) (*models.Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.amount, t.currency, t.category_id, t.description, 
		       t.date, t.type, t.tags, t.created_at, t.updated_at,
		       c.id, c.name, c.type, c.color, c.icon
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.id = $1 AND t.user_id = $2
	`

	tx := &models.Transaction{
		Category: &models.Category{},
	}
	var txType, catType string
	var tags pq.StringArray
	var categoryID sql.NullString
	var catID, catName, catColor, catIcon sql.NullString

	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&tx.ID,
		&tx.UserID,
		&tx.Amount,
		&tx.Currency,
		&categoryID,
		&tx.Description,
		&tx.Date,
		&txType,
		&tags,
		&tx.CreatedAt,
		&tx.UpdatedAt,
		&catID,
		&catName,
		&catType,
		&catColor,
		&catIcon,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	tx.Type = models.TransactionType(txType)
	tx.Tags = tags

	if catID.Valid {
		tx.Category.ID = catID.String
		tx.Category.Name = catName.String
		tx.Category.Type = models.TransactionType(catType)
		tx.Category.Color = catColor.String
		tx.Category.Icon = catIcon.String
		tx.CategoryID = categoryID.String
	} else {
		tx.Category = nil
	}

	return tx, nil
}

// GetTransactions получает список транзакций с фильтрацией
func (r *PostgresRepository) GetTransactions(ctx context.Context, filter *models.TransactionFilter) (*models.TransactionSummary, error) {
	// Строим WHERE условия
	whereClauses := []string{"t.user_id = $1"}
	args := []interface{}{filter.UserID}
	argIndex := 2

	if filter.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.date >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.date <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	if filter.CategoryID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.category_id = $%d", argIndex))
		args = append(args, *filter.CategoryID)
		argIndex++
	}

	if filter.Type != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.type = $%d", argIndex))
		args = append(args, string(*filter.Type))
		argIndex++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Получаем общее количество и суммы
	summaryQuery := fmt.Sprintf(`
		SELECT 
			COUNT(*),
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expense
		FROM transactions t
		WHERE %s
	`, whereClause)

	var total int
	var totalIncome, totalExpense float64
	err := r.db.QueryRowContext(ctx, summaryQuery, args[:len(args)]...).Scan(&total, &totalIncome, &totalExpense)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction summary: %w", err)
	}

	// Определяем сортировку
	sortBy := "date"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "DESC"
	if !filter.SortDesc {
		sortOrder = "ASC"
	}

	// Получаем транзакции
	limit := 50
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	offset := filter.Offset

	query := fmt.Sprintf(`
		SELECT t.id, t.user_id, t.amount, t.currency, t.category_id, t.description,
		       t.date, t.type, t.tags, t.created_at, t.updated_at,
		       c.id, c.name, c.type, c.color, c.icon
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE %s
		ORDER BY t.%s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		tx := &models.Transaction{Category: &models.Category{}}
		var txType, catType string
		var tags pq.StringArray
		var categoryID sql.NullString
		var catID, catName, catColor, catIcon sql.NullString

		err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.Amount,
			&tx.Currency,
			&categoryID,
			&tx.Description,
			&tx.Date,
			&txType,
			&tags,
			&tx.CreatedAt,
			&tx.UpdatedAt,
			&catID,
			&catName,
			&catType,
			&catColor,
			&catIcon,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		tx.Type = models.TransactionType(txType)
		tx.Tags = tags

		if catID.Valid {
			tx.Category.ID = catID.String
			tx.Category.Name = catName.String
			tx.Category.Type = models.TransactionType(catType)
			tx.Category.Color = catColor.String
			tx.Category.Icon = catIcon.String
			tx.CategoryID = categoryID.String
		} else {
			tx.Category = nil
		}

		transactions = append(transactions, tx)
	}

	page := (offset / limit) + 1
	if limit == 0 {
		page = 1
	}

	return &models.TransactionSummary{
		Transactions: transactions,
		Total:        total,
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
		Page:         page,
		PageSize:     limit,
	}, nil
}

// UpdateTransaction обновляет транзакцию
func (r *PostgresRepository) UpdateTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
		UPDATE transactions
		SET amount = $3, category_id = $4, description = $5, date = $6, tags = $7, updated_at = $8
		WHERE id = $1 AND user_id = $2
	`

	tx.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		tx.ID,
		tx.UserID,
		tx.Amount,
		tx.CategoryID,
		tx.Description,
		tx.Date,
		pq.Array(tx.Tags),
		tx.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}

// DeleteTransaction удаляет транзакцию
func (r *PostgresRepository) DeleteTransaction(ctx context.Context, id, userID string) error {
	query := `DELETE FROM transactions WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}

// GetBalance получает баланс пользователя
func (r *PostgresRepository) GetBalance(ctx context.Context, userID, currency string) (*models.Balance, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) as balance,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expense,
			MIN(date) as period_start,
			MAX(date) as period_end
		FROM transactions
		WHERE user_id = $1 AND currency = $2
	`

	balance := &models.Balance{Currency: currency}
	var periodStart, periodEnd sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID, currency).Scan(
		&balance.Balance,
		&balance.TotalIncome,
		&balance.TotalExpense,
		&periodStart,
		&periodEnd,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if periodStart.Valid {
		balance.PeriodStart = periodStart.Time
	}
	if periodEnd.Valid {
		balance.PeriodEnd = periodEnd.Time
	}

	return balance, nil
}

// Category methods

func (r *PostgresRepository) CreateCategory(ctx context.Context, cat *models.Category) error {
	query := `
		INSERT INTO categories (id, user_id, name, type, color, icon, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	cat.ID = uuid.New().String()
	cat.CreatedAt = time.Now()
	cat.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		cat.ID,
		cat.UserID,
		cat.Name,
		string(cat.Type),
		cat.Color,
		cat.Icon,
		cat.ParentID,
		cat.CreatedAt,
		cat.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetCategory(ctx context.Context, id, userID string) (*models.Category, error) {
	query := `
		SELECT id, user_id, name, type, color, icon, parent_id, created_at, updated_at
		FROM categories
		WHERE id = $1 AND (user_id = $2 OR user_id = 'system')
	`

	cat := &models.Category{}
	var catType string

	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&cat.ID,
		&cat.UserID,
		&cat.Name,
		&catType,
		&cat.Color,
		&cat.Icon,
		&cat.ParentID,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("category not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	cat.Type = models.TransactionType(catType)
	return cat, nil
}

func (r *PostgresRepository) GetCategories(ctx context.Context, userID string, txType *models.TransactionType) ([]*models.Category, error) {
	query := `
		SELECT id, user_id, name, type, color, icon, parent_id, created_at, updated_at
		FROM categories
		WHERE (user_id = $1 OR user_id = 'system')
	`

	args := []interface{}{userID}

	if txType != nil {
		query += " AND type = $2"
		args = append(args, string(*txType))
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []*models.Category
	for rows.Next() {
		cat := &models.Category{}
		var catType string

		err := rows.Scan(
			&cat.ID,
			&cat.UserID,
			&cat.Name,
			&catType,
			&cat.Color,
			&cat.Icon,
			&cat.ParentID,
			&cat.CreatedAt,
			&cat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		cat.Type = models.TransactionType(catType)
		categories = append(categories, cat)
	}

	return categories, nil
}

func (r *PostgresRepository) UpdateCategory(ctx context.Context, cat *models.Category) error {
	query := `
		UPDATE categories
		SET name = $3, color = $4, icon = $5, updated_at = $6
		WHERE id = $1 AND user_id = $2
	`

	cat.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		cat.ID,
		cat.UserID,
		cat.Name,
		cat.Color,
		cat.Icon,
		cat.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("category not found or is system category")
	}

	return nil
}

func (r *PostgresRepository) DeleteCategory(ctx context.Context, id, userID string) error {
	query := `DELETE FROM categories WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("category not found or is system category")
	}

	return nil
}
