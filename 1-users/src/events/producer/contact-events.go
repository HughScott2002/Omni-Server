package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"omni/src/models/events"
	"github.com/segmentio/kafka-go"
)

const (
	contactRequestSentTopic     = "contact-request-sent"
	contactRequestAcceptedTopic = "contact-request-accepted"
	contactRequestRejectedTopic = "contact-request-rejected"
	contactBlockedTopic         = "contact-blocked"
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

// ProduceContactRequestSentEvent publishes a contact request sent event to Kafka
func ProduceContactRequestSentEvent(event events.ContactRequestSentEvent) error {
	w := createKafkaWriter(contactRequestSentTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal contact request sent event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.AddresseeID), // Key by addressee to ensure they receive in order
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write contact request sent event: %v", err)
	}

	return nil
}

// ProduceContactRequestAcceptedEvent publishes a contact request accepted event to Kafka
func ProduceContactRequestAcceptedEvent(event events.ContactRequestAcceptedEvent) error {
	w := createKafkaWriter(contactRequestAcceptedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal contact request accepted event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.RequesterID), // Key by requester to notify them
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write contact request accepted event: %v", err)
	}

	return nil
}

// ProduceContactRequestRejectedEvent publishes a contact request rejected event to Kafka
func ProduceContactRequestRejectedEvent(event events.ContactRequestRejectedEvent) error {
	w := createKafkaWriter(contactRequestRejectedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal contact request rejected event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.RequesterID), // Key by requester to notify them
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write contact request rejected event: %v", err)
	}

	return nil
}

// ProduceContactBlockedEvent publishes a contact blocked event to Kafka
func ProduceContactBlockedEvent(event events.ContactBlockedEvent) error {
	w := createKafkaWriter(contactBlockedTopic)
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal contact blocked event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Determine who to notify (the other person)
	notifyAccountID := event.AddresseeID
	if event.BlockedBy == event.AddresseeID {
		notifyAccountID = event.RequesterID
	}

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(notifyAccountID),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write contact blocked event: %v", err)
	}

	return nil
}
