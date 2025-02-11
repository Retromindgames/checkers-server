package redisdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

// RedisClient represents a wrapper around Redis client.
type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}
func NewRedisClient(addr string) (*RedisClient, error) {
	// Initialize the Redis client
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Attempt to ping Redis to check connection
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("[pkg/redisdb/cliente] - failed to connect to Redis at %s: %w", addr, err)
	}

	// Return the initialized RedisClient on success
	return &RedisClient{Client: client}, nil
}

func (rc *RedisClient) PublishPlayerEvent(playerID string, status string) error {
	event := map[string]string{
		"playerID": playerID,
		"status":   status,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[pkg/redisdb/cliente] - failed to marshal event data: %w", err)
	}

	err = rc.Client.Publish(context.Background(), "player_updates", data).Err()
	if err != nil {
		return fmt.Errorf("[pkg/redisdb/cliente] - failed to publish player event: %w", err)
	}
	return nil
}


// Subscribe listens to the Redis channel for updates.
func (rc *RedisClient) Subscribe() {
	pubsub := rc.Client.Subscribe(rc.Ctx, "player_updates")

	// Make sure to unsubscribe when done
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(rc.Ctx)
		if err != nil {
			log.Printf("[pkg/redisdb/cliente] - Failed to receive message: %v", err)
			return
		}

		// Process message
		var playerEvent map[string]string
		if err := json.Unmarshal([]byte(msg.Payload), &playerEvent); err != nil {
			log.Printf("[pkg/redisdb/cliente] - Failed to unmarshal event: %v", err)
			continue
		}

		// For example, you can print or update the player's data based on the event
		fmt.Printf("[pkg/redisdb/cliente] - Received update for player: %s, status: %s\n", playerEvent["playerID"], playerEvent["status"])
	}
}