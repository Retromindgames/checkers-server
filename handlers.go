package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var waitingQueue []*Player
var rooms []*GameRoom
var mutex sync.Mutex

// handleConnection manages the logic for new player connections.
func handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade failed:", err)
		return
	}

	fmt.Println("New player connected:", r.RemoteAddr)

	player := &Player{conn: conn}
	conn.WriteMessage(websocket.TextMessage, []byte("Connected successfully!"))

	mutex.Lock()
	waitingQueue = append(waitingQueue, player)

	// If two players are waiting, create a game room
	if len(waitingQueue) >= 2 {
		p1 := waitingQueue[0]
		p2 := waitingQueue[1]
		waitingQueue = waitingQueue[2:] // Remove matched players from queue

		room := &GameRoom{player1: p1, player2: p2}
		rooms = append(rooms, room) // Store active room
		p1.room = room
		p2.room = room

		mutex.Unlock()

		fmt.Println("New game started!")

		p1.conn.WriteMessage(websocket.TextMessage, []byte("Paired! Game started."))
		p2.conn.WriteMessage(websocket.TextMessage, []byte("Paired! Game started."))

		go gameLoop(p1, p2) // Start the game
	} else {
		mutex.Unlock()
		fmt.Println("Waiting for opponent...")
		conn.WriteMessage(websocket.TextMessage, []byte("Waiting for opponent..."))
	}
}

// handleDisconnection handles the disconnection of players.
func handleDisconnection(player *Player, opponent *Player) {
	// Notify both players about the disconnection
	player.conn.WriteMessage(websocket.TextMessage, []byte("You disconnected."))
	opponent.conn.WriteMessage(websocket.TextMessage, []byte("Opponent disconnected."))

	// Close both connections
	player.conn.Close()
	opponent.conn.Close()

	// Remove room from active list
	mutex.Lock()
	for i, room := range rooms {
		if room == player.room {
			rooms = append(rooms[:i], rooms[i+1:]...)
			break
		}
	}
	mutex.Unlock()

	fmt.Println("Game ended.")
}
