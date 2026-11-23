package models

import "time"

// TransactionType определяет тип транзакции
type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "income"
	TransactionTypeExpense TransactionType = "expense"
)

// Transaction представляет финансовую транзакцию
type Transaction struct {
	ID          string
	UserID      string
	Amount      float64
	Currency    string
	CategoryID  string
	Category    *Category
	Description string
	Date        time.Time
	Type        TransactionType
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Category представляет категорию транзакций
type Category struct {
	ID        string
	UserID    string
	Name      string
	Type      TransactionType
	Color     string
	Icon      string
	ParentID  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Balance представляет баланс пользователя
type Balance struct {
	Balance      float64
	Currency     string
	TotalIncome  float64
	TotalExpense float64
	PeriodStart  time.Time
	PeriodEnd    time.Time
}

// TransactionFilter для фильтрации транзакций
type TransactionFilter struct {
	UserID     string
	StartDate  *time.Time
	EndDate    *time.Time
	CategoryID *string
	Type       *TransactionType
	Limit      int
	Offset     int
	SortBy     string
	SortDesc   bool
}

// TransactionSummary суммарная информация о транзакциях
type TransactionSummary struct {
	Transactions []*Transaction
	Total        int
	TotalIncome  float64
	TotalExpense float64
	Page         int
	PageSize     int
}
