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
	accountCreatedTopic = "account-created"
	kafkaBroker         = "broker:9092"
	// kafkaPartition      = 0
)

// func ProduceAccountCreatedEvent(event events.AccountCreatedEvent) error {
// 	conn, err := kafka.DialLeader(context.Background(), "tcp", kafkaBroker, accountCreatedTopic, kafkaPartition)
// 	if err != nil {
// 		return fmt.Errorf("failed to dail leader: %v", err)
// 	}
// 	// defer log.Println(conn.Close())
// 	defer conn.Close()

// 	eventJSON, err := json.Marshal(event)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal event: %v", err)
// 	}
// 	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
// 	_, err = conn.WriteMessages(
// 		kafka.Message{Value: eventJSON},
// 	)
// 	if err != nil {
// 		return fmt.Errorf("failed to write message: %v", err)
// 	}

// 	return nil

// }
func ProduceAccountCreatedEvent(event events.AccountCreatedEvent) error {
	// Create a writer instead of direct connection to handle partitioning
	w := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBroker),
		Topic:        accountCreatedTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: time.Millisecond * 100,
		// Add retry logic
		WriteTimeout: time.Second * 10,
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
	}
	defer w.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.AccountId), // Use AccountId as key for consistent partitioning
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write messages: %v", err)
	}

	return nil
}
