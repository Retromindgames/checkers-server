package main

import (
	"checkers-server/config"
	"checkers-server/messages"
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
	config.LoadConfig()
	redisAddr := config.Cfg.Redis.Addr
	client, err := redisdb.NewRedisClient(redisAddr)
	if err != nil {
		log.Fatalf("[Redis] Error initializing Redis client: %v\n", err)
	}
	redisClient = client
}

func main() {
	fmt.Printf("[Worker-%d] - Waiting for player connections...\n", pid)
	go processPlayerOnline()
	go processPlayerOffline()
	go processPlayerUpdate()

	select {}
}

func processPlayerOnline(){
	for {
		playerData, err := redisClient.BLPop("player_online", 0) // Block
		if err != nil {
			fmt.Printf("[Worker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[Worker-%d] - Player connected: %+v\n", pid,playerData)
		handleNewPlayer(playerData)
	}
}

func processPlayerUpdate(){
	for {
		playerData, err := redisClient.BLPop("player_update", 0) // Block
		if err != nil {
			fmt.Printf("[Worker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		updatePlayer(playerData)
	}
}

func processPlayerOffline(){
	for {
		playerData, err := redisClient.BLPop("player_offline", 0) // Block
		if err != nil {
			fmt.Printf("[Worker-%d] - Error retrieving player: %d\n", pid, err)
			continue
		}
		fmt.Printf("[Worker-%d] - Player disconnected: %+v\n", pid, playerData)
		handleRemovePlayer(playerData)
	}
}

func handleRemovePlayer(player *models.Player) {
	fmt.Printf("[Worker-%d] - Removing player: %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)

	err := redisClient.RemovePlayer("player:"+player.ID, player.ID)
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to remove player: %v\n", pid, err)
		return
	}
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

	err := redisClient.AddPlayer(redisdb.GetPlayerPubSubChannel(*player), player)
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to add player: %v\n", pid, err)
		return
	}
	jsonMessage, err := messages.GenerateConnectedMessage(player)
	err = redisClient.PublishToPlayer(*player, jsonMessage)
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to publish message to player: %v\n", pid, err)
		return
	}
	fmt.Printf("[Worker-%d] - Player successfully handled and notified.\n", pid)
}

func updatePlayer(player *models.Player) {
	
	err := redisClient.AddPlayer(redisdb.GetPlayerPubSubChannel(*player), player)
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to add player: %v\n", pid, err)
		return
	}
	jsonMessage, err := messages.GenerateConnectedMessage(player)
	err = redisClient.PublishToPlayer(*player, jsonMessage)
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to publish message to player: %v\n", pid, err)
		return
	}
	fmt.Printf("[Worker-%d] - Player successfully handled and notified.\n", pid)
}