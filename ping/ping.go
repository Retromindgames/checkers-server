// ping.go in the ping package
package ping

import (
	"checkers-server/core" // assuming core package holds your Player struct and other logic
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const pingInterval = 30 * time.Second // Set the interval for pinging
// TODO: I dont think i need this.. 
// HandleDisconnects will handle disconnections based on the ping response
func HandleDisconnects(p1, p2 *core.Player, disconnectCallback func(*core.Player, *core.Player)) {
    go func() {
        for {
            time.Sleep(pingInterval)

            if err := p1.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                fmt.Println("Player 1 disconnected (ping failed).")
                disconnectCallback(p1, p2) // Call handler-provided function
                return
            }

            if err := p2.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                fmt.Println("Player 2 disconnected (ping failed).")
                disconnectCallback(p2, p1)
                return
            }
        }
    }()
}