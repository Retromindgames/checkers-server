package models

/*

{
  "gameId": "game123",
  "board": [
    ["b", "", "b", "", "b", "", "b", ""],
    ["", "b", "", "b", "", "b", "", "b"],
    ["b", "", "b", "", "b", "", "b", ""],
    ["", "", "", "", "", "", "", ""],
    ["", "", "", "", "", "", "", ""],
    ["", "w", "", "w", "", "w", "", "w"],
    ["w", "", "w", "", "w", "", "w", ""],
    ["", "w", "", "w", "", "w", "", "w"]
  ],
  "players": [
    {
      "id": 1,
      "name": "player1",
      "color": "white",
      "sessionId": "sessionA"
    },
    {
      "id": 2,
      "name": "player2",
      "color": "black",
      "sessionId": "sessionB"
    }
  ],
  "turn": 1,
  "kinged": {
    "w": [],
    "b": []
  },
  "moves": [],
  "startTime": "2025-02-18T14:00:00Z",
  "endTime": "2025-02-18T15:00:00Z",
  "winner": "white"
}
*/

import (
	"time"

	"github.com/google/uuid"
)

// TODO: Have this expanded upon.
// ? - Copy this into our game? as to have a new instance of it to work with?
// ? - Update validSquares at game start?
// ? - Move it to redis??
var validSquares = map[string]bool{
	"A1": true, "A3": true, "A5": true, "A7": true,
	"B2": true, "B4": true, "B6": true, "B8": true,
	"C1": true, "C3": true, "C5": true, "C7": true,
	"D2": true, "D4": true, "D6": true, "D8": true,
	"E1": true, "E3": true, "E5": true, "E7": true,
	"F2": true, "F4": true, "F6": true, "F8": true,
	"G1": true, "G3": true, "G5": true, "G7": true,
	"H2": true, "H4": true, "H6": true, "H8": true,
}

/*
	? - Check if a square is valid
	square := "A3"
	if valid, exists := validSquares[square]; exists && valid {
		fmt.Println(square, "is a valid square")
	} else {
		fmt.Println(square, "is not a valid square")
	}

	? - Display the valid squares
	fmt.Println("Valid squares on the checkers board:")
	for square := range validSquares {
		fmt.Println(square)
	}
*/

var initialBoard = map[string]*string{
	"A1": getPieceUUID(), "A3": getPieceUUID(), "A5": getPieceUUID(), "A7": getPieceUUID(),
	"B2": getPieceUUID(), "B4": getPieceUUID(), "B6": getPieceUUID(), "B8": getPieceUUID(),
	"C1": nil, "C3": nil, "C5": nil, "C7": nil,
	"D2": nil, "D4": nil, "D6": nil, "D8": nil,
	"E1": nil, "E3": nil, "E5": nil, "E7": nil,
	"F2": nil, "F4": nil, "F6": nil, "F8": nil,
	"G1": getPieceUUID(), "G3": getPieceUUID(), "G5": getPieceUUID(), "G7": getPieceUUID(),
	"H2": getPieceUUID(), "H4": getPieceUUID(), "H6": getPieceUUID(), "H8": getPieceUUID(),
}

func getPieceUUID() *string {
	id := uuid.New().String() // Generate a UUID
	return &id
}

type GamePlayer struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	SessionID string `json:"sessionId"`
}

type Game struct {
	GameID    string       `json:"gameId"`
	Board     [8][8]string `json:"board"`
	Players   []GamePlayer `json:"players"`
	Turn      int          `json:"turn"`
	Kinged    Kinged       `json:"kinged"`
	Moves     []string     `json:"moves"`
	StartTime time.Time    `json:"startTime"`
	EndTime   time.Time    `json:"endTime"`
	Winner    string       `json:"winner"`
}

type Kinged struct {
	W []string `json:"w"`
	B []string `json:"b"`
}

// Move represents a single move in the game
type Move struct {
	PlayerID  string `json:"playerId"`  // The player making the move
	From      string `json:"from"`      // e.g., "A1"
	To        string `json:"to"`        // e.g., "B2"
	IsCapture bool   `json:"isCapture"` // Whether the move captured an opponent's piece
	IsKinged  bool   `json:"isKinged"`  // Whether the piece was kinged after the move
}

func MapPlayerToGamePlayer(player Player) GamePlayer {
	return GamePlayer{
		ID:        player.ID,
		Name:      player.Name,
		Token:     player.Token,
		SessionID: player.SessionID,
	}
}

func (r *Room) NewGame() Game {
	return Game{
		GameID:    r.ID,
		Board:     initialBoard,
		Players:   mapPlayers(r), // TODO:
		Turn:      r.Turn,
		Kinged:    Kinged{W: []string{}, B: []string{}},
		Moves:     []string{},
		StartTime: r.StartDate,
		Winner:    "",
	}
}
