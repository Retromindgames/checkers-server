package main

import (
	"checkers-server/models"
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
	fmt.Println("[Worker] - Waiting for player connections...")
	go processPlayerOnline()
	go processPlayerOffline()

	select {}
}


func processPlayerOnline(){
	for {
		playerData, err := redisClient.BLPop("player_online", 0) // Block until a player appears
		if err != nil {
			fmt.Println("[Worker] - Error retrieving player:", err)
			continue
		}

		fmt.Printf("[Worker] - Player connected: %+v\n", playerData)
		// Process the player (e.g., matchmaking, authentication, etc.)
		handleNewPlayer(playerData)
	}
}

func processPlayerOffline(){
	for {
		playerData, err := redisClient.BLPop("player_offline", 0) // Block until a player appears
		if err != nil {
			fmt.Println("[Worker] - Error retrieving player:", err)
			continue
		}

		fmt.Printf("[Worker] - Player disconnected: %+v\n", playerData)
		// Process the player (e.g., matchmaking, authentication, etc.)
		handleRemovePlayer(playerData)
	}
}
func handleRemovePlayer(player *models.Player) {
	// Print the worker log with PID
	fmt.Printf("[Worker-%d] - Removing player: %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)

	// Remove the player from Redis
	err := redisClient.RemovePlayer("player:"+player.ID, player.ID)
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to remove player: %v\n", pid, err)
		return
	}

	// Publish player-offline event to Redis
	err = redisClient.PublishToPlayer(*player, "player-offline")
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to publish player-offline event: %v\n", pid, err)
		return
	}

	fmt.Printf("[Worker-%d] - Player successfully removed and notified.\n", pid)
}

func handleNewPlayer(player *models.Player) {
	fmt.Printf("[Worker-%d] - Handling player: %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)

	err := redisClient.AddPlayer("player:"+player.ID, player)
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to add player: %v\n", pid, err)
		return
	}
	err = redisClient.PublishToPlayer(*player, "Welcome to the game!")
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to publish message to player: %v\n", pid, err)
		return
	}

	fmt.Printf("[Worker-%d] - Player successfully handled and notified.\n", pid)
}