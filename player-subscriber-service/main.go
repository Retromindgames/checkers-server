package main

import (
	"checkers-server/player-subscriber-service/redisdb"
	"fmt"
	"log"
)

func main() {
	// Create a new Redis subscriber instance
	redisSubscriber, err := redisdb.NewRedisSubscriber("localhost:6379")
	if err != nil {
		log.Fatalf("Failed to initialize Redis subscriber: %v", err)
	} else {
		fmt.Println("[Player-Subscriber-Service] - Started")
	}

	// "player-events" channel
	redisSubscriber.SubscribeToPlayerEvents()
}