package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func InitRedis() (*redis.Client, error) {
	redisPassword := os.Getenv("WALLET_REDIS_PASSWORD")
	redisPort := os.Getenv("WALLET_REDIS_PORT")
	if redisPassword == "" {
		panic("WALLET_REDIS_PASSWORD environment variable is not set")
	}
	initAddr := "wallet-redis:" + redisPort
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     initAddr,
		Password: redisPassword,
		DB:       0,
	})

	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	return RedisClient, err
}
func RedisHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := RedisClient.Ping(ctx).Result()
			if err != nil {
				log.Printf("Redis Wallet health check failed: %v", err)
				// You might want to implement a retry mechanism or alert system here
			} else {
				log.Println("Redis Wallet health check passed")
			}
		}
	}
}
