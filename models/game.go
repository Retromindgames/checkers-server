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
			board[pos] = nil
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

func generateEndGameTestBoard(blackID, whiteID string) map[string]*Piece {
	board := make(map[string]*Piece)

	// Set positions for testing
	testPositions := map[string]*Piece{
		"A2": &Piece{Type: "b", PieceID: uuid.New().String(), PlayerID: blackID}, // Black piece
		"B3": &Piece{Type: "w", PieceID: uuid.New().String(), PlayerID: whiteID}, // White piece
	}

	// Initialize board and place test pieces
	for row := 'A'; row <= 'H'; row++ {
		for col := 1; col <= 8; col++ {
			pos := fmt.Sprintf("%c%d", row, col)
			if piece, exists := testPositions[pos]; exists {
				board[pos] = piece // Place test pieces
			} else {
				board[pos] = nil // Empty squares
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
	NumPieces int	 `json:"num_pieces"`
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
		//Board:     generateInitialBoard(r.CurrentPlayerID, whiteID),
		Board:    	generateEndGameTestBoard(r.CurrentPlayerID, whiteID),
		Players:   mapPlayers(r), 
		CurrentPlayerID: r.CurrentPlayerID,
		Turn:      r.Turn,
		Kinged:    Kinged{W: []string{}, B: []string{}},
		Moves:     []string{},
		StartTime: r.StartDate,
		Winner:    "",
	}
	game.UpdatePlayerPieces() // Set NumPieces for each player

	//printBoard(game.Board) 
	return &game
}

func (g * Game) CountPlayerPieces(playerID string) int {
	count := 0
	for _, piece := range g.Board {
		if piece != nil && piece.PlayerID == playerID {
			count++
		}
	}
	return count
}

func (g *Game) UpdatePlayerPieces() {
	for i := range g.Players {
		g.Players[i].NumPieces = g.CountPlayerPieces(g.Players[i].ID)
	}
}

func (g *Game) GetOpponentPlayerID(playerID string) (string, error) {
	if len(g.Players) != 2 {
		return "", fmt.Errorf("invalid number of players in game")
	}

	for _, player := range g.Players {
		if player.ID != playerID {
			return player.ID, nil
		}
	}
	return "", fmt.Errorf("opponent not found for player ID: %s", playerID)
}

func (g *Game) GetOpponentGamePlayer(playerID string) (*GamePlayer, error) {
	if len(g.Players) != 2 {
		return nil, fmt.Errorf("invalid number of players in game")
	}

	for _, player := range g.Players {
		if player.ID != playerID {
			return &player, nil
		}
	}

	return nil, fmt.Errorf("opponent not found for player ID: %s", playerID)
}

func (g *Game) RemovePiece(pos string) {
	if _, exists := g.Board[pos]; exists {
		g.Board[pos] = nil
	}
}

func(g *Game) MovePiece(move Move){
	
	// Validate move
	piece, exists := g.Board[move.From]								// TODO: This was commented, since the FE seems to be sending the wrong ids.
	if !exists || piece == nil || piece.PieceID != move.PieceID /*|| piece.PlayerID != move.PlayerID*/ {
		// Invalid move, update and break		
		return
	}

	// Move piece to new position
	g.Board[move.To] = piece
	g.Board[move.From] = nil

	// TODO: Handle kinging, find kinged piece.
	if move.IsKinged {
		//piece.Type = strings.ToUpper(piece.Type) // Convert to uppercase to indicate a king
	}

	// TODO: Review and test this. Handle capture (assumes captured piece's position is between From and To)
	if move.IsCapture {
		midRow := (move.From[0] + move.To[0]) / 2
		midCol := (move.From[1] + move.To[1]) / 2
		capturePos := fmt.Sprintf("%c%c", midRow, midCol)
		g.Board[capturePos] = nil // Remove captured piece
	}	
}

func (g *Game) CheckGameOver() bool {
	// Check each player for pieces
	for _, player := range g.Players {
		if player.NumPieces == 0 {
			return true // Game over if any player's pieces are zero
		}
	}
	return false // Game continues if both players have pieces
}


// TODO: USE helper function for logging errors
func logError(message string, err error) {
	//fmt.Printf("[%s-%d] - (Process Game Moves) - %s: %v\n", name, pid, message, err)
}