package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config содержит конфигурацию сервиса аутентификации
type Config struct {
	Environment string
	ServiceName string

	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string

	// JWT
	JWTSecret         string
	JWTExpiration     time.Duration
	TokenRefreshTTL   time.Duration
	PasswordMinLength int

	// gRPC
	GRPCPort int

	// Logging
	LogLevel string
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		ServiceName: getEnv("SERVICE_NAME", "auth-service"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvAsInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "finance_user"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "finance_tracker"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvAsInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		JWTSecret:         getEnv("JWT_SECRET", ""),
		JWTExpiration:     getEnvAsDuration("JWT_EXPIRATION", 15*time.Minute),
		TokenRefreshTTL:   getEnvAsDuration("TOKEN_REFRESH_TTL", 168*time.Hour),
		PasswordMinLength: getEnvAsInt("PASSWORD_MIN_LENGTH", 6),

		GRPCPort: getEnvAsInt("GRPC_PORT", 50051),

		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.DBHost == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.PasswordMinLength < 6 {
		return fmt.Errorf("PASSWORD_MIN_LENGTH must be at least 6")
	}
	return nil
}

// GetDatabaseURL возвращает строку подключения к БД
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

// GetRedisAddr возвращает адрес Redis
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

// Helper functions

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

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
