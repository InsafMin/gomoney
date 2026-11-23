package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Environment string
	ServiceName string

	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// Auth Service
	AuthServiceURL string

	// Kafka
	KafkaBrokers       []string
	KafkaTopicTrans    string
	KafkaProducerRetry int

	// Currency
	DefaultCurrency   string
	AllowedCurrencies []string

	// gRPC
	GRPCPort int

	// Logging
	LogLevel string
}

func Load() (*Config, error) {
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		ServiceName: getEnv("SERVICE_NAME", "transaction-service"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvAsInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "finance_user"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "finance_tracker"),

		AuthServiceURL: getEnv("AUTH_SERVICE_URL", "auth-service:50051"),

		KafkaBrokers:       getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaTopicTrans:    getEnv("KAFKA_TOPIC_TRANSACTIONS", "transactions"),
		KafkaProducerRetry: getEnvAsInt("KAFKA_PRODUCER_RETRY", 3),

		DefaultCurrency:   getEnv("DEFAULT_CURRENCY", "RUB"),
		AllowedCurrencies: getEnvAsSlice("ALLOWED_CURRENCIES", []string{"RUB", "USD", "EUR"}),

		GRPCPort: getEnvAsInt("GRPC_PORT", 50052),

		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.DBHost == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	return nil
}

func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func (c *Config) IsCurrencyAllowed(currency string) bool {
	for _, allowed := range c.AllowedCurrencies {
		if strings.EqualFold(allowed, currency) {
			return true
		}
	}
	return false
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
