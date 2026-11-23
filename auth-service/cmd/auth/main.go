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

	"github.com/InsafMin/gomoney/auth-service/internal/config"
	"github.com/InsafMin/gomoney/auth-service/internal/handler"
	"github.com/InsafMin/gomoney/auth-service/internal/jwt"
	"github.com/InsafMin/gomoney/auth-service/internal/password"
	"github.com/InsafMin/gomoney/auth-service/internal/repository"
	"github.com/InsafMin/gomoney/auth-service/internal/server"
	"github.com/InsafMin/gomoney/auth-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Printf("Starting %s in %s mode", cfg.ServiceName, cfg.Environment)

	// Подключение к PostgreSQL
	db, err := initDatabase(cfg)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}
	defer db.Close()

	// Подключение к Redis
	redisClient, err := initRedis(cfg)
	if err != nil {
		return fmt.Errorf("failed to init redis: %w", err)
	}
	defer redisClient.Close()

	// Инициализация компонентов
	repo := repository.NewRepository(db)
	jwtManager := jwt.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration, cfg.TokenRefreshTTL)
	passwordHasher := password.NewBcryptHasher()

	// Инициализация сервиса
	authService := service.NewService(repo, jwtManager, passwordHasher, redisClient, cfg.PasswordMinLength)

	// Инициализация gRPC handler
	h := handler.NewHandler(authService)

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

	// Запуск периодической очистки токенов
	go cleanupExpiredTokens(repo)

	// Запуск gRPC сервера (блокирующий вызов)
	log.Printf("Auth service is ready to accept connections on port %d", cfg.GRPCPort)
	return grpcServer.Start()
}

func initDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL")
	return db, nil
}

func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("Successfully connected to Redis")
	return client, nil
}

func cleanupExpiredTokens(repo repository.Repository) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		if err := repo.CleanupExpiredTokens(ctx); err != nil {
			log.Printf("Failed to cleanup expired tokens: %v", err)
		} else {
			log.Println("Successfully cleaned up expired tokens")
		}
	}
}
