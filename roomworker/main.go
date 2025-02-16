package main

import (
	"checkers-server/config"
	"checkers-server/messages"
	"checkers-server/models"
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
	config.LoadConfig()
	redisAddr := config.Cfg.Redis.Addr
	client, err := redisdb.NewRedisClient(redisAddr)
	if err != nil {
		log.Fatalf("[Redis] Error initializing Redis client: %v\n", err)
	}
	redisClient = client
}

func main() {
	fmt.Printf("[RoomWorker-%d] - Waiting for room messages...\n", pid)
	go processRoomCreation()
	go processRoomJoin()
	go processRoomEnding()
	select {}
}

func processRoomCreation(){
	for {
		playerData, err := redisClient.BLPop("create_room", 0) // Block
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[RoomWorker-%d] - create room!: %+v\n", pid, playerData)
		handleCreateRoom(playerData)
	}
}

func processRoomJoin(){
	for {
		playerData, err := redisClient.BLPop("join_room", 0) // Block
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[RoomWorker-%d] - processing join room!: %+v\n", pid, playerData)
		handleJoinRoom(playerData)
	}
}

//TODO!!!
func processRoomEnding(){
	for {
		playerData, err := redisClient.BLPop("leave_room", 0) // Block
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[RoomWorker-%d] - end room!: %+v\n", pid, playerData)
	}
}

func handleCreateRoom(player *models.Player) {
	fmt.Printf("[RoomWorker-%d] - Handling player (CREATE ROOM): %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)

	room := &models.Room{
		ID:        models.GenerateUUID(),
		Player1:   player,
		StartDate: time.Now(),
		Currency:  player.Currency,
		BidAmount: player.SelectedBid,
	}
	err := redisClient.AddRoom2(room)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Failed to add room to Redis: %v\n", pid, err)
		return
	}

	player.RoomID = room.ID
	player.Status = "waiting_oponente"
	redisClient.AddPlayer(player)		// This should update out player room info.
	
	messageBytes, err := messages.GenerateRoomCreatedMessage(*room)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Invalid message format: %v\n", pid, err)
		return
	}
	// Publish the validated message to Redis
	err = redisClient.PublishToPlayer(*player, string(messageBytes))
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Failed to publish message to player: %v\n", pid, err)
		return
	}
	fmt.Printf("[RoomWorker-%d] - Player successfully handled and notified, %+v\n", pid, string(messageBytes))
}


func handleJoinRoom(player *models.Player) {
	fmt.Printf("[RoomWorker-%d] - Handling player (JOIN ROOM): %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)
	rooms, err := redisClient.GetEmptyRoomsByBidAmount(player.SelectedBid)
	if err != nil {
		return
	}
	message, err := messages.GeneratePairedMessage(rooms[0].Player1, player)
	redisClient.PublishPlayerEvent(rooms[0].Player1, message)
	redisClient.PublishPlayerEvent(player, message)

}





