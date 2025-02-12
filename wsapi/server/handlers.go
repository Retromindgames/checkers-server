package server

import (
	"checkers-server/messages"
	"checkers-server/models"
	"fmt"

	"github.com/gorilla/websocket"
)

func handleMessages(player *models.Player) {
	defer player.Conn.Close()
	for {
		_, msg, err := player.Conn.ReadMessage()
		if err != nil {
			fmt.Println("Player disconnected:", player.ID)
			err = redisClient.RPush("player_offline", player)
			break
		}
		fmt.Printf("Message from %s: %s\n", player.ID, string(msg))

		// Process the received message (expecting JSON)
		message, err := messages.ParseMessage(msg, player.Conn)
		if err != nil {
			player.Conn.WriteMessage(websocket.TextMessage, []byte("Invalid message format."))
			continue
		}
		err = redisClient.RPush(message.Command , player)
	}
}
