package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/InsafMin/gomoney/analytics-service/internal/cache"
	"github.com/InsafMin/gomoney/analytics-service/internal/handler"
	"github.com/InsafMin/gomoney/analytics-service/internal/kafka"
	"github.com/InsafMin/gomoney/analytics-service/internal/repository"
	"github.com/InsafMin/gomoney/analytics-service/internal/server"
	"github.com/InsafMin/gomoney/analytics-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

// run инициализирует и запускает все компоненты сервиса
func run() error {
	// Загрузка конфигурации из переменных окружения
	cfg := loadConfig()

	// Подключение к PostgreSQL
	db, err := initDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}
	defer db.Close()

	// Подключение к Redis
	redisClient := initRedis(cfg.RedisURL)
	defer redisClient.Close()

	// Проверка подключения к Redis
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Инициализация слоев приложения
	repo := repository.NewRepository(db)
	cacheLayer := cache.NewCache(redisClient, cfg.CacheTTL)
	svc := service.NewService(repo, cacheLayer)
	h := handler.NewHandler(svc)

	// Инициализация Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(
		cfg.KafkaBrokers,
		cfg.KafkaGroup,
		cfg.KafkaTopic,
		repo,
		cacheLayer,
	)
	if err != nil {
		return fmt.Errorf("failed to create kafka consumer: %w", err)
	}
	defer kafkaConsumer.Close()

	// Запуск Kafka consumer в отдельной горутине
	go func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			log.Printf("Kafka consumer error: %v", err)
		}
	}()

	// Инициализация gRPC сервера
	grpcServer := server.NewServer(cfg.GRPCPort, h)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gracefully...")
		grpcServer.Stop()
	}()

	// Запуск gRPC сервера (блокирующий вызов)
	return grpcServer.Start()
}

// Config содержит конфигурацию сервиса
type Config struct {
	DatabaseURL  string
	RedisURL     string
	GRPCPort     int
	KafkaBrokers []string
	KafkaGroup   string
	KafkaTopic   string
	CacheTTL     time.Duration
}

// loadConfig загружает конфигурацию из переменных окружения
func loadConfig() Config {
	return Config{
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/analytics?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
		GRPCPort:     getEnvAsInt("GRPC_PORT", 50052),
		KafkaBrokers: getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaGroup:   getEnv("KAFKA_GROUP", "analytics-service"),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "transactions"),
		CacheTTL:     getEnvAsDuration("CACHE_TTL", 5*time.Minute),
	}
}

// initDatabase инициализирует подключение к PostgreSQL
func initDatabase(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL")
	return db, nil
}

// initRedis инициализирует подключение к Redis
func initRedis(redisURL string) *redis.Client {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Failed to parse Redis URL, using defaults: %v", err)
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opt)
	log.Println("Successfully connected to Redis")
	return client
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt возвращает значение переменной окружения как int
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsSlice возвращает значение переменной окружения как срез строк
func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return []string{value}
	}
	return defaultValue
}

// getEnvAsDuration возвращает значение переменной окружения как duration
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
