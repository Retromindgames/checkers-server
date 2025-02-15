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
			handlePlayerDisconnect(player)
			break
		}
		fmt.Printf("Message from %s: %s\n", player.ID, string(msg))

		// Process the received message (expecting JSON)
		message, err := messages.ParseMessage(msg)
		if err != nil {
			player.Conn.WriteMessage(websocket.TextMessage, []byte("Invalid message format." + err.Error()))
			continue
		}
		// We update the player bidValue. This is so our RPUSH seends the player setAmount.
		if(message.Command == "create_room" || message.Command == "join_room") {
			var bidValue float64
			err := json.Unmarshal(message.Value, &bidValue)
			if err != nil {
				fmt.Println("Error unmarshaling bid value:", err)
				return
			}
			player.SelectedBid = bidValue
		}
		// Now we push the command to our worker, he will determine if we can start a match
		err = redisClient.RPush(message.Command , player)
	}
}


func handlePlayerDisconnect(player *models.Player) {
	fmt.Println("Player disconnected:", player.ID)
	// Unsubscribe from Redis channels
	unsubscribeFromPlayerChannel(player)
	unsubscribeFromBroadcastChannel(player)
	// Notify worker of disconnection
	redisClient.RPush("player_offline", player)
}