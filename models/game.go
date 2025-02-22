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
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TODO: Have this expanded upon.
// ? - Copy this into our game? as to have a new instance of it to work with?
// ? - Update validSquares at game start?
// ? - Move it to redis??
var validSquares = map[string]bool{
	"A1": false, "A3": true, "A5": true, "A7": true,
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

type Piece struct {
	Type    string `json:"type"`
	PlayerID string `json:"player_id"`
	PieceID string `json:"piece_id"`
}

func generateInitialBoard(blackID, whiteID string) map[string]*Piece {
	board := make(map[string]*Piece)
	rows := []string{"A", "B", "C", "D", "E", "F", "G", "H"}

	for i, row := range rows {
		for col := 1; col <= 8; col++ {
			pos := fmt.Sprintf("%s%d", row, col)

			// Only place pieces on dark squares
			if (i+col)%2 == 1 {
				if i < 3 { // Top 3 rows for black pieces
					board[pos] = &Piece{Type: "b", PieceID: uuid.New().String(), PlayerID: blackID}
				} else if i > 4 { // Bottom 3 rows for white pieces
					board[pos] = &Piece{Type: "w", PieceID: uuid.New().String(), PlayerID: whiteID}
				} else {
					board[pos] = nil // Empty middle rows
				}
			}
		}
	}
	return board
}

func printBoard(board map[string]*Piece) {
	rows := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	fmt.Println("  1 2 3 4 5 6 7 8")
	for _, row := range rows {
		fmt.Print(row + " ")
		for col := 1; col <= 8; col++ {
			pos := fmt.Sprintf("%s%d", row, col)
			if piece, exists := board[pos]; exists && piece != nil {
				fmt.Print(piece.Type + " ")
			} else {
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

func getPieceUUID() *string {
	id := uuid.New().String() 
	return &id
}

type GamePlayer struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	SessionID string `json:"session_id"`
}

type Game struct {
	ID    string       `json:"id"`
	Board     map[string]*Piece `json:"board"`
	Players   []GamePlayer `json:"players"`
	CurrentPlayerID string `json:"current_player_id"`
	Turn      int          `json:"turn"`
	Kinged    Kinged       `json:"kinged"`
	Moves     []string     `json:"moves"`
	StartTime time.Time    `json:"start_time"`
	EndTime   time.Time    `json:"end_time"`
	Winner    string       `json:"winner"`
}

type Kinged struct {
	W []string `json:"w"`
	B []string `json:"b"`
}

// Move represents a single move in the game
type Move struct {
	PlayerID  string `json:"player_id"`  // The player making the move
	PieceID   string `json:"piece_id"`  // Will be given to clientes by the server.
	From      string `json:"from"`      // e.g., "A1"
	To        string `json:"to"`        // e.g., "B2"
	IsCapture bool   `json:"is_capture"` // Whether the move captured an opponent's piece
	IsKinged  bool   `json:"is_kinged"`  // Whether the piece was kinged after the move
}

func MapPlayerToGamePlayer(player Player) GamePlayer {
	return GamePlayer{
		ID:        player.ID,
		Name:      player.Name,
		Token:     player.Token,
		SessionID: player.SessionID,
	}
}

func mapPlayers(r *Room) []GamePlayer {
	players := []GamePlayer{}

	if r.Player1 != nil {
		players = append(players, MapPlayerToGamePlayer(*r.Player1))
	}
	if r.Player2 != nil {
		players = append(players, MapPlayerToGamePlayer(*r.Player2))
	}

	return players
}

func (r *Room) NewGame() *Game {
	whiteID , err := r.GetOpponentPlayerID(r.CurrentPlayerID)
	if err != nil {
		// TODO: Return an error?
	}

	game := Game{
		ID:     r.ID,
		Board:     generateInitialBoard(r.CurrentPlayerID, whiteID),
		Players:   mapPlayers(r), 
		CurrentPlayerID: r.CurrentPlayerID,
		Turn:      r.Turn,
		Kinged:    Kinged{W: []string{}, B: []string{}},
		Moves:     []string{},
		StartTime: r.StartDate,
		Winner:    "",
	}
	printBoard(game.Board) 
	return &game
}
