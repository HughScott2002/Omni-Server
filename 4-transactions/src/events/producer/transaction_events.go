package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"example.com/transactions/v1/src/models/events"
	"github.com/segmentio/kafka-go"
)

const (
	transactionCreatedTopic   = "transaction-created"
	transactionCompletedTopic = "transaction-completed"
	transactionFailedTopic    = "transaction-failed"
	moneyReceivedTopic        = "money-received"
	moneySentTopic            = "money-sent"
	cardPurchaseTopic         = "card-purchase"
	cardRefundTopic           = "card-refund"
	kafkaBroker               = "broker:9092"
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

// writeEvent marshals and publishes a single event to the given topic.
// Set KAFKA_DISABLED=true to make publishing a no-op (local dev and tests
// without a broker).
func writeEvent(topic string, key string, event interface{}) error {
	if os.Getenv("KAFKA_DISABLED") == "true" {
		return nil
	}

	w := createKafkaWriter(topic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal %s event: %v", topic, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(key),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write %s event: %v", topic, err)
	}

	return nil
}

// partitionKey picks the sender account for partitioning, falling back to the
// receiver for receiver-only events (e.g. deposits).
func partitionKey(senderAccountID, receiverAccountID string) string {
	if senderAccountID == "" {
		return receiverAccountID
	}
	return senderAccountID
}

// ProduceTransactionCreatedEvent publishes a transaction created event to Kafka
func ProduceTransactionCreatedEvent(event events.TransactionCreatedEvent) error {
	return writeEvent(transactionCreatedTopic, partitionKey(event.SenderAccountID, event.ReceiverAccountID), event)
}

// ProduceTransactionCompletedEvent publishes a transaction completed event to Kafka
func ProduceTransactionCompletedEvent(event events.TransactionCompletedEvent) error {
	return writeEvent(transactionCompletedTopic, partitionKey(event.SenderAccountID, event.ReceiverAccountID), event)
}

// ProduceTransactionFailedEvent publishes a transaction failed event to Kafka
func ProduceTransactionFailedEvent(event events.TransactionFailedEvent) error {
	return writeEvent(transactionFailedTopic, partitionKey(event.SenderAccountID, event.ReceiverAccountID), event)
}

// ProduceMoneyReceivedEvent publishes a money received event to Kafka
func ProduceMoneyReceivedEvent(event events.MoneyReceivedEvent) error {
	return writeEvent(moneyReceivedTopic, event.AccountID, event)
}

// ProduceMoneySentEvent publishes a money sent event to Kafka
func ProduceMoneySentEvent(event events.MoneySentEvent) error {
	return writeEvent(moneySentTopic, event.AccountID, event)
}

// ProduceCardPurchaseEvent publishes a card purchase event to Kafka
func ProduceCardPurchaseEvent(event events.CardPurchaseEvent) error {
	return writeEvent(cardPurchaseTopic, event.AccountID, event)
}

// ProduceCardRefundEvent publishes a card refund event to Kafka
func ProduceCardRefundEvent(event events.CardRefundEvent) error {
	return writeEvent(cardRefundTopic, event.AccountID, event)
}
