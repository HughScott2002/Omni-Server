package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"example.com/m/v2/src/models/events"
	"github.com/segmentio/kafka-go"
)

const (
	virtualCardCreatedTopic          = "virtual-card-created"
	virtualCardBlockedTopic          = "virtual-card-blocked"
	virtualCardToppedUpTopic         = "virtual-card-topped-up"
	physicalCardRequestedTopic       = "physical-card-requested"
	virtualCardDeletedTopic          = "virtual-card-deleted"
	kafkaBroker                      = "broker:9092"
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

// ProduceVirtualCardCreatedEvent publishes a virtual card created event to Kafka
func ProduceVirtualCardCreatedEvent(event events.VirtualCardCreatedEvent) error {
	w := createKafkaWriter(virtualCardCreatedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal virtual card created event: %v", err)
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
		return fmt.Errorf("failed to write virtual card created event: %v", err)
	}

	return nil
}

// ProduceVirtualCardBlockedEvent publishes a virtual card blocked event to Kafka
func ProduceVirtualCardBlockedEvent(event events.VirtualCardBlockedEvent) error {
	w := createKafkaWriter(virtualCardBlockedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal virtual card blocked event: %v", err)
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
		return fmt.Errorf("failed to write virtual card blocked event: %v", err)
	}

	return nil
}

// ProduceVirtualCardToppedUpEvent publishes a virtual card topped up event to Kafka
func ProduceVirtualCardToppedUpEvent(event events.VirtualCardToppedUpEvent) error {
	w := createKafkaWriter(virtualCardToppedUpTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal virtual card topped up event: %v", err)
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
		return fmt.Errorf("failed to write virtual card topped up event: %v", err)
	}

	return nil
}

// ProducePhysicalCardRequestedEvent publishes a physical card requested event to Kafka
func ProducePhysicalCardRequestedEvent(event events.PhysicalCardRequestedEvent) error {
	w := createKafkaWriter(physicalCardRequestedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal physical card requested event: %v", err)
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
		return fmt.Errorf("failed to write physical card requested event: %v", err)
	}

	return nil
}

// ProduceVirtualCardDeletedEvent publishes a virtual card deleted event to Kafka
func ProduceVirtualCardDeletedEvent(event events.VirtualCardDeletedEvent) error {
	w := createKafkaWriter(virtualCardDeletedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal virtual card deleted event: %v", err)
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
		return fmt.Errorf("failed to write virtual card deleted event: %v", err)
	}

	return nil
}
