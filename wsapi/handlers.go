package wsapi

import (
	"checkers-server/messages"
	"checkers-server/models"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

func handleMessages(player *models.Player) {
	defer player.Conn.Close()
	for {
		_, msg, err := player.Conn.ReadMessage()
		if err != nil {
			UpdatePlayerDataFromRedis(player)
			handlePlayerDisconnect(player)
			break
		}
		fmt.Printf("Message from %s: %s\n", player.ID, string(msg))
		UpdatePlayerDataFromRedis(player)

		// Process the received message (expecting JSON), this will read the command but leave the value.
		message, err := messages.ParseMessage(msg)
		if err != nil {
			player.Conn.WriteMessage(websocket.TextMessage, []byte("Invalid message format."+err.Error()))
			continue
		}

		// Directly route to the right handler based on the command
		switch message.Command {
		case "queue":
			handleQueue(message, player)

		case "ready_queue":
			handleReadyQueue(message, player)
		}
	}
}

func handleQueue(msg *messages.Message[json.RawMessage], player *models.Player) {
	if player.UpdatePlayerStatus(models.StatusInQueue) != nil {
		player.Conn.WriteMessage(websocket.TextMessage, []byte("Invalid status transition to 'queue'"))
		return
	}
	// update the player bet and push it to Redis
	var betValue float64
	err := json.Unmarshal(msg.Value, &betValue)
	player.SelectedBet = betValue
	player.Status = models.StatusInQueue
	// Pushing the player to the "queue" Redis list
	err = redisClient.RPush("queue", player) // Assuming "queue" is the appropriate Redis list
	if err != nil {
		fmt.Printf("Error pushing player to Redis queue: %v\n", err)
		player.Conn.WriteMessage(websocket.TextMessage, []byte("Error adding player to queue"))
		return
	}
	// we update out player status.
	redisClient.AddPlayer(player)
	// send a confirmation message back to the player
	m, err := messages.GenerateQueueConfirmationMessage(true)
	if err != nil {
		fmt.Println("Error generating queue confirmation:", err)
		player.Conn.WriteMessage(websocket.TextMessage, []byte("Error generating confirmation"))
		return
	}
	player.Conn.WriteMessage(websocket.TextMessage, m)
	// Pushing the player to the "ready" Redis list, as to be processed by the room worker.
	err = redisClient.RPush("ready_queue", player) // Assuming "queue" is the appropriate Redis list
	if err != nil {
		fmt.Printf("Error pushing player to Redis queue: %v\n", err)
		player.Conn.WriteMessage(websocket.TextMessage, []byte("Error adding player to queue"))
		return
	}
}

func handleReadyQueue(msg *messages.Message[json.RawMessage], player *models.Player) {
	if player.UpdatePlayerStatus(models.StatusAwaitingOponenteReady) != nil {
		player.Conn.WriteMessage(websocket.TextMessage, []byte("Invalid status transition to 'ready_queue'"))
		return
	}
	player.Conn.WriteMessage(websocket.TextMessage, []byte("processing 'ready_queue'"))
	// update the player status
	player.Status = models.StatusAwaitingOponenteReady
	// we update out player status.
	redisClient.AddPlayer(player)

}

func handlePlayerDisconnect(player *models.Player) {
	fmt.Println("Player disconnected:", player.ID)
	playersMutex.Lock()
	delete(players, player.ID)
	playersMutex.Unlock()
	// Unsubscribe from Redis channels
	unsubscribeFromPlayerChannel(player)
	// Notify worker of disconnection
	redisClient.RPush("player_offline", player)
}

func UpdatePlayerDataFromRedis(player *models.Player) {
	playerData, err := redisClient.GetPlayer(string(player.ID))
	if err != nil {
		fmt.Printf("[Handlers] - Failed to update player data from redis!: Player: %s", player.ID)
		return
	}
	player.Currency = playerData.Currency
	player.CurrencyAmount = playerData.CurrencyAmount
	player.Status = playerData.Status
	player.SelectedBet = playerData.SelectedBet
	player.RoomID = playerData.RoomID
}
