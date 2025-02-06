package game

import (
	"checkers-server/core"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

var Mutex sync.Mutex

func AddPlayerToQueue(player *core.Player) {
	Mutex.Lock()
	core.WaitingQueue = append(core.WaitingQueue, player)

	if len(core.WaitingQueue) >= 2 {
		p1 := core.WaitingQueue[0]
		p2 := core.WaitingQueue[1]
		core.WaitingQueue = core.WaitingQueue[2:]

		room := &core.Room{Player1: p1, Player2: p2}
		core.Rooms = append(core.Rooms, room)
		p1.Room = room
		p2.Room = room

		Mutex.Unlock()

		fmt.Println("New game started!")

		p1.Conn.WriteMessage(websocket.TextMessage, []byte("Paired! Game started."))
		p2.Conn.WriteMessage(websocket.TextMessage, []byte("Paired! Game started."))

		go Loop(p1, p2) // Start game loop
	} else {
		Mutex.Unlock()
		fmt.Println("Waiting for opponent...")
		player.Conn.WriteMessage(websocket.TextMessage, []byte("Waiting for opponent..."))
	}
}
