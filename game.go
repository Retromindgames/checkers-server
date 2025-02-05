package main

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// Ping interval in seconds
const pingInterval = 10 * time.Second

// gameLoop contains the main game loop and periodic pinging.
func gameLoop(p1, p2 *Player) {
	// Send periodic ping to both players
	go func() {
		for {
			select {
			case <-time.After(pingInterval):
				// Send ping to player 1
				if err := p1.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					fmt.Println("Player 1 disconnected (ping failed).")
					handleDisconnection(p1, p2)
					return
				}
				// Send ping to player 2
				if err := p2.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					fmt.Println("Player 2 disconnected (ping failed).")
					handleDisconnection(p2, p1)
					return
				}
			}
		}
	}()

	// Existing message loop
	for {
		_, msg, err := p1.conn.ReadMessage()
		if err != nil {
			fmt.Println("Player 1 disconnected.")
			handleDisconnection(p1, p2)
			return
		}
		p2.conn.WriteMessage(websocket.TextMessage, msg)

		_, msg, err = p2.conn.ReadMessage()
		if err != nil {
			fmt.Println("Player 2 disconnected.")
			handleDisconnection(p2, p1)
			return
		}
		p1.conn.WriteMessage(websocket.TextMessage, msg)
	}
}
