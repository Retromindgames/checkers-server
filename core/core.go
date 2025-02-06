package core

import "github.com/gorilla/websocket"

// Room represents a game room containing two players.
type Room struct {
	Player1 *Player
	Player2 *Player
}

// Player represents a single player in the game.
type Player struct {
	Conn *websocket.Conn
	Room *Room
}

var WaitingQueue []*Player
var Rooms []*Room
