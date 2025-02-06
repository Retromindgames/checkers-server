package game

import (
	"checkers-server/core"
	"fmt"
	"time"
)

const pingInterval = 10 * time.Second

func Loop(p1, p2 *core.Player) {
	go func() {
		for {
			time.Sleep(pingInterval)

			// Ping player 1
			if err := p1.Conn.WriteMessage(9, nil); err != nil {
				fmt.Println("Player 1 disconnected (ping failed).")
				HandleDisconnection(p1, p2)
				return
			}

			// Ping player 2
			if err := p2.Conn.WriteMessage(9, nil); err != nil {
				fmt.Println("Player 2 disconnected (ping failed).")
				HandleDisconnection(p2, p1)
				return
			}
		}
	}()

	for {
		_, msg, err := p1.Conn.ReadMessage()
		if err != nil {
			fmt.Println("Player 1 disconnected.")
			HandleDisconnection(p1, p2)
			return
		}
		p2.Conn.WriteMessage(1, msg)

		_, msg, err = p2.Conn.ReadMessage()
		if err != nil {
			fmt.Println("Player 2 disconnected.")
			HandleDisconnection(p2, p1)
			return
		}
		p1.Conn.WriteMessage(1, msg)
	}
}
