package handlers

import (
	"checkers-server/core"
	"checkers-server/message"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var Mutex sync.Mutex

func HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade failed:", err)
		return
	}

	player := &core.Player{Conn: conn}
	core.AddPlayer(player)
	fmt.Println("New player connected:", r.RemoteAddr)
	conn.WriteMessage(websocket.TextMessage, []byte("Connected successfully!"))

	// TODO: Since this is pretty much the main "loop" of the client connection. Maybe move it elsewheere??
	go func() {
		defer conn.Close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Player disconnected:", r.RemoteAddr)
				HandleDisconnection(player, player.Room.GetOpponent(player))
				return
			}

			message, err := message.ParseMessage(msg, conn)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte("Invalid message format."))
				continue
			}

			// TODO: After creating the unmarshal package, organize this code.
			if message.Command == "join_queue" {
				if core.IsPlayerInQueue(player) {
					fmt.Println("Player already in queue:", r.RemoteAddr)
					player.Conn.WriteMessage(websocket.TextMessage, []byte("You are already in a Queue!..."))
				} else {
					fmt.Println("Player joining queue:", r.RemoteAddr)
					player.SelectedBid = message.Value
					core.AddToQueue(player)
					if len(core.WaitingQueue) >= 2 {
						filteredQueue := core.FilterWaitingQueue(core.WaitingQueue, func(player *core.Player) bool {
							return player.SelectedBid == message.Value
						})
						if len(filteredQueue) >= 2{
							HandleRoomCreation(player.SelectedBid)	
						}
					}else {
						player.Conn.WriteMessage(websocket.TextMessage, []byte("Waiting for an opponent..."))
					}
				}
			} 
		}
	}()
}

func HandleRoomCreation(bidAmount float64) {
	// Created room withh the first two players of the queue.
	room := core.CreateRoom(core.WaitingQueue[0], core.WaitingQueue[1], bidAmount);
	// remove them from the Queue (!)
	core.RemoveFromQueue(room.Player1);
	core.RemoveFromQueue(room.Player2);
	fmt.Println("New game started!")
	room.Player1.Conn.WriteMessage(websocket.TextMessage, []byte(`{"command": "paired", "value": 1}`))
	room.Player1.Color = 1
	room.Player2.Conn.WriteMessage(websocket.TextMessage, []byte(`{"command": "paired", "value": 0}`))
	room.Player2.Color = 0

}

func HandleDisconnection(player *core.Player, opponent *core.Player) {
	if player.Conn != nil {
		opponent.Conn.WriteMessage(websocket.TextMessage, []byte("Opponent disconnected."))
		player.Conn.Close()
		core.RemovePlayer(player)
	}
	core.RemoveRoom(player.Room);
	fmt.Println("Game ended.")
}
