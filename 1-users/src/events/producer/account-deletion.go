package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	accountDeletionRequestedTopic = "account-deletion-requested"
)

type AccountDeletionRequestedEvent struct {
	AccountId         string    `json:"accountId"`
	Email             string    `json:"email"`
	RequestedAt       time.Time `json:"requestedAt"`
	ScheduledDeletion time.Time `json:"scheduledDeletion"`
}

func ProduceAccountDeletionRequestedEvent(event AccountDeletionRequestedEvent) error {
	w := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBroker),
		Topic:        accountDeletionRequestedTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: time.Millisecond * 100,
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
			Key:   []byte(event.AccountId),
			Value: eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write messages: %v", err)
	}

	return nil
}
