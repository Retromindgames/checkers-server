package models

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/logger"
)

type GamePlayer struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Name      string `json:"name"`
	Timer     int    `json:"timer"`
	Color     string `json:"color"`
	SessionID string `json:"session_id"`
	NumPieces int    `json:"num_pieces"`
}

type Game struct {
	ID                 string             `json:"id"`
	Board              Board              `json:"board"`
	Players            []GamePlayer       `json:"players"`
	CurrentPlayerID    string             `json:"current_player_id"`
	Turn               int                `json:"turn"`
	Moves              []MoveInterface    `json:"moves"`
	StartTime          time.Time          `json:"start_time"`
	EndTime            time.Time          `json:"end_time"`
	Winner             string             `json:"winner"`
	BetValue           float64            `json:"bet_value"` // Bet amount for the game
	TimerSetting       string             `json:"timer_settings"`
	OperatorIdentifier OperatorIdentifier `json:"operator_identifier"`
}

type rawGame struct {
	ID                 string             `json:"id"`
	Board              json.RawMessage    `json:"board"`
	OperatorIdentifier OperatorIdentifier `json:"operator_identifier"`
	Players            []GamePlayer       `json:"players"`
	CurrentPlayerID    string             `json:"current_player_id"`
	Turn               int                `json:"turn"`
	Moves              []json.RawMessage  `json:"moves"` // raw moves
	StartTime          time.Time          `json:"start_time"`
	EndTime            time.Time          `json:"end_time"`
	Winner             string             `json:"winner"`
	BetValue           float64            `json:"bet_value"`
	TimerSettings      string             `json:"timer_settings"`
}

func UnmarshalGame(data []byte) (*Game, error) {
	var rg rawGame
	if err := json.Unmarshal(data, &rg); err != nil {
		return nil, err
	}

	var board Board
	switch rg.OperatorIdentifier.GameName {
	case "BatalhaDoChess":
		var cb ChessBoard
		if err := json.Unmarshal(rg.Board, &cb); err != nil {
			return nil, err
		}
		board = &cb
	case "BatalhaDasDamas":
		var db DamasBoard
		if err := json.Unmarshal(rg.Board, &db); err != nil {
			return nil, err
		}
		board = &db
	default:
		return nil, fmt.Errorf("unknown game type: %s", rg.OperatorIdentifier.GameName)
	}

	// decode moves
	var moves []MoveInterface
	for _, raw := range rg.Moves {
		switch rg.OperatorIdentifier.GameName {
		case "BatalhaDoChess":
			var m ChessMove
			if err := json.Unmarshal(raw, &m); err != nil {
				return nil, err
			}
			moves = append(moves, m)
		case "BatalhaDasDamas":
			var m Move // plain move
			if err := json.Unmarshal(raw, &m); err != nil {
				return nil, err
			}
			moves = append(moves, m)
		}
	}

	return &Game{
		ID:                 rg.ID,
		Board:              board,
		Players:            rg.Players,
		CurrentPlayerID:    rg.CurrentPlayerID,
		Turn:               rg.Turn,
		Moves:              moves,
		StartTime:          rg.StartTime,
		EndTime:            rg.EndTime,
		Winner:             rg.Winner,
		BetValue:           rg.BetValue,
		TimerSetting:       rg.TimerSettings,
		OperatorIdentifier: rg.OperatorIdentifier,
	}, nil
}

// Define the interface
type MoveInterface interface {
	GetPlayerID() string
	GetPieceID() string
	GetFrom() string
	GetTo() string
	IsCaptureMove() bool
	IsKingedMove() bool
	SetIsKingedMove(bool)
}
type Move struct {
	PlayerID  string `json:"player_id"`
	PieceID   string `json:"piece_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	IsCapture bool   `json:"is_capture"`
	IsKinged  bool   `json:"is_kinged"`
}

func (m Move) GetPlayerID() string    { return m.PlayerID }
func (m Move) GetPieceID() string     { return m.PieceID }
func (m Move) GetFrom() string        { return m.From }
func (m Move) GetTo() string          { return m.To }
func (m Move) IsCaptureMove() bool    { return m.IsCapture }
func (m Move) SetIsKingedMove(b bool) { m.IsCapture = b }
func (m Move) IsKingedMove() bool     { return m.IsKinged }

type ChessMove struct {
	Move                  // embed base move
	PromotionPiece string `json:"promotion_piece"`
}

func UnmarshalMove(raw []byte, gameName string) (MoveInterface, error) {
	switch gameName {
	case "BatalhaDasDamas":
		var m Move
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil

	case "BatalhaDoChess":
		var m ChessMove
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil

	default:
		return nil, fmt.Errorf("unknown game type: %s", gameName)
	}
}

type GameMovesRequest struct {
	GameID string `json:"game_id"`
}

func MapPlayerToGamePlayer(player Player) GamePlayer {
	return GamePlayer{
		ID:        player.ID,
		Name:      player.Name,
		Token:     player.Token,
		SessionID: player.SessionID,
		Timer:     0,
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
	whiteID, _ := r.GetOpponentPlayerID(r.CurrentPlayerID)
	game := Game{
		ID:    r.ID,
		Board: NewBoard(r.CurrentPlayerID, whiteID, "std-game", r.OperatorIdentifier.GameName),
		//Board:           *NewBoard(r.CurrentPlayerID, whiteID, "two-pieces-endgame"),
		//Board:           *NewBoard(r.CurrentPlayerID, whiteID, "multiple-capture"),
		Players:            mapPlayers(r),
		CurrentPlayerID:    r.CurrentPlayerID,
		Turn:               0,
		Moves:              []MoveInterface{},
		StartTime:          time.Now(),
		Winner:             "",
		BetValue:           r.BetValue,
		TimerSetting:       config.Cfg.Services["gameworker"].TimerSetting,
		OperatorIdentifier: r.OperatorIdentifier,
	}

	if game.Players[0].ID == whiteID {
		game.Players[0].Color = "w"
		game.Players[1].Color = "b"
	} else {
		game.Players[0].Color = "b"
		game.Players[1].Color = "w"
	}
	game.SetUpPlayerTimers()
	game.UpdatePlayerPieces() // Set NumPieces for each player
	return &game
}

func (g *Game) SetUpPlayerTimers() {

	switch g.TimerSetting {
	case "reset":
		g.Players[0].Timer = config.Cfg.Services["gameworker"].Timer
		g.Players[1].Timer = g.Players[0].Timer

	case "cumulative":
		calculatedTimer := config.Cfg.Services["gameworker"].Timer * config.Cfg.Services["gameworker"].PiecesInMatch
		g.Players[0].Timer = calculatedTimer + 1
		g.Players[1].Timer = g.Players[0].Timer

	}
}

func (g *Game) CalcGameMaxTimer() (int, error) {
	switch g.TimerSetting {
	case "reset":
		return config.Cfg.Services["gameworker"].Timer, nil

	case "cumulative":
		return config.Cfg.Services["gameworker"].Timer * config.Cfg.Services["gameworker"].PiecesInMatch, nil
	}
	return 0, nil
}

func (g *Game) CountPlayerPieces(playerID string) int {
	count := 0
	for _, piece := range g.Board.GetPieces() {
		if piece != nil && piece.GetPlayerID() == playerID {
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

func (g *Game) GetGamePlayer(playerID string) (*GamePlayer, error) {
	if len(g.Players) != 2 {
		return nil, fmt.Errorf("invalid number of players in game")
	}

	for _, player := range g.Players {
		if player.ID == playerID {
			return &player, nil
		}
	}

	return nil, fmt.Errorf("player not found for player ID: %s", playerID)
}

func (g *Game) UpdatePlayerTimer(playerID string, timer int) error {
	if len(g.Players) != 2 {
		return fmt.Errorf("invalid number of players in game")
	}

	for i := range g.Players {
		if g.Players[i].ID == playerID {
			g.Players[i].Timer = timer
			return nil
		}
	}
	return fmt.Errorf("player not found for player ID: %s", playerID)
}

// Updates player id and turn count.
func (g *Game) NextPlayer() {
	nextPlayerId, err := g.GetOpponentPlayerID(g.CurrentPlayerID)
	if err != nil {
		log.Printf("Error NextPlayer getting opponent ID: %v\n", err)
	}
	g.CurrentPlayerID = nextPlayerId
	g.Turn += 1
}

func (g *Game) RemovePiece(pos string) {
	if _, exists := g.Board.GetPiece(pos); exists {
		g.Board.RemovePiece(pos)
	}
}

// TODO: THIS NEEDS TO BE MOVED TO THE BOARD, SINCE THE BOARD IS WHAT CHANGED EACH GAME IT SHOULD BE THE ONE TO MAKE THE MOVE!
func (g *Game) MovePiece(move MoveInterface) bool {

	// Validate move
	piece, exists := g.Board.GetPiece(move.GetFrom())
	if !exists {
		logger.Default.Warnf("invalid move: no piece exists at position %v", move.GetFrom())
		return false
	}
	if piece == nil {
		logger.Default.Warnf("invalid move: piece at %v is nil", move.GetFrom())
		return false
	}
	if piece.GetID() != move.GetPieceID() {
		logger.Default.Warnf("invalid move: piece ID %v != move piece ID %v",
			piece.GetID(), move.GetPieceID())
		return false
	}
	if piece.GetPlayerID() != move.GetPlayerID() {
		logger.Default.Warnf("invalid move: piece ID %v does not belong to player %v",
			piece.GetID(), move.GetPlayerID())
		return false
	}

	// Move piece to new position // TODO: This is new code. Handle error, check it works.
	g.Board.MovePiece(move.GetFrom(), move.GetTo())

	// Handle capture
	if move.IsCaptureMove() {
		var capturePos string
		if piece.IsPieceKinged() {
			// For kinged pieces, the captured piece is the last square before the landing position
			fromRow, fromCol := move.GetFrom()[0], move.GetFrom()[1]
			toRow, toCol := move.GetTo()[0], move.GetTo()[1]

			// Calculate the direction of movement
			rowStep := 1
			if toRow < fromRow {
				rowStep = -1
			}
			colStep := 1
			if toCol < fromCol {
				colStep = -1
			}

			// Calculate the position of the captured piece
			captureRow := toRow - byte(rowStep)
			captureCol := toCol - byte(colStep)
			capturePos = fmt.Sprintf("%c%c", captureRow, captureCol)
		} else {
			// For regular pieces, the captured piece is in the middle of the from and to positions
			midRow := (move.GetFrom()[0] + move.GetTo()[0]) / 2
			midCol := (move.GetFrom()[1] + move.GetTo()[1]) / 2
			capturePos = fmt.Sprintf("%c%c", midRow, midCol)
		}

		// Remove the captured piece
		g.Board.RemovePiece(capturePos)
	}
	return true
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

func (g *Game) FinishGame(winnerID string) {
	g.Winner = winnerID
	g.EndTime = time.Now()
}
