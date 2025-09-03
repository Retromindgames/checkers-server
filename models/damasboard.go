package models

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

type DamasBoard struct {
	Grid map[string]*DamasPiece
}

func (b *DamasBoard) GetPieceByID(id string) (PieceInterface, bool) {
	for _, p := range b.Grid {
		if p != nil && p.GetID() == id {
			return p, true
		}
	}
	return nil, false
}

func (b *DamasBoard) GetPiece(pos string) (PieceInterface, bool) {
	p, ok := b.Grid[pos]
	return p, ok
}

func (b *DamasBoard) GetPieces() []PieceInterface {
	pieces := make([]PieceInterface, 0, len(b.Grid))
	for _, p := range b.Grid {
		if p != nil {
			pieces = append(pieces, p)
		}
	}
	return pieces
}

func (b *DamasBoard) GetGrid() map[string]PieceInterface {
	grid := make(map[string]PieceInterface)
	for pos, piece := range b.Grid {
		if piece != nil {
			grid[pos] = piece // *DamasPiece implements PieceInterface
		}
	}
	return grid
}

func (b *DamasBoard) RemovePiece(pos string) {
	b.Grid[pos] = nil
}

func (b *DamasBoard) MovePiece(from, to string) error {
	piece, ok := b.Grid[from]
	if !ok {
		return fmt.Errorf("no piece at %s", from)
	}
	b.Grid[to] = piece
	b.Grid[from] = nil
	return nil
}

// GenerateInitialBoard initializes the board with starting pieces
func (b *DamasBoard) GenerateInitialBoard(blackID, whiteID string) {
	rows := []string{"A", "B", "C", "D", "E", "F", "G", "H"}

	for i, row := range rows {
		for col := 1; col <= 8; col++ {
			pos := fmt.Sprintf("%s%d", row, col)
			b.Grid[pos] = nil
			// Only place pieces on dark squares
			if (i+col)%2 == 1 {
				if i < 3 { // Top 3 rows for black pieces
					b.Grid[pos] = &DamasPiece{Type: "b", PieceID: uuid.New().String(), PlayerID: blackID}
				} else if i > 4 { // Bottom 3 rows for white pieces
					b.Grid[pos] = &DamasPiece{Type: "w", PieceID: uuid.New().String(), PlayerID: whiteID}
				} else {
					b.Grid[pos] = nil // Empty middle rows
				}
			}
		}
	}
}

// GenerateEndGameTestBoard initializes the board with a test configuration
func (b *DamasBoard) GenerateEndGameTestBoard(blackID, whiteID string) {
	// Set positions for testing
	testPositions := map[string]*DamasPiece{
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

// Test config for multiple capture
func (b *DamasBoard) GenerateMultipleCaptureTestBoard(blackID, whiteID string) {
	// Set positions for testing
	testPositions := map[string]*DamasPiece{
		"A1": {Type: "b", PieceID: uuid.New().String(), PlayerID: blackID},
		"A3": {Type: "b", PieceID: uuid.New().String(), PlayerID: blackID},
		"A7": {Type: "b", PieceID: uuid.New().String(), PlayerID: blackID},
		"B2": {Type: "b", PieceID: uuid.New().String(), PlayerID: blackID},
		"B4": {Type: "b", PieceID: uuid.New().String(), PlayerID: blackID},
		"C5": {Type: "b", PieceID: uuid.New().String(), PlayerID: blackID},
		"D6": {Type: "w", PieceID: uuid.New().String(), PlayerID: whiteID},
		"E7": {Type: "w", PieceID: uuid.New().String(), PlayerID: whiteID},
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

func (b *DamasBoard) CanPieceCaptureNEW(pos string) bool {
	piece, exists := b.Grid[pos]
	if !exists || piece == nil {
		log.Printf("(CanPieceCapture) - Piece doesn't exist on the board\n")
		return false // No piece at this position
	}

	// Define the possible capture directions
	var directions []struct{ rowDelta, colDelta int }
	if piece.IsKinged {
		// Kings can move in all four diagonal directions
		directions = []struct{ rowDelta, colDelta int }{
			{1, 1},   // Diagonal right (down)
			{1, -1},  // Diagonal left (down)
			{-1, 1},  // Diagonal right (up)
			{-1, -1}, // Diagonal left (up)
		}
		//log.Printf("(CanPieceCapture) - Piece is king, multiple directions.")
	} else {
		//log.Printf("(CanPieceCapture) - Piece is not king, single direction.")
		// Normal pieces can only move forward
		var direction = b.GetPieceDirection(*piece)
		directions = []struct{ rowDelta, colDelta int }{
			{direction, 1},  // Diagonal right
			{direction, -1}, // Diagonal left
		}
	}

	// Convert position (e.g., "A3" → row 'A', col 3)
	fromRow, fromCol, err := parsePosition(pos)
	if err != nil {
		log.Println(err)
		return false
	}
	if piece.IsKinged {
		return b.canKingCapture(fromRow, fromCol, piece)
	}
	for _, dir := range directions {
		// Compute middle position (opponent's piece)
		midRow := fromRow + rune(dir.rowDelta)
		midCol := fromCol + dir.colDelta
		midPos := fmt.Sprintf("%c%d", midRow, midCol)

		// Compute landing position
		landRow := fromRow + rune(2*dir.rowDelta)
		landCol := fromCol + 2*dir.colDelta
		landPos := fmt.Sprintf("%c%d", landRow, landCol)
		//log.Printf("(CanPieceCapture) - Checking landPos [%s]\n", landPos)
		// Ensure middle square has an opponent piece
		midPiece, midExists := b.Grid[midPos]
		if !midExists || midPiece == nil || midPiece.PlayerID == piece.PlayerID {
			//log.Printf("(CanPieceCapture) - Middle piece doesn't exist or is not an opponent!\n")
			continue // No opponent to jump over
		}
		//log.Printf("(CanPieceCapture) - Middle piece exists and is an opponent!\n")
		// Ensure landing square is empty
		if destPiece, destExists := b.Grid[landPos]; destExists && destPiece == nil {
			//log.Printf("(CanPieceCapture) - Valid capture move found\n")
			return true // Valid capture move found!
		}
	}
	//log.Printf("(CanPieceCapture) - No captures available\n")
	return false // No captures available
}

func (b *DamasBoard) canKingCapture(fromRow rune, fromCol int, piece *DamasPiece) bool {
	directions := []struct{ rowDelta, colDelta int }{
		{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
	}

	for _, dir := range directions {
		foundEnemy := false
		row := fromRow + rune(dir.rowDelta)
		col := fromCol + dir.colDelta

		for isInBounds(row, col) {
			pos := fmt.Sprintf("%c%d", row, col)
			target, exists := b.Grid[pos]
			if !exists {
				break
			}
			if target == nil {
				if foundEnemy {
					return true
				}
			} else if target.PlayerID != piece.PlayerID {
				if foundEnemy {
					break
				}
				foundEnemy = true
			} else {
				break
			}

			row += rune(dir.rowDelta)
			col += dir.colDelta
		}
	}
	return false
}

func (b *DamasBoard) IsValidMove(move Move) (bool, error) {
	piece, exists := b.Grid[move.From]
	if !exists || piece == nil {
		return false, fmt.Errorf("(isValidMove) - piece does not exist at the source square")
	}
	// Check if the piece belongs to the player making the move
	if piece.PlayerID != move.PlayerID {
		return false, fmt.Errorf("(isValidMove) - piece does not belong to the player")
	}
	fromRow, fromCol, err := parsePosition(move.From)
	if err != nil {
		return false, fmt.Errorf("(isValidMove) - invalid source position: %v", err)
	}
	toRow, toCol, err := parsePosition(move.To)
	if err != nil {
		return false, fmt.Errorf("(isValidMove) - invalid destination position: %v", err)
	}
	// Check if the destination square is empty
	_, exists = b.Grid[move.To]
	if !exists {
		return false, fmt.Errorf("(isValidMove) - destination square doesn't exist")
	}
	if b.Grid[move.To] != nil {
		return false, fmt.Errorf("(isValidMove) - destination square is not empty")
	}
	// Calculate the difference in rows and columns
	deltaRow := int(toRow - fromRow)
	deltaCol := toCol - fromCol
	// Skip direction validation if the piece is kinged
	if !piece.IsKinged {
		direction := b.GetPieceDirection(*piece)
		if deltaRow*direction <= 0 { // Check if the move is in the correct direction
			return false, fmt.Errorf("(isValidMove) - move is not in the correct direction for the piece type")
		}
	}
	// Check if the move is diagonal
	if abs(deltaCol) != 1 || abs(deltaRow) != 1 {
		// If it's not a single diagonal move, check if it's a capture
		if !move.IsCapture || abs(deltaCol) != 2 || abs(deltaRow) != 2 {
			return false, fmt.Errorf("(isValidMove) - move is not diagonal or a valid capture")
		}

		// For a capture, check if the intermediate square has an opponent's piece
		captureRow := fromRow + rune(deltaRow/2)
		captureCol := fromCol + deltaCol/2
		captureSquare := string(captureRow) + string('0'+captureCol)
		capturedPiece, exists := b.Grid[captureSquare]
		if !exists || capturedPiece == nil || capturedPiece.PlayerID == move.PlayerID {
			return false, fmt.Errorf("(isValidMove) - invalid capture: no opponent's piece to capture")
		}
	}
	// If all checks pass, the move is valid
	return true, nil
}

func (b *DamasBoard) IsValidMoveKing(move Move) (bool, error) {
	piece, exists := b.Grid[move.From]
	if !exists || piece == nil {
		return false, fmt.Errorf("(IsValidMoveKing) - piece does not exist at source")
	}
	if piece.PlayerID != move.PlayerID {
		return false, fmt.Errorf("(IsValidMoveKing) - piece does not belong to player")
	}
	if !piece.IsKinged {
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

func (b *DamasBoard) WasPieceKinged(pos string, piece PieceInterface) bool {
	if piece.IsPieceKinged() { // If its alerady a king we just return false.
		return false
	}

	if len(pos) == 0 {
		return false
	}
	firstChar := pos[0]
	if piece.GetType() == "b" && firstChar == 'H' {
		log.Printf("(WasPieceKinged) - Black piece was kinged!")
		return true
	}
	if piece.GetType() == "w" && firstChar == 'A' {
		log.Printf("(WasPieceKinged) - White piece was kinged!")
		return true
	}
	return false
}

func (b *DamasBoard) GetPieceDirection(piece DamasPiece) int {
	if piece.Type == "w" {
		return -1 // White pieces move "up" (decreasing row)
	} else {
		return 1 // Black pieces move "down" (increasing row)
	}
}

func (b *DamasBoard) PiecesThatCanCapture(playerID string) []PieceInterface {
	var capturers []PieceInterface
	for pos, piece := range b.Grid {
		if piece == nil {
			continue
		}
		if piece.PlayerID != playerID {
			continue
		}
		if b.CanPieceCaptureNEW(pos) {
			capturers = append(capturers, piece) // piece is *DamasPiece, which implements PieceInterface
		}
	}
	return capturers
}
