package models

import (
	"fmt"

	"github.com/google/uuid"
)

type ChessBoard struct {
	Grid map[string]*ChessPiece
}

func (b *ChessBoard) GetPieceByID(id string) (PieceInterface, bool) {
	for _, p := range b.Grid {
		if p != nil && p.GetID() == id {
			return p, true
		}
	}
	return nil, false
}

func (b *ChessBoard) GetPiece(pos string) (PieceInterface, bool) {
	p, ok := b.Grid[pos]
	return p, ok
}

func (b *ChessBoard) GetPieces() []PieceInterface {
	pieces := make([]PieceInterface, 0, len(b.Grid))
	for _, p := range b.Grid {
		if p != nil {
			pieces = append(pieces, p)
		}
	}
	return pieces
}

func (b *ChessBoard) GetGrid() map[string]PieceInterface {
	grid := make(map[string]PieceInterface)
	for pos, piece := range b.Grid {
		if piece != nil {
			grid[pos] = piece // *ChessPiece implements PieceInterface
		}
	}
	return grid
}

func (b *ChessBoard) RemovePiece(pos string) {
	b.Grid[pos] = nil
}

func (b *ChessBoard) MovePiece(from, to string) error {
	piece, ok := b.Grid[from]
	if !ok {
		return fmt.Errorf("no piece at %s", from)
	}
	b.Grid[to] = piece
	b.Grid[from] = nil
	return nil
}

// GenerateInitialBoard initializes the chess board with starting pieces
func (b *ChessBoard) GenerateInitialBoard(blackID, whiteID string) {
	files := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	b.Grid = make(map[string]*ChessPiece)

	// Place pawns
	for _, file := range files {
		// White pawns on row 2
		b.Grid[fmt.Sprintf("%s2", file)] = &ChessPiece{
			Type:     "pawn",
			PieceID:  uuid.New().String(),
			PlayerID: whiteID,
			Color:    "w",
			IsAlive:  true,
		}
		// Black pawns on row 7
		b.Grid[fmt.Sprintf("%s7", file)] = &ChessPiece{
			Type:     "pawn",
			PieceID:  uuid.New().String(),
			PlayerID: blackID,
			Color:    "b",
			IsAlive:  true,
		}
	}

	// Place major pieces
	pieceOrder := []string{"rook", "knight", "bishop", "queen", "king", "bishop", "knight", "rook"}
	for i, file := range files {
		// White back rank (row 1)
		b.Grid[fmt.Sprintf("%s1", file)] = &ChessPiece{
			Type:     pieceOrder[i],
			PieceID:  uuid.New().String(),
			PlayerID: whiteID,
			Color:    "w",
			IsAlive:  true,
		}
		// Black back rank (row 8)
		b.Grid[fmt.Sprintf("%s8", file)] = &ChessPiece{
			Type:     pieceOrder[i],
			PieceID:  uuid.New().String(),
			PlayerID: blackID,
			Color:    "b",
			IsAlive:  true,
		}
	}

	// Fill the empty squares with nil
	for _, file := range files {
		for rank := 1; rank <= 8; rank++ {
			pos := fmt.Sprintf("%s%d", file, rank)
			if _, ok := b.Grid[pos]; !ok {
				b.Grid[pos] = nil
			}
		}
	}
}

// GenerateEndGameTestBoard initializes the board with a test configuration
func (b *ChessBoard) GenerateEndGameTestBoard(blackID, whiteID string) {

}

// Test config for multiple capture
func (b *ChessBoard) GenerateMultipleCaptureTestBoard(blackID, whiteID string) {

}

func (b *ChessBoard) IsValidMove(move Move) (bool, error) {
	return true, nil
}

func (b *ChessBoard) IsValidMoveKing(move Move) (bool, error) {
	piece, exists := b.Grid[move.From]
	if !exists || piece == nil {
		return false, fmt.Errorf("(IsValidMoveKing) - piece does not exist at source")
	}
	if piece.PlayerID != move.PlayerID {
		return false, fmt.Errorf("(IsValidMoveKing) - piece does not belong to player")
	}
	if !piece.IsPieceKinged() {
		return false, fmt.Errorf("(IsValidMoveKing) - piece is not kinged")
	}

	fromRow, fromCol, err := parsePosition(move.From)
	if err != nil {
		return false, fmt.Errorf("(IsValidMoveKing) - invalid source: %v", err)
	}
	toRow, toCol, err := parsePosition(move.To)
	if err != nil {
		return false, fmt.Errorf("(IsValidMoveKing) - invalid destination: %v", err)
	}

	if _, ok := b.Grid[move.To]; !ok {
		return false, fmt.Errorf("(IsValidMoveKing) - destination does not exist")
	}
	if b.Grid[move.To] != nil {
		return false, fmt.Errorf("(IsValidMoveKing) - destination not empty")
	}

	deltaRow := int(toRow - fromRow)
	deltaCol := int(toCol - fromCol)

	if abs(deltaRow) != abs(deltaCol) {
		return false, fmt.Errorf("(IsValidMoveKing) - move not diagonal")
	}

	stepRow := 1
	if deltaRow < 0 {
		stepRow = -1
	}
	stepCol := 1
	if deltaCol < 0 {
		stepCol = -1
	}

	enemySeen := false
	for r, c := fromRow+rune(stepRow), fromCol+stepCol; r != toRow && c != toCol; r, c = r+rune(stepRow), c+stepCol {
		square := string(r) + string('0'+c)
		p, exists := b.Grid[square]
		if !exists {
			return false, fmt.Errorf("(IsValidMoveKing) - square %v does not exist", square)
		}
		if p == nil {
			continue
		}
		if p.PlayerID == move.PlayerID {
			return false, fmt.Errorf("(IsValidMoveKing) - path blocked by own piece at %v", square)
		}
		// This looked sketchi.
		if enemySeen {
			return false, fmt.Errorf("(IsValidMoveKing) - multiple captures not supported in one move")
		}
		enemySeen = true
	}

	if enemySeen && !move.IsCapture {
		return false, fmt.Errorf("(IsValidMoveKing) - move is a capture but not flagged as capture")
	}
	if !enemySeen && move.IsCapture {
		return false, fmt.Errorf("(IsValidMoveKing) - flagged as capture but no enemy on path")
	}

	return true, nil
}

func (b *ChessBoard) PiecesThatCanCapture(playerID string) []PieceInterface {
	return nil
}

func (b *ChessBoard) WasPieceKinged(pos string, piece PieceInterface) bool {
	return false
}

func (b *ChessBoard) CanPieceCaptureNEW(pos string) bool {
	return true
}
