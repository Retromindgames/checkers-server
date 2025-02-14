package main

import (
	"checkers-server/redisdb"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

var pid int
var redisClient *redisdb.RedisClient

func init() {
	pid = os.Getpid()
	client, err := redisdb.NewRedisClient("redis:6379")
	if err != nil {
		log.Fatalf("[Redis] Error initializing Redis client: %v", err)
	}
	redisClient = client
}

func main() {
	fmt.Printf("[Worker-%d] - Broadcasting room stats...\n", pid)

	ticker := time.NewTicker(5 * time.Second) // Adjust interval as needed
	defer ticker.Stop()

	for range ticker.C {
		aggregates, err := redisdb.GetRoomAggregates(redisClient.Client)
		if err != nil {
			log.Printf("[Worker-%d] Error fetching room aggregates: %v", pid, err)
			continue
		}

		messageBytes, err := json.Marshal(aggregates)
		if err != nil {
			log.Printf("[Worker-%d] Error marshalling message: %v", pid, err)
			continue
		}

		err = redisClient.Publish("room-info", string(messageBytes))
		if err != nil {
			log.Printf("[Worker-%d] Error publishing message: %v", pid, err)
		} else {
			fmt.Printf("[Worker-%d] Published room aggregates\n", pid)
		}
	}
}