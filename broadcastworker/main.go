package main

import (
	"checkers-server/redisdb"
	"fmt"
	"log"
	"os"
	"time"
)

var pid int
var redisClient *redisdb.RedisClient

func init() {
	pid = os.Getpid()
	client, err := redisdb.NewRedisClient("localhost:6379")
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
		message := fmt.Sprintf("Room stats from Worker-%d", pid)
		err := redisClient.Publish("room-info", message)
		if err != nil {
			log.Printf("[Worker-%d] Error publishing message: %v", pid, err)
		} else {
			fmt.Printf("[Worker-%d] Published: %s\n", pid, message)
		}
	}
}
