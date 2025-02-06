package game

import (
	"checkers-server/core"
	"fmt"
)

// HandleDisconnection safely removes a player from a game and room.
func HandleDisconnection(player *core.Player, opponent *core.Player) {
	if player.Conn != nil {
		player.Conn.WriteMessage(1, []byte("You disconnected."))
		player.Conn.Close()
	}
	if opponent.Conn != nil {
		opponent.Conn.WriteMessage(1, []byte("Opponent disconnected."))
		opponent.Conn.Close()
	}

	// Remove room from active list
	Mutex.Lock()
	for i, room := range core.Rooms {
		if room == player.Room {
			core.Rooms = append(core.Rooms[:i], core.Rooms[i+1:]...)
			break
		}
	}
	Mutex.Unlock()

	fmt.Println("Game ended.")
}
