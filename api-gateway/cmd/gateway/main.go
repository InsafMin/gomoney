package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/InsafMin/gomoney/api-gateway/internal/clients"
	"github.com/InsafMin/gomoney/api-gateway/internal/config"
	"github.com/InsafMin/gomoney/api-gateway/internal/handlers"
	"github.com/InsafMin/gomoney/api-gateway/internal/middleware"
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

	log.Printf("Starting %s in %s mode on port %d", cfg.ServiceName, cfg.Environment, cfg.HTTPPort)

	authClient, err := clients.NewAuthClient(cfg.AuthServiceURL)
	if err != nil {
		return fmt.Errorf("failed to create auth client: %w", err)
	}
	defer authClient.Close()

	transactionClient, err := clients.NewTransactionClient(cfg.TransactionServiceURL)
	if err != nil {
		return fmt.Errorf("failed to create transaction client: %w", err)
	}
	defer transactionClient.Close()

	analyticsClient, err := clients.NewAnalyticsClient(cfg.AnalyticsServiceURL)
	if err != nil {
		return fmt.Errorf("failed to create analytics client: %w", err)
	}
	defer analyticsClient.Close()

	router := setupRouter(cfg, authClient, transactionClient, analyticsClient)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  cfg.HTTPTimeout,
		WriteTimeout: cfg.HTTPTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("API Gateway is ready on port %d", cfg.HTTPPort)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}

func setupRouter(cfg *config.Config, authClient *clients.AuthClient, transactionClient *clients.TransactionClient, analyticsClient *clients.AnalyticsClient) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	router.Use(rateLimiter.Middleware())

	authMiddleware := middleware.NewAuthMiddleware(authClient)
	authHandler := handlers.NewAuthHandler(authClient)
	transactionHandler := handlers.NewTransactionHandler(transactionClient)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsClient)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": cfg.ServiceName,
		})
	})

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/profile", authMiddleware.ValidateToken(), authHandler.GetProfile)
		}

		transactions := api.Group("/transactions")
		transactions.Use(authMiddleware.ValidateToken())
		{
			transactions.POST("", transactionHandler.CreateTransaction)
			transactions.GET("", transactionHandler.GetTransactions)
			transactions.GET("/balance", transactionHandler.GetBalance)
		}

		categories := api.Group("/categories")
		categories.Use(authMiddleware.ValidateToken())
		{
			categories.GET("", transactionHandler.GetCategories)
			categories.POST("", transactionHandler.CreateCategory)
		}

		analytics := api.Group("/analytics")
		analytics.Use(authMiddleware.ValidateToken())
		{
			analytics.GET("/daily", analyticsHandler.GetDailyReport)
			analytics.GET("/period", analyticsHandler.GetPeriodReport)
			analytics.GET("/categories", analyticsHandler.GetCategoryStats)
			analytics.GET("/trends", analyticsHandler.GetTrend)
			analytics.GET("/comparison", analyticsHandler.GetMonthlyComparison)
		}
	}

	return router
}
