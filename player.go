package main

import "github.com/gorilla/websocket"

// Player represents a single player in the game.
type Player struct {
	conn *websocket.Conn
	room *GameRoom
}
