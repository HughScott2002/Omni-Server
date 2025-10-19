package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"example.com/transactions/v1/src/models/events"
	"github.com/segmentio/kafka-go"
)

const (
	transactionCreatedTopic  = "transaction-created"
	transactionCompletedTopic = "transaction-completed"
	transactionFailedTopic   = "transaction-failed"
	moneyReceivedTopic       = "money-received"
	moneySentTopic           = "money-sent"
	cardPurchaseTopic        = "card-purchase"
	cardRefundTopic          = "card-refund"
	kafkaBroker              = "broker:9092"
)

// createKafkaWriter creates a new Kafka writer with standard configuration
func createKafkaWriter(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(kafkaBroker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: time.Millisecond * 100,
		WriteTimeout: time.Second * 10,
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
	}
}

// ProduceTransactionCreatedEvent publishes a transaction created event to Kafka
func ProduceTransactionCreatedEvent(event events.TransactionCreatedEvent) error {
	w := createKafkaWriter(transactionCreatedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction created event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use sender account ID as key for partitioning
	key := event.SenderAccountID
	if key == "" {
		key = event.ReceiverAccountID
	}

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(key),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write transaction created event: %v", err)
	}

	return nil
}

// ProduceTransactionCompletedEvent publishes a transaction completed event to Kafka
func ProduceTransactionCompletedEvent(event events.TransactionCompletedEvent) error {
	w := createKafkaWriter(transactionCompletedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction completed event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	key := event.SenderAccountID
	if key == "" {
		key = event.ReceiverAccountID
	}

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(key),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write transaction completed event: %v", err)
	}

	return nil
}

// ProduceTransactionFailedEvent publishes a transaction failed event to Kafka
func ProduceTransactionFailedEvent(event events.TransactionFailedEvent) error {
	w := createKafkaWriter(transactionFailedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction failed event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	key := event.SenderAccountID
	if key == "" {
		key = event.ReceiverAccountID
	}

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(key),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write transaction failed event: %v", err)
	}

	return nil
}

// ProduceMoneyReceivedEvent publishes a money received event to Kafka
func ProduceMoneyReceivedEvent(event events.MoneyReceivedEvent) error {
	w := createKafkaWriter(moneyReceivedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal money received event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.AccountID),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write money received event: %v", err)
	}

	return nil
}

// ProduceMoneySentEvent publishes a money sent event to Kafka
func ProduceMoneySentEvent(event events.MoneySentEvent) error {
	w := createKafkaWriter(moneySentTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal money sent event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.AccountID),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write money sent event: %v", err)
	}

	return nil
}

// ProduceCardPurchaseEvent publishes a card purchase event to Kafka
func ProduceCardPurchaseEvent(event events.CardPurchaseEvent) error {
	w := createKafkaWriter(cardPurchaseTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal card purchase event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.AccountID),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write card purchase event: %v", err)
	}

	return nil
}

// ProduceCardRefundEvent publishes a card refund event to Kafka
func ProduceCardRefundEvent(event events.CardRefundEvent) error {
	w := createKafkaWriter(cardRefundTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal card refund event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.AccountID),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write card refund event: %v", err)
	}

	return nil
}
