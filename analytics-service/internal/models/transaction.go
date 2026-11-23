package models

import "time"

// Transaction представляет транзакцию из Kafka
type Transaction struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	Amount      float64   `json:"amount" db:"amount"`
	CategoryID  string    `json:"category_id" db:"category_id"`
	Description string    `json:"description" db:"description"`
	Date        time.Time `json:"date" db:"date"`
	Type        string    `json:"type" db:"type"` // income или expense
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Category представляет категорию транзакции
type Category struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// KafkaEvent представляет событие из Kafka
type KafkaEvent struct {
	EventType string                 `json:"event_type"`
	EventID   string                 `json:"event_id"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// DailySummary суммарная статистика за день
type DailySummary struct {
	UserID  string    `db:"user_id"`
	Date    time.Time `db:"date"`
	Income  float64   `db:"income"`
	Expense float64   `db:"expense"`
	Balance float64   `db:"balance"`
}

// CategoryStat статистика по категории
type CategoryStat struct {
	CategoryID       string  `db:"category_id"`
	CategoryName     string  `db:"category_name"`
	Type             string  `db:"type"`
	Amount           float64 `db:"amount"`
	TransactionCount int32   `db:"transaction_count"`
	Percentage       float64 `db:"-"`
}

// TrendPoint точка на графике тренда
type TrendPoint struct {
	Date   time.Time `json:"date" db:"date"`
	Amount float64   `json:"amount" db:"amount"`
}

// TrendData данные трендов с метаинформацией
type TrendData struct {
	Period     string       `json:"period"`      // daily, weekly, monthly
	Type       string       `json:"type"`        // income, expense
	TotalCount int          `json:"total_count"` // количество точек
	Points     []TrendPoint `json:"points"`      // данные по периодам
}
