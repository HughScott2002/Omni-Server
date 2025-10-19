package events

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	AccountCreatedTopic           = "account-created"
	AccountDeletionRequestedTopic = "account-deletion-requested"
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
		log.Printf("Topics listing error, %s", err.Error())
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
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil && err.(kafka.Error) != kafka.TopicAlreadyExists {
		return fmt.Errorf("failed to create topic: %v", err)
	}

	return nil
}

func list_topics() error {
	log.Printf("Checking the Broker Connection...")
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
	for k := range m {
		fmt.Println(k)
	}
	return nil
}
