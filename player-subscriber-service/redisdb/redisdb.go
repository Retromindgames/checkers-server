package redisdb

import (
	"context"
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

		// Optionally, you can parse the event and take action based on the message
		// Example: {"playerID": "player1", "event": "connected"}
	}
}

