package broadcast

import (
	"checkers-server/core"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func PlayersInQueue() {
	ticker := time.NewTicker(5 * time.Second) 
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, player := range core.ConnectedPlayers {
				message := fmt.Sprintf(`{"command": "update_waiting_queue", "value": {"waiting_queue_size": %d}}`, len(core.WaitingQueue))
				if err := player.Conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
					fmt.Printf("Error sending message to player %s: %v\n", player.Name, err)
				}
			}
		}
	}
}
