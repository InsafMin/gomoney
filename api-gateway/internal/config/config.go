package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment string
	ServiceName string

	// HTTP Server
	HTTPPort    int
	HTTPTimeout time.Duration

	// Service URLs
	AuthServiceURL        string
	TransactionServiceURL string
	AnalyticsServiceURL   string

	// CORS
	AllowOrigins []string

	// Circuit Breaker
	CircuitBreakerTimeout    time.Duration
	CircuitBreakerMaxRequest uint32
	CircuitBreakerInterval   time.Duration

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   time.Duration

	// Logging
	LogLevel string
}

func Load() (*Config, error) {
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		ServiceName: getEnv("SERVICE_NAME", "api-gateway"),

		HTTPPort:    getEnvAsInt("HTTP_PORT", 8080),
		HTTPTimeout: getEnvAsDuration("HTTP_TIMEOUT", 30*time.Second),

		AuthServiceURL:        getEnv("AUTH_SERVICE_URL", "auth-service:50051"),
		TransactionServiceURL: getEnv("TRANSACTION_SERVICE_URL", "transaction-service:50052"),
		AnalyticsServiceURL:   getEnv("ANALYTICS_SERVICE_URL", "analytics-service:50053"),

		AllowOrigins: []string{"*"},

		CircuitBreakerTimeout:    5 * time.Second,
		CircuitBreakerMaxRequest: 3,
		CircuitBreakerInterval:   10 * time.Second,

		RateLimitRequests: getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvAsDuration("RATE_LIMIT_WINDOW", 1*time.Minute),

		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.HTTPPort <= 0 {
		return fmt.Errorf("HTTP_PORT must be positive")
	}
	return nil
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

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
