package handlers

import (
	"checkers-server/core"
	"checkers-server/game"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade failed:", err)
		return
	}

	fmt.Println("New player connected:", r.RemoteAddr)

	player := &core.Player{Conn: conn}
	conn.WriteMessage(websocket.TextMessage, []byte("Connected successfully!"))

	game.AddPlayerToQueue(player)
}
