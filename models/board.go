package models

import (
	"fmt"

	"github.com/google/uuid"
)

type Board struct {
	Grid map[string]*Piece
}

func NewBoard(blackID, whiteID, boardtype string) *Board {
	board := &Board{Grid: make(map[string]*Piece)}
	switch boardtype {
	case "std-game":
		board.GenerateInitialBoard(blackID, whiteID) // Automatically initialize board state
	case "two-pieces-endgame":
		board.GenerateEndGameTestBoard(blackID, whiteID) // Automatically initialize board state
	}
	return board
}

// GenerateInitialBoard initializes the board with starting pieces
func (b *Board) GenerateInitialBoard(blackID, whiteID string) {
	rows := []string{"A", "B", "C", "D", "E", "F", "G", "H"}

	for i, row := range rows {
		for col := 1; col <= 8; col++ {
			pos := fmt.Sprintf("%s%d", row, col)
			b.Grid[pos] = nil
			// Only place pieces on dark squares
			if (i+col)%2 == 1 {
				if i < 3 { // Top 3 rows for black pieces
					b.Grid[pos] = &Piece{Type: "b", PieceID: uuid.New().String(), PlayerID: blackID}
				} else if i > 4 { // Bottom 3 rows for white pieces
					b.Grid[pos] = &Piece{Type: "w", PieceID: uuid.New().String(), PlayerID: whiteID}
				} else {
					b.Grid[pos] = nil // Empty middle rows
				}
			}
		}
	}
}

// GenerateEndGameTestBoard initializes the board with a test configuration
func (b *Board) GenerateEndGameTestBoard(blackID, whiteID string) {
	// Set positions for testing
	testPositions := map[string]*Piece{
		"A2": {Type: "b", PieceID: uuid.New().String(), PlayerID: blackID}, // Black piece
		"B3": {Type: "w", PieceID: uuid.New().String(), PlayerID: whiteID}, // White piece
	}

	// Initialize board and place test pieces
	for row := 'A'; row <= 'H'; row++ {
		for col := 1; col <= 8; col++ {
			pos := fmt.Sprintf("%c%d", row, col)
			if piece, exists := testPositions[pos]; exists {
				b.Grid[pos] = piece // Place test pieces
			} else {
				b.Grid[pos] = nil // Empty squares
			}
		}
	}
}

// TODO: Gotta finish implementing this.
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
