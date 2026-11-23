package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/InsafMin/gomoney/analytics-service/internal/cache"
	"github.com/InsafMin/gomoney/analytics-service/internal/models"
	"github.com/InsafMin/gomoney/analytics-service/internal/repository"
)

type Consumer struct {
	consumer sarama.ConsumerGroup
	repo     *repository.Repository
	cache    *cache.Cache
	topic    string
	ready    chan bool
}

// NewConsumer создает новый Kafka consumer для обработки событий транзакций
func NewConsumer(brokers []string, group, topic string, repo *repository.Repository, cache *cache.Cache) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V3_3_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return &Consumer{
		consumer: consumer,
		repo:     repo,
		cache:    cache,
		topic:    topic,
		ready:    make(chan bool),
	}, nil
}

// Start запускает consumer и начинает обработку сообщений
func (c *Consumer) Start(ctx context.Context) error {
	handler := &consumerGroupHandler{
		repo:  c.repo,
		cache: c.cache,
		ready: c.ready,
	}

	go func() {
		for {
			if err := c.consumer.Consume(ctx, []string{c.topic}, handler); err != nil {
				log.Printf("Error from consumer: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
			handler.ready = make(chan bool)
		}
	}()

	<-handler.ready
	log.Println("Kafka consumer started and ready")
	return nil
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	return c.consumer.Close()
}

type consumerGroupHandler struct {
	repo  *repository.Repository
	cache *cache.Cache
	ready chan bool
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if err := h.processMessage(session.Context(), message); err != nil {
			log.Printf("Error processing message: %v", err)
			continue
		}
		session.MarkMessage(message, "")
	}
	return nil
}

func (h *consumerGroupHandler) processMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event models.KafkaEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("Processing event: %s (ID: %s)", event.EventType, event.EventID)

	switch event.EventType {
	case "TransactionCreated", "TransactionUpdated":
		return h.handleTransactionCreatedOrUpdated(ctx, event)
	case "TransactionDeleted":
		return h.handleTransactionDeleted(ctx, event)
	case "CategoryCreated", "CategoryUpdated":
		return h.handleCategoryCreatedOrUpdated(ctx, event)
	default:
		log.Printf("Unknown event type: %s", event.EventType)
	}

	return nil
}

func (h *consumerGroupHandler) handleTransactionCreatedOrUpdated(ctx context.Context, event models.KafkaEvent) error {
	payload := event.Payload

	transaction := &models.Transaction{
		ID:          getStringField(payload, "transaction_id"),
		UserID:      getStringField(payload, "user_id"),
		Amount:      getFloatField(payload, "amount"),
		CategoryID:  getStringField(payload, "category_id"),
		Description: getStringField(payload, "description"),
		Type:        getStringField(payload, "type"),
		CreatedAt:   event.Timestamp,
		UpdatedAt:   event.Timestamp,
	}

	if dateStr := getStringField(payload, "date"); dateStr != "" {
		if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
			transaction.Date = date
		}
	}

	if err := h.repo.SaveTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	if err := h.cache.InvalidateUserCache(ctx, transaction.UserID); err != nil {
		log.Printf("Warning: failed to invalidate cache for user %s: %v", transaction.UserID, err)
	}

	log.Printf("Transaction %s processed successfully for user %s", transaction.ID, transaction.UserID)
	return nil
}

func (h *consumerGroupHandler) handleTransactionDeleted(ctx context.Context, event models.KafkaEvent) error {
	transactionID := getStringField(event.Payload, "transaction_id")
	userID := getStringField(event.Payload, "user_id")

	if err := h.repo.DeleteTransaction(ctx, transactionID); err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	if err := h.cache.InvalidateUserCache(ctx, userID); err != nil {
		log.Printf("Warning: failed to invalidate cache for user %s: %v", userID, err)
	}

	log.Printf("Transaction %s deleted successfully", transactionID)
	return nil
}

func (h *consumerGroupHandler) handleCategoryCreatedOrUpdated(ctx context.Context, event models.KafkaEvent) error {
	payload := event.Payload

	category := &models.Category{
		ID:        getStringField(payload, "category_id"),
		UserID:    getStringField(payload, "user_id"),
		Name:      getStringField(payload, "name"),
		CreatedAt: event.Timestamp,
		UpdatedAt: event.Timestamp,
	}

	if err := h.repo.SaveCategory(ctx, category); err != nil {
		return fmt.Errorf("failed to save category: %w", err)
	}

	log.Printf("Category %s processed successfully", category.ID)
	return nil
}

// getStringField извлекает строковое поле из payload
func getStringField(payload map[string]interface{}, key string) string {
	if val, ok := payload[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getFloatField извлекает числовое поле из payload
func getFloatField(payload map[string]interface{}, key string) float64 {
	if val, ok := payload[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}
