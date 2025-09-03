package models

import (
	"fmt"
)

type Board interface {
	GetPiece(pos string) (PieceInterface, bool)
	GetPieceByID(id string) (PieceInterface, bool)
	GetPieces() []PieceInterface
	GetGrid() map[string]PieceInterface
	RemovePiece(pos string)
	MovePiece(from, to string) error
	PiecesThatCanCapture(playerID string) []PieceInterface
	CanPieceCaptureNEW(pos string) bool
	IsValidMoveKing(move Move) (bool, error)
	IsValidMove(move Move) (bool, error)
	WasPieceKinged(pos string, piece PieceInterface) bool
}

func NewBoard(blackID, whiteID, boardtype, gameName string) Board {
	switch gameName {
	case "BatalhaDasDamas":
		return NewDamasBoard(blackID, whiteID, boardtype)
	case "BatalhaDoChess":
		return NewChessBoard(blackID, whiteID, boardtype)
	}
	return nil
}

func NewDamasBoard(blackID, whiteID, boardtype string) Board {
	board := &DamasBoard{Grid: make(map[string]*DamasPiece)}
	switch boardtype {
	case "std-game":
		board.GenerateInitialBoard(blackID, whiteID) // Automatically initialize board state
	case "two-pieces-endgame":
		board.GenerateEndGameTestBoard(blackID, whiteID) // Automatically initialize board state
	case "multiple-capture":
		board.GenerateMultipleCaptureTestBoard(blackID, whiteID) // Automatically initialize board state
	}
	return board
}

func NewChessBoard(blackID, whiteID, boardtype string) Board {
	board := &ChessBoard{Grid: make(map[string]*ChessPiece)} // use ChessBoard here!
	switch boardtype {
	case "std-game":
		board.GenerateInitialBoard(blackID, whiteID)
	case "two-pieces-endgame":
		board.GenerateEndGameTestBoard(blackID, whiteID)
	case "multiple-capture":
		board.GenerateMultipleCaptureTestBoard(blackID, whiteID)
	}
	return board
}

func isInBounds(row rune, col int) bool {
	return row >= 'A' && row <= 'H' && col >= 1 && col <= 8
}

// parsePosition converts a position string (e.g., "A3") into row (rune) and column (int).
// Returns an error if the position is invalid.
func parsePosition(pos string) (rune, int, error) {
	if len(pos) != 2 {
		return 0, 0, fmt.Errorf("(Parse Position) - invalid position format: must be 2 characters (e.g., 'A3')")
	}
	row := rune(pos[0])      // Convert the first character to a rune (e.g., 'A')
	col := int(pos[1] - '0') // Convert the second character to an integer (e.g., '3' → 3)

	// Validate the row and column
	if row < 'A' || row > 'H' || col < 1 || col > 8 {
		return 0, 0, fmt.Errorf("(Parse Position) - position is out of bounds: must be between A1 and H8")
	}
	return row, col, nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
