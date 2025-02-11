package main

import (
	"checkers-server/player-subscriber-service/redisdb"
	"log"
)

func main() {
	// Create a new Redis subscriber instance
	redisSubscriber, err := redisdb.NewRedisSubscriber("localhost:6379")
	if err != nil {
		log.Fatalf("Failed to initialize Redis subscriber: %v", err)
	}

	// Start subscribing to the "player-events" channel
	redisSubscriber.SubscribeToPlayerEvents()
}