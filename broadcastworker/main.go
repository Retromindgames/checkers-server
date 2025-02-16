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
		log.Fatalf("[%d][BroadcastWorker-%d][Redis] Error initializing Redis client: %v", name, err)
	}
	redisClient = client
}

func main() {
	fmt.Printf("[BroadcastWorker-%d] - Broadcasting room stats...\n", pid)

	ticker := time.NewTicker(5 * time.Second) // Adjust interval as needed
	defer ticker.Stop()

	for range ticker.C {
		aggregates, err := redisClient.GetRoomAggregates()
		if err != nil {
			log.Printf("[BroadcastWorker-%d] Error fetching room aggregates: %v\n", pid, err)
			continue
		}
		aggregatesBytes, err := json.Marshal(aggregates)
		if err != nil {
			log.Printf("[BroadcastWorker-%d] Error marshalling message: %v\n", pid, err)
			continue
		}
		// Wrap in Message struct
		message := messages.Message{
			Command: "game-info",
			Value:   json.RawMessage(aggregatesBytes),
		}
		// Marshal final message
		messageBytes, err := json.Marshal(message)
		if err != nil {
			log.Printf("[BroadcastWorker-%d] Error marshalling message: %v\n", pid, err)
			continue
		}

		err = redisClient.Publish("room_info", messageBytes)
		if err != nil {
			log.Printf("[BroadcastWorker-%d] Error publishing message: %v\n", pid, err)
		} else {
			fmt.Printf("[BroadcastWorker-%d] Published room aggregates\n", pid)
		}
	}
}
