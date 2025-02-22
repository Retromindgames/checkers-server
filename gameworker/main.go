package main

import (
	"checkers-server/config"
	"checkers-server/messages"
	"checkers-server/models"
	"checkers-server/redisdb"
	"encoding/json"
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
		log.Fatalf("[%s-Redis] Error initializing Redis client: %v\n", name, err)
	}
	redisClient = client
}

func main() {
	fmt.Printf("[%s-%d] - Waiting for Game messages...\n", name, pid)
	go processGameCreation()
	select {}
}

func processGameCreation() {
	for {
		roomData, err := redisClient.BLPopGeneric("create_game", 0) // Block
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Creation) - Error retrieving room data: %v\n", name, pid, err)
			continue
		}

		if len(roomData) < 2 {
			fmt.Printf("[%s-%d] - (Process Game Creation) - Unexpected BLPop result: %+v\n", name, pid, roomData)
			continue
		}
		fmt.Printf("[%s-%d] - (Process Game Creation) - create game!: %+v\n", name, pid, roomData)
		
		var room models.Room 
		err = json.Unmarshal([]byte(roomData[1]), &room) // Extract second element
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Creation) - JSON Unmarshal Error: %v\n", name, pid, err)
			continue
		}
		
		// TODO: Generate game!
		game := room.NewGame()
		err = redisClient.AddGame(game)
		msg, err := messages.GenerateGameStartMessage(*game)
		fmt.Printf("[%s-%d] - (Process Game Creation) - Message to publish: %v\n", name, pid, msg)
		redisClient.PublishToGamePlayer(game.Players[0], string(msg))
		redisClient.PublishToGamePlayer(game.Players[1], string(msg))
	}
}
