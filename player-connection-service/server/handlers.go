package server

import (
	"checkers-server/pkg/redisdb"
	"fmt"
)


func handleMessages(player *redisdb.Player) {
	defer player.Conn.Close()

	// Start listening for messages from the player
	for {
		_, msg, err := player.Conn.ReadMessage()
		if err != nil {
			// Player disconnected or there was an error reading the message
			fmt.Println("Player disconnected:", player.ID)
			redisClient.PublishPlayerEvent(player, "disconnected")
		}
		// Print received message
		fmt.Println("Received message:", string(msg))

		// TODO: Handle the message further if needed
	}
}