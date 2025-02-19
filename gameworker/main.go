package main

import (
	"checkers-server/config"
	"checkers-server/redisdb"
	"fmt"
	"log"
	"os"
)

var pid int
var redisClient *redisdb.RedisClient
var name = "GameWorker"

func init() {
	pid = os.Getpid()
	config.LoadConfig()
	redisAddr := config.Cfg.Redis.Addr
	client, err := redisdb.NewRedisClient(redisAddr)
	if err != nil {
		log.Fatalf("[Redis] Error initializing Redis client: %v\n", err)
	}
	redisClient = client
}

func main() {
	fmt.Printf("[%s-%d] - Waiting for room messages...\n", name, pid)
	go processGameCreation()
	select {}
}

func processGameCreation() {
	for {
		roomData, err := redisClient.BLPopGeneric("create_game", 0) // Block
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - (Process Game Creation) -  Error retrieving room data:%v\n", pid, err)
			continue
		}
		fmt.Printf("[RoomWorker-%d] - (Process Game Creation)  - create room!: %+v\n", pid, roomData)
		//TODO: Generate game!

	}
}
