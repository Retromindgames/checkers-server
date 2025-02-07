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
				fmt.Println("Player disconnected: ", r.RemoteAddr)
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
							HandleRoomCreation(filteredQueue)	// pass the filtered queue to create a room	
						} else {
							player.Conn.WriteMessage(websocket.TextMessage, []byte("Waiting for an opponent..."))
						}
					} else {
						player.Conn.WriteMessage(websocket.TextMessage, []byte("Waiting for an opponent..."))
					}
				}
			} 
		}
	}()
}

func HandleRoomCreation(filteredQueue []*core.Player) {
	// Created room withh the first two players of the queue.
	room := core.CreateRoom(filteredQueue[0], filteredQueue[1]);
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
	// So when a player disconnects he can be:
	// - In a Queue, and not in a room.
	// - In a Room, and not in a Queue.
	// This means, we will need to handle it diferently.

	if player.Room == nil{
		core.RemoveFromQueue(player)	// We remove it from the Queue, it might now always be there tho. Dont think its an issue.
	} else {
		opponent.Conn.WriteMessage(websocket.TextMessage, []byte("Opponent disconnected."))		
		core.RemoveRoom(player.Room); 	// If there is a room, we need to remove it.
	}
	core.RemovePlayer(player)			// Either way, we need to remove the player from the room.

	fmt.Println("Game ended.")
}
