package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/InsafMin/gomoney/transaction-service/internal/models"
)

// TransactionEvent событие транзакции для Kafka
type TransactionEvent struct {
	EventType   string           `json:"event_type"` // created, updated, deleted
	Transaction *TransactionData `json:"transaction"`
	Timestamp   time.Time        `json:"timestamp"`
}

type TransactionData struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	CategoryID  string    `json:"category_id"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Type        string    `json:"type"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			WriteTimeout:           10 * time.Second,
			RequiredAcks:           kafka.RequireOne,
			AllowAutoTopicCreation: true,
			Async:                  false, // Синхронная запись для надежности
		},
	}
}

func (p *Producer) PublishTransactionCreated(ctx context.Context, tx *models.Transaction) error {
	return p.publishEvent(ctx, "created", tx)
}

func (p *Producer) PublishTransactionUpdated(ctx context.Context, tx *models.Transaction) error {
	return p.publishEvent(ctx, "updated", tx)
}

func (p *Producer) PublishTransactionDeleted(ctx context.Context, tx *models.Transaction) error {
	return p.publishEvent(ctx, "deleted", tx)
}

func (p *Producer) publishEvent(ctx context.Context, eventType string, tx *models.Transaction) error {
	log.Printf("Publishing %s event for transaction %s to Kafka", eventType, tx.ID)

	event := TransactionEvent{
		EventType: eventType,
		Transaction: &TransactionData{
			ID:          tx.ID,
			UserID:      tx.UserID,
			Amount:      tx.Amount,
			Currency:    tx.Currency,
			CategoryID:  tx.CategoryID,
			Description: tx.Description,
			Date:        tx.Date,
			Type:        string(tx.Type),
			Tags:        tx.Tags,
			CreatedAt:   tx.CreatedAt,
			UpdatedAt:   tx.UpdatedAt,
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(tx.UserID),
		Value: data,
		Time:  time.Now(),
	}

	log.Printf("Writing message to Kafka...")
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("Failed to write message to Kafka: %v", err)
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("Published %s event for transaction %s to Kafka successfully", eventType, tx.ID)
	return nil
}

func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
