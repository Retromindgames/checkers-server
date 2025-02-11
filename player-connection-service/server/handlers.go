package server

import (
	"fmt"
)

// handleMessages listens for messages from a player.
func handleMessages(player *Player) {
	defer player.Conn.Close()

	for {
		_, msg, err := player.Conn.ReadMessage()
		if err != nil {
			fmt.Println("Player disconnected:", player.ID)
			return
		}

		fmt.Println("Received message:", string(msg))

		// TODO: Publish event to Redis
	}
}
