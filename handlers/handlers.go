package handlers

import (
	"checkers-server/game"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var Mutex sync.Mutex


// handleConnection manages the logic for new player connections.
func HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade failed:", err)
		return
	}

	fmt.Println("New player connected:", r.RemoteAddr)

	player := &game.Player{Conn: conn}
	conn.WriteMessage(websocket.TextMessage, []byte("Connected successfully!"))

	Mutex.Lock()
	game.WaitingQueue = append(game.WaitingQueue, player)

	// If two players are waiting, create a game room
	if len(game.WaitingQueue) >= 2 {
		p1 := game.WaitingQueue[0]
		p2 := game.WaitingQueue[1]
		game.WaitingQueue = game.WaitingQueue[2:] // Remove matched players from queue

		room := &game.Room{Player1: p1, Player2: p2}
		game.Rooms = append(game.Rooms, room) // Store active room
		p1.Room = room
		p2.Room = room

		Mutex.Unlock()

		fmt.Println("New game started!")

		p1.Conn.WriteMessage(websocket.TextMessage, []byte("Paired! Game started."))
		p2.Conn.WriteMessage(websocket.TextMessage, []byte("Paired! Game started."))

		go game.gameloop.Loop(p1, p2) // Start the game
	} else {
		Mutex.Unlock()
		fmt.Println("Waiting for opponent...")
		conn.WriteMessage(websocket.TextMessage, []byte("Waiting for opponent..."))
	}
}


// handleDisconnection handles the disconnection of players.
func HandleDisconnection(player *game.Player, opponent *game.Player) {
	// Notify both players about the disconnection
	player.Conn.WriteMessage(websocket.TextMessage, []byte("You disconnected."))
	opponent.Conn.WriteMessage(websocket.TextMessage, []byte("Opponent disconnected."))

	// Close both connections
	player.Conn.Close()
	opponent.Conn.Close()

	// Remove room from active list
	Mutex.Lock()
	for i, room := range game.Rooms {
		if room == player.Room {
			game.Rooms = append(game.Rooms[:i], game.Rooms[i+1:]...)
			break
		}
	}
	Mutex.Unlock()

	fmt.Println("Game ended.")
}
