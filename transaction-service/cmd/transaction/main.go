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

	"github.com/InsafMin/gomoney/transaction-service/internal/config"
	"github.com/InsafMin/gomoney/transaction-service/internal/handler"
	"github.com/InsafMin/gomoney/transaction-service/internal/kafka"
	"github.com/InsafMin/gomoney/transaction-service/internal/repository"
	"github.com/InsafMin/gomoney/transaction-service/internal/server"
	"github.com/InsafMin/gomoney/transaction-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Printf("Starting %s in %s mode", cfg.ServiceName, cfg.Environment)

	db, err := initDatabase(cfg)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}
	defer db.Close()

	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopicTrans)
	defer kafkaProducer.Close()

	repo := repository.NewRepository(db)
	transactionService := service.NewService(repo, kafkaProducer, cfg.AllowedCurrencies)

	h := handler.NewHandler(transactionService)
	grpcServer := server.NewServer(cfg.GRPCPort, h)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gracefully...")
		grpcServer.Stop()
	}()

	log.Printf("Transaction service is ready on port %d", cfg.GRPCPort)
	return grpcServer.Start()
}

func initDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL")
	return db, nil
}
