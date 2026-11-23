package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/InsafMin/gomoney/analytics-service/internal/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// SaveTransaction сохраняет транзакцию из Kafka
func (r *Repository) SaveTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (id, user_id, amount, currency, category_id, description, date, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			amount = EXCLUDED.amount,
			category_id = EXCLUDED.category_id,
			description = EXCLUDED.description,
			date = EXCLUDED.date,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		tx.ID, tx.UserID, tx.Amount, tx.CategoryID,
		tx.Description, tx.Date, tx.Type, tx.CreatedAt, tx.UpdatedAt,
	)
	return err
}

// DeleteTransaction удаляет транзакцию
func (r *Repository) DeleteTransaction(ctx context.Context, transactionID string) error {
	query := `DELETE FROM transactions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, transactionID)
	return err
}

// GetDailyReport получает дневной отчёт
func (r *Repository) GetDailyReport(ctx context.Context, userID string, date time.Time) (*models.DailySummary, []*models.CategoryStat, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Суммарная статистика
	summaryQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
		FROM transactions
		WHERE user_id = $1 AND date >= $2 AND date < $3
	`
	var summary models.DailySummary
	summary.UserID = userID
	summary.Date = startOfDay

	err := r.db.QueryRowContext(ctx, summaryQuery, userID, startOfDay, endOfDay).
		Scan(&summary.Income, &summary.Expense)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, fmt.Errorf("failed to get daily summary: %w", err)
	}
	summary.Balance = summary.Income - summary.Expense

	// Статистика по категориям
	categoryQuery := `
		SELECT 
			t.category_id,
			COALESCE(c.name, 'Unknown') as category_name,
			t.type,
			SUM(t.amount) as amount,
			COUNT(*) as transaction_count
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.user_id = $1 AND t.date >= $2 AND t.date < $3
		GROUP BY t.category_id, c.name, t.type
		ORDER BY amount DESC
	`
	rows, err := r.db.QueryContext(ctx, categoryQuery, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get category stats: %w", err)
	}
	defer rows.Close()

	var categoryStats []*models.CategoryStat
	for rows.Next() {
		var stat models.CategoryStat
		if err := rows.Scan(&stat.CategoryID, &stat.CategoryName, &stat.Type, &stat.Amount, &stat.TransactionCount); err != nil {
			return nil, nil, fmt.Errorf("failed to scan category stat: %w", err)
		}
		categoryStats = append(categoryStats, &stat)
	}

	return &summary, categoryStats, nil
}

// GetPeriodReport получает отчёт за период
func (r *Repository) GetPeriodReport(ctx context.Context, userID string, startDate, endDate time.Time) (float64, float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
		FROM transactions
		WHERE user_id = $1 AND date >= $2 AND date < $3
	`
	var income, expense float64
	err := r.db.QueryRowContext(ctx, query, userID, startDate, endDate).Scan(&income, &expense)
	if err != nil && err != sql.ErrNoRows {
		return 0, 0, err
	}
	return income, expense, nil
}

// GetCategoryStats получает статистику по категориям за период
func (r *Repository) GetCategoryStats(ctx context.Context, userID string, txType *string, startDate, endDate *time.Time, limit int) ([]*models.CategoryStat, error) {
	query := `
		SELECT 
			t.category_id,
			COALESCE(c.name, 'Unknown') as category_name,
			t.type,
			SUM(t.amount) as amount,
			COUNT(*) as transaction_count
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.user_id = $1
	`
	args := []interface{}{userID}
	argNum := 2

	if txType != nil {
		query += fmt.Sprintf(" AND t.type = $%d", argNum)
		args = append(args, *txType)
		argNum++
	}

	if startDate != nil {
		query += fmt.Sprintf(" AND t.date >= $%d", argNum)
		args = append(args, *startDate)
		argNum++
	}

	if endDate != nil {
		query += fmt.Sprintf(" AND t.date < $%d", argNum)
		args = append(args, *endDate)
		argNum++
	}

	query += ` GROUP BY t.category_id, c.name, t.type ORDER BY amount DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.CategoryStat
	for rows.Next() {
		var stat models.CategoryStat
		if err := rows.Scan(&stat.CategoryID, &stat.CategoryName, &stat.Type, &stat.Amount, &stat.TransactionCount); err != nil {
			return nil, err
		}
		stats = append(stats, &stat)
	}

	return stats, nil
}

// GetTrendData получает данные для трендов
func (r *Repository) GetTrendData(ctx context.Context, userID, txType string, startDate, endDate time.Time, period string) ([]time.Time, []float64, error) {
	var groupBy string
	switch period {
	case "daily":
		groupBy = "DATE(date)"
	case "weekly":
		groupBy = "DATE_TRUNC('week', date)"
	case "monthly":
		groupBy = "DATE_TRUNC('month', date)"
	default:
		groupBy = "DATE(date)"
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as period,
			SUM(amount) as total
		FROM transactions
		WHERE user_id = $1 AND type = $2 AND date >= $3 AND date < $4
		GROUP BY period
		ORDER BY period
	`, groupBy)

	rows, err := r.db.QueryContext(ctx, query, userID, txType, startDate, endDate)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var dates []time.Time
	var values []float64

	for rows.Next() {
		var date time.Time
		var value float64
		if err := rows.Scan(&date, &value); err != nil {
			return nil, nil, err
		}
		dates = append(dates, date)
		values = append(values, value)
	}

	return dates, values, nil
}

// SaveCategory сохраняет категорию
func (r *Repository) SaveCategory(ctx context.Context, cat *models.Category) error {
	query := `
		INSERT INTO categories (id, user_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		cat.ID, cat.UserID, cat.Name, cat.CreatedAt, cat.UpdatedAt,
	)
	return err
}
