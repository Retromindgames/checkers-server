package main

import (
	"checkers-server/config"
	"checkers-server/messages"
	"checkers-server/redisdb"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

var pid int
var name = "Broadcaster"
var redisClient *redisdb.RedisClient

func init() {
	config.LoadConfig()
	pid = os.Getpid()
	redisAddr := config.Cfg.Redis.Addr
	client, err := redisdb.NewRedisClient(redisAddr)
	if err != nil {
		log.Fatalf("[%d][Redis] Error initializing Redis client: %v", name, err)
	}
	redisClient = client
}

func main() {
	fmt.Printf("[Worker-%d] - Broadcasting room stats...\n", pid)

	ticker := time.NewTicker(5 * time.Second) // Adjust interval as needed
	defer ticker.Stop()

	for range ticker.C {
		aggregates, err := redisClient.GetRoomAggregates()
		if err != nil {
			log.Printf("[Worker-%d] Error fetching room aggregates: %v\n", pid, err)
			continue
		}
		aggregatesBytes, err := json.Marshal(aggregates)
		if err != nil {
			log.Printf("[Worker-%d] Error marshalling message: %v\n", pid, err)
			continue
		}
		// Wrap in Message struct
		message := messages.Message{
			Command: "game-info",
			Value:   aggregatesBytes,
		}
		// Marshal final message
		messageBytes, err := json.Marshal(message)
		if err != nil {
			log.Printf("[Worker-%d] Error marshalling message: %v\n", pid, err)
			continue
		}
		err = redisClient.Publish("room-info", string(messageBytes))
		if err != nil {
			log.Printf("[Worker-%d] Error publishing message: %v\n", pid, err)
		} else {
			fmt.Printf("[Worker-%d] Published room aggregates\n", pid)
		}
	}
}
