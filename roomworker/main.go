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
	fmt.Printf("[Worker-%d] - Waiting for room messages...\n", pid)
	go processRoomCreation()
	go processRoomJoin()
	go processRoomEnding()
	select {}
}

func processRoomCreation(){
	for {
		playerData, err := redisClient.BLPop("create_room", 0) // Block
		if err != nil {
			fmt.Printf("[Worker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[Worker-%d] - create room!: %+v\n", pid, playerData)
		handleCreateRoom(playerData)
	}
}

// TODO
func processRoomJoin(){
	for {
		playerData, err := redisClient.BLPop("join_room", 0) // Block
		if err != nil {
			fmt.Printf("[Worker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[Worker-%d] - processing join room!: %+v\n", pid, playerData)
		handleJoinRoom(playerData)
	}
}

// TODO
func processRoomEnding(){
	for {
		playerData, err := redisClient.BLPop("end_room", 0) // Block
		if err != nil {
			fmt.Printf("[Worker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[Worker-%d] - end room!: %+v\n", pid,playerData)
	}
}

func handleCreateRoom(player *models.Player) {
	fmt.Printf("[Worker-%d] - Handling player (CREATE ROOM): %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)

	room := &models.Room{
		ID:    models.GenerateUUID(),
		Player1:   player,
		StartDate: time.Now(),
		Currency:  player.Currency,
		BidAmount: player.SelectedBid,
	}
	err := redisClient.AddRoom("room:"+room.ID, room)
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to add room to Redis: %v\n", pid, err)
		return
	}
	roomValue := models.RoomValue{
		ID:          room.ID,
		Player:      room.Player1.Name,
		Currency:    room.Currency,
		SelectedBid: room.BidAmount,
	}
	messageJson := &models.CreateRoomMessage{
		Command: "room_created",
		Value: roomValue,
	}
	messageBytes, err := json.Marshal(messageJson)
	if err != nil {
		fmt.Printf("[Worker-%d] - Error marshalling message: %v\n", pid, err)
		return
	}
	err = redisClient.PublishToPlayer(*player, string(messageBytes))
	if err != nil {
		fmt.Printf("[Worker-%d] - Failed to publish message to player: %v\n", pid, err)
		return
	}
	fmt.Printf("[Worker-%d] - Player successfully handled and notified, Room ID: %s\n", pid, room.ID)
}

func handleJoinRoom(player *models.Player) {
	fmt.Printf("[Worker-%d] - Handling player (JOIN ROOM): %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)
	rooms, err := redisClient.GetEmptyRoomsByBidAmount(player.SelectedBid)
	if err != nil {
		return
	}
	message, err := messages.GeneratePairedMessage(rooms[0].Player1, player)
	redisClient.PublishPlayerEvent(rooms[0].Player1, message)
	redisClient.PublishPlayerEvent(player, message)
}





