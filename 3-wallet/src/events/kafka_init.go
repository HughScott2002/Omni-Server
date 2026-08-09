package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	AccountCreatedTopic           = "account-created"
	AccountDeletionRequestedTopic = "account-deletion-requested"
	KYCApprovedTopic              = "kyc-approved"
	ConsumerGroup                 = "wallet-service"
)

func KafkaInit(ctx context.Context) bool {
	// First ensure topic exists
	for i := 0; i < 5; i++ {
		if err := ensureTopic(ctx); err == nil {
			break
		} else if err := ensureTopic(ctx); err != nil && i == 4 {
			return false
		}
		time.Sleep(time.Second * 2)
	}
	// Then list the topics
	err := list_topics()
	if err != nil {
		slog.Error("Topics listing error", "error", err.Error())
		// if you can't list the topics then send back false
		return false
	}

	return true
}

func ensureTopic(ctx context.Context) error {
	conn, err := kafka.DialContext(ctx, "tcp", "broker:9092")
	if err != nil {
		return fmt.Errorf("failed to dial: %v", err)
	}
	defer conn.Close()

	// Create topic with 3 partitions
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             AccountCreatedTopic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
		{
			Topic:             AccountDeletionRequestedTopic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
		{
			Topic:             KYCApprovedTopic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil && err.(kafka.Error) != kafka.TopicAlreadyExists {
		return fmt.Errorf("failed to create topic: %v", err)
	}

	return nil
}

func list_topics() error {
	slog.Info("Checking the Broker Connection...")
	conn, err := kafka.Dial("tcp", "broker:9092")
	if err != nil {
		return fmt.Errorf("error in getting connected to broker")
		// panic("Error in getting connected to broker")
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return err
	}

	m := map[string]struct{}{}

	for _, p := range partitions {
		m[p.Topic] = struct{}{}
	}
	topics := make([]string, 0, len(m))
	for k := range m {
		topics = append(topics, k)
	}
	slog.Debug("Kafka topics present", "topics", topics)
	return nil
}
