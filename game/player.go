package game

import "github.com/gorilla/websocket"

// Player represents a single player in the game.
type Player struct {
	Conn *websocket.Conn
	Room *Room
}
