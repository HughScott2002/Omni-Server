package producer

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestProducer() {
	// to produce messages
	topic := "my-topic"
	partition := 0

	conn, err := kafka.DialLeader(context.Background(), "tcp", "broker:9092", topic, partition)
	if err != nil {
		slog.Error("failed to dial leader", "error", err)
		os.Exit(1)
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.WriteMessages(
		kafka.Message{Value: []byte("one!")},
		kafka.Message{Value: []byte("two!")},
		kafka.Message{Value: []byte("three!")},
	)
	if err != nil {
		slog.Error("failed to write messages", "error", err)
		os.Exit(1)
	}

	if err := conn.Close(); err != nil {
		slog.Error("failed to close writer", "error", err)
		os.Exit(1)
	}
}
