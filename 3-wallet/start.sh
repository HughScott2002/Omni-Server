#!/bin/sh
# start.sh

# Wait for Kafka to be ready
echo "Waiting for Kafka to be ready..."
while ! kafka-topics.sh --bootstrap-server broker:9092 --list > /dev/null 2>&1; do
    echo "Kafka is not ready - sleeping"
    sleep 2
done

# Create the topic if it doesn't exist
kafka-topics.sh --bootstrap-server broker:9092 --create --if-not-exists \
    --topic account-created --partitions 3 --replication-factor 1

# Start the wallet service
exec ./main