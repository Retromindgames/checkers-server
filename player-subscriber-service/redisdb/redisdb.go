package redisdb

import (
	"checkers-server/pkg/redisdb"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisSubscriber struct {
	Client *redis.Client
}

func NewRedisSubscriber(addr string) (*RedisSubscriber, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})

	// Test connection
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	return &RedisSubscriber{Client: client}, nil
}

func (rs *RedisSubscriber) SubscribeToPlayerEvents() {
	pubsub := rs.Client.Subscribe(context.Background(), "player-events")
	defer pubsub.Close()

	// Channel to receive messages
	ch := pubsub.Channel()

	for msg := range ch {
		// Handle incoming message (event)
		fmt.Printf("Received message: %s\n", msg.Payload)

		// Parse the JSON message into a Player struct
		var player redisdb.Player
		if err := json.Unmarshal([]byte(msg.Payload), &player); err != nil {
			fmt.Printf("Error parsing message: %v\n", err)
			continue
		}

		// Save the player information to Redis
		playerKey := fmt.Sprintf("player:%s", player.ID) 
		// Store the player data as a hash in Redis
		err := rs.Client.HSet(context.Background(), playerKey, "status", player).Err()
		if err != nil {
			fmt.Printf("Error saving player to Redis: %v\n", err)
			continue
		}

	}
}