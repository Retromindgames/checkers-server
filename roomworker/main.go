package main

import (
	"checkers-server/redisdb"
	"fmt"
	"log"
	"os"
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
	fmt.Printf("[Worker-%d] - Waiting for player connections...\n", pid)

	go processRoomCreation()
	go processRoomJoin()
	go processRoomEnding()

	select {}
}


func processRoomCreation(){
	for {
		playerData, err := redisClient.BLPop("room_creation", 0) // Block
		if err != nil {
			fmt.Println("[Worker] - Error retrieving player:", err)
			continue
		}
		fmt.Printf("[Worker] - Player connected: %+v\n", playerData)
	
	}
}

// TODO
func processRoomJoin(){
	for {
		playerData, err := redisClient.BLPop("room_join", 0) // Block 
		if err != nil {
			fmt.Println("[Worker] - Error retrieving player:", err)
			continue
		}
		fmt.Printf("[Worker] - Player disconnected: %+v\n", playerData)
	}
}

// TODO
func processRoomEnding(){
	for {
		playerData, err := redisClient.BLPop("room_end", 0) // Block 
		if err != nil {
			fmt.Println("[Worker] - Error retrieving player:", err)
			continue
		}
		fmt.Printf("[Worker] - Player disconnected: %+v\n", playerData)

	}
}



