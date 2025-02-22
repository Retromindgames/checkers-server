package main

import (
	"checkers-server/config"
	"checkers-server/messages"
	"checkers-server/models"
	"checkers-server/redisdb"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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

	go processReadyQueue()
	go processRoomEnding()
	go processQueue()
	select {}
}

func processRoomCreation() {
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

func processRoomJoin() {
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

func processQueue() {
	for {
		player1, err := redisClient.BLPop("queue", 0)
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving player 1: %v\n", pid, err)
			continue
		}
		fmt.Printf("[RoomWorker-%d] - retrieved player 1: %v\n", pid, err)

		// Try fetching the second player with a timeout
		player2, err := redisClient.BLPop("queue", 5)
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - No second player found, re-queueing player 1.\n", pid)
			// before we add the player back we have to get it from the players and check if its still in queue.
			player1, err = redisClient.GetPlayer(player1.ID)
			// Re-add player1 back to the queue
			if player1.Status == models.StatusInQueue{
				redisClient.RPush("queue", player1)
			}
			continue
		}
		fmt.Printf("[RoomWorker-%d] - retrieved player 2: %v\n", pid, err)

		// Process both players
		fmt.Printf("[RoomWorker-%d] - Pairing players: %s and %s\n", pid, player1, player2)
		handleQueuePaired(player1, player2)
	}
}

func processReadyQueue() {
	for {
		playerData, err := redisClient.BLPop("ready_queue", 0) // Block
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[RoomWorker-%d] - processing ready room!: %+v\n", pid, playerData)
		// Aqui ou damos handle do ready queue ou handle do unreadyqueue
		if playerData.Status == models.StatusInRoomReady {
			handleReadyQueue(playerData)
		}
		if playerData.Status == models.StatusInRoom {
			handleUnReadyQueue(playerData)
		}
	}
}

func processRoomEnding() {
	for {
		playerData, err := redisClient.BLPop("leave_room", 0) 
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		fmt.Printf("[RoomWorker-%d] - Processing the end of room: %+v\n", pid, playerData)
		room, err := redisClient.GetRoomByID(playerData.RoomID)
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving room:%v\n", pid, err)
			continue
		}
		player2ID, err := room.GetOpponentPlayerID(playerData.ID)
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving opponent id:%v\n", pid, err)
			continue
		}
		player2, err := redisClient.GetPlayer(player2ID)
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error retrieving opponent player:%v\n", pid, err)
			continue
		}
		msg, err := messages.NewMessage("opponent_left_room", true)
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error generating message:%v\n", pid, err)
			continue
		}
		redisClient.PublishToPlayer(*player2, string(msg))
		// Reset both player data.
		playerData.RoomID = ""
		playerData.Status = models.StatusOnline
		player2.RoomID = ""
		player2.Status = models.StatusInQueue
		redisClient.AddPlayer(playerData)
		redisClient.AddPlayer(player2)
		
		// Pushing the player to the "queue" Redis list
		err = redisClient.RPush("queue", player2) 
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Placing player back on queue:%v\n", pid, err)
			return
		}

		err = redisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(room.ID))
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error removing room: %v\n", pid, err)
			continue
		}
		redisClient.DecrementRoomAggregate(playerData.SelectedBet)
		fmt.Printf("[RoomWorker-%d] - End of room ending: %v\n", pid, err)
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
		BetValue:  player.SelectedBet,
	}
	err := redisClient.AddRoom(room)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Failed to add room to Redis: %v\n", pid, err)
		return
	}

	player.RoomID = room.ID
	player.Status = "waiting_oponente"
	redisClient.AddPlayer(player) // This should update out player room info.

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

func handleQueuePaired(player1, player2 *models.Player) {
	fmt.Printf("[RoomWorker-%d] - Handling player1 (CREATE ROOM): %s (Session: %s, Currency: %s)\n",
		pid, player1.ID, player1.SessionID, player1.Currency)
	fmt.Printf("[RoomWorker-%d] - Handling player2 (CREATE ROOM): %s (Session: %s, Currency: %s)\n",
		pid, player2.ID, player2.SessionID, player2.Currency)

	room := &models.Room{
		ID:        models.GenerateUUID(),
		Player1:   player1,
		Player2:   player2,
		StartDate: time.Now(),
		Currency:  player1.Currency,
		BetValue:  player1.SelectedBet,
	}
	err := redisClient.AddRoom(room)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Failed to add room to Redis: %v\n", pid, err)
		return
	}
	player1.RoomID = room.ID
	player2.RoomID = room.ID
	// TODO: review this 
	player1.Status = models.StatusInRoom
	player2.Status = models.StatusInRoom
	redisClient.AddPlayer(player1)
	redisClient.AddPlayer(player2)
	// Now we set the player colors. TODO: This is ugly, this should be changed.
	colorp1 := rand.Intn(2)
	colorp2 := 1
	if colorp1 == 1 {
		room.CurrentPlayerID = player1.ID
		colorp2 = 0
	} else {
		room.CurrentPlayerID = player2.ID
	}
	// we save the room again since we made changed to it.
	err = redisClient.AddRoom(room)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Failed to add room to Redis: %v\n", pid, err)
		return
	}
	message1, err := messages.GeneratePairedMessage(room.Player1, room.Player2, room.ID, colorp1)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handling paired message1 for p1: %s\n", pid, err)
		return
	}
	message2, err2 := messages.GeneratePairedMessage(room.Player2, room.Player1, room.ID, colorp2)
	if err2 != nil {
		fmt.Printf("[RoomWorker-%d] - Error handling paired message1 for p2:%s\n", pid, err2)
		return
	}
	fmt.Printf("[RoomWorker-%d] - Handling player (JOIN ROOM) - Message1 for player1: %s\n", pid, message1)
	fmt.Printf("[RoomWorker-%d] - Handling player (JOIN ROOM) - Message2 for player2: %s\n", pid, message2)
	// Publish the validated message to Redis
	err = redisClient.PublishToPlayer(*player1, string(message1))
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Failed to publish message1 to player1: %v\n", pid, err)
		return
	}
	err = redisClient.PublishToPlayer(*player2, string(message2))
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Failed to publish message2 to player2: %v\n", pid, err)
		return
	}
	fmt.Printf("[RoomWorker-%d] - Player successfully handled and notified, of room pairing.\n", pid)
}

func handleReadyQueue(player *models.Player) {
	fmt.Printf("[RoomWorker-%d] - Handling player (READY QUEUE): %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)

	proom, err := redisClient.GetRoomByID(player.RoomID)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleReadyQueue getting player room:%s\n", pid, err)
		return
	}

	player2ID, err := proom.GetOpponentPlayerID(player.ID)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleReadyQueue getting player opponent ID:%s\n", pid, err)
		return
	}

	player2, err := redisClient.GetPlayer(player2ID)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleReadyQueue getting opponent player:%s\n", pid, err)
		return
	}

	// We will always notify the opponent the we are ready.
	msg, err := messages.GenerateOpponentReadyMessage(true)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleReadyQueue getting GenerateOpponentReadyMessage(true) for opponent:%s\n", pid, err)
		return
	}
	redisClient.PublishPlayerEvent(player2, string(msg))

	// now we tell our player that is ready if the opponent is ready or not.
	if player2.Status != models.StatusInRoomReady {
		fmt.Printf("[RoomWorker-%d] - handleReadyQueue Opponent aint ready yet!:%s\n", pid, err)
		msg, err := messages.GenerateOpponentReadyMessage(false)
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error handleReadyQueue getting GenerateOpponentReadyMessage(false) for player:%s\n", pid, err)
		}
		redisClient.PublishPlayerEvent(player, string(msg))
		return
	}

	// Now! If both players are ready...!!
	// We are ready to start a match!!
	roomdata, err := json.Marshal(proom)
	err = redisClient.RPushGeneric("create_game", roomdata)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleReadyQueue Creating Game RPushGeneric:%s\n", pid, err)
	}
}

func handleUnReadyQueue(player *models.Player) {
	fmt.Printf("[RoomWorker-%d] - Handling player (UN-READY QUEUE): %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)

	proom, err := redisClient.GetRoomByID(player.RoomID)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleUnReadyQueue getting player room:%s\n", pid, err)
		return
	}

	player2ID, err := proom.GetOpponentPlayerID(player.ID)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleUnReadyQueue getting player opponent ID:%s\n", pid, err)
		return
	}

	player2, err := redisClient.GetPlayer(player2ID)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleUnReadyQueue getting opponent player:%s\n", pid, err)
		return
	}

	// We will always notify the opponent the we are no longer ready.
	msg, err := messages.GenerateOpponentReadyMessage(false)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handleUnReadyQueue getting GenerateOpponentReadyMessage(false) for opponent:%s\n", pid, err)
		return
	}
	redisClient.PublishPlayerEvent(player2, string(msg))

	// now we tell our player that is ready if the opponent is ready or not.
	if player2.Status != models.StatusInRoomReady {
		fmt.Printf("[RoomWorker-%d] - handleUnReadyQueue Opponent aint ready yet!:%s\n", pid, err)
		msg, err := messages.GenerateOpponentReadyMessage(false)
		if err != nil {
			fmt.Printf("[RoomWorker-%d] - Error handleUnReadyQueue getting GenerateOpponentReadyMessage(false) for player:%s\n", pid, err)
		}
		redisClient.PublishPlayerEvent(player, string(msg))
		return
	}
}

func handleJoinRoom(player *models.Player) {
	fmt.Printf("[RoomWorker-%d] - Handling player (JOIN ROOM): %s (Session: %s, Currency: %s)\n",
		pid, player.ID, player.SessionID, player.Currency)
	rooms, err := redisClient.GetEmptyRoomsByBetValue(player.SelectedBet)
	if err != nil {
		return
	}
	colorp1 := rand.Intn(2)
	colorp2 := 1
	if colorp1 == 1 {
		colorp2 = 0
	}
	message, err := messages.GeneratePairedMessage(rooms[0].Player1, player, rooms[0].ID, colorp1)
	if err != nil {
		fmt.Printf("[RoomWorker-%d] - Error handling paired message for p1: %s\n", pid, err)
		return
	}
	message2, err2 := messages.GeneratePairedMessage(player, rooms[0].Player1, rooms[0].ID, colorp2)
	if err2 != nil {
		fmt.Printf("[RoomWorker-%d] - Error handling paired message for p2:%s\n", pid, err2)
		return
	}
	fmt.Printf("[RoomWorker-%d] - Handling player (JOIN ROOM) - Message for player1: %s\n", pid, message)
	fmt.Printf("[RoomWorker-%d] - Handling player (JOIN ROOM) - Message for player2: %s\n", pid, message2)

	redisClient.PublishPlayerEvent(rooms[0].Player1, string(message))
	redisClient.PublishPlayerEvent(player, string(message2))
}
