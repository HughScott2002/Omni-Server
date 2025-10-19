package db

// import (
// 	"context"
// 	"log"
// 	"os"
// 	"time"

// 	"github.com/go-redis/redis/v8"
// )

// var RedisClient *redis.Client

// func InitRedis() {
// 	redisPassword := os.Getenv("USER_REDIS_PASSWORD")
// 	if redisPassword == "" {
// 		panic("USER_REDIS_PASSWORD environment variable is not set")
// 	}

// 	RedisClient = redis.NewClient(&redis.Options{
// 		Addr:     "user-redis:6379",
// 		Password: redisPassword,
// 		DB:       0,
// 	})

// 	ctx := context.Background()
// 	_, err := RedisClient.Ping(ctx).Result()
// 	if err != nil {
// 		panic("Failed to connect to Redis: " + err.Error())
// 	}
// }
// func RedisHealthCheck() {
// 	ticker := time.NewTicker(30 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 			defer cancel()

// 			_, err := RedisClient.Ping(ctx).Result()
// 			if err != nil {
// 				log.Printf("Redis User health check failed: %v", err)
// 				// You might want to implement a retry mechanism or alert system here
// 			} else {
// 				log.Println("Redis User health check passed")
// 			}
// 		}
// 	}
// }
