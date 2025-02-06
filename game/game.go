package game

import (
	"checkers-server/core"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const pingInterval = 10 * time.Second

func Start(p1, p2 *core.Player, disconnectCallback func(*core.Player, *core.Player)) {
    
	//ping.HandleDisconnects(p1, p2, disconnectCallback) // TODO: Dont think I need this here...

	for {
		_, msg, err := p1.Conn.ReadMessage()
		if err != nil {
			fmt.Println("Player 1 disconnected.")
			disconnectCallback(p1, p2)
			return
		}
		p2.Conn.WriteMessage(websocket.TextMessage, msg)

		_, msg, err = p2.Conn.ReadMessage()
		if err != nil {
			fmt.Println("Player 2 disconnected.")
			disconnectCallback(p2, p1)
			return
		}
		p1.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}
