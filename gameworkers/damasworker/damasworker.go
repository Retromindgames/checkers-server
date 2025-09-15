package damasworker

import (
	"fmt"

	"github.com/Lavizord/checkers-server/gameworkers/gameworker"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

type DamasWorker struct {
	*gameworker.GameWorker
}

func New(redis *redisdb.RedisClient, db *postgrescli.PostgresCli) *DamasWorker {
	return &DamasWorker{&gameworker.GameWorker{RedisClient: redis, Db: db, GameName: "BatalhaDasDamas"}}
}

func (dw *DamasWorker) Run() {
	go dw.ProcessGameCreationList()
	go dw.ProcessGameMovesList()
	go dw.ProcessLeaveGameList()
	go dw.ProcessDisconnectFromGameList()
	go dw.ProcessReconnectFromGameList()
}

func (cw *DamasWorker) ProcessGameMovesList() {
	cw.ProcessMovesLoop(cw)
}

func (dw *DamasWorker) HandleMove(game *models.Game, move models.MoveInterface, player *models.Player, piece models.PieceInterface) error {

	_, err := game.Board.ValidateMove(move, piece)
	if err != nil {
		msg := fmt.Errorf("(Process Game Moves) - invalid move: %+v from game with id: %v, from player with id: %v", move, player.GameID, move.GetPlayerID())
		logger.Default.Errorf(msg.Error())
		boardState, _ := messages.GenerateGameBoardState(*game)
		msginv, _ := messages.NewMessage("invalid_move", boardState)
		dw.RedisClient.PublishToPlayer(*player, string(msginv))
		return msg
		//continue
	}
	// We move our piece.
	if !game.MovePiece(move) {
		msg := fmt.Errorf("(Process Game Moves) - invalid move, board missmatch: %+v from game with id: %v, from player with id: %v", move, player.GameID, move.GetPlayerID())
		logger.Default.Errorf(msg.Error())
		boardState, _ := messages.GenerateGameBoardState(*game)
		msginv, _ := messages.NewMessage("invalid_move", boardState)
		dw.RedisClient.PublishToPlayer(*player, string(msginv))
		return msg
	}
	game.UpdatePlayerPieces()
	move.SetIsKingedMove(game.Board.WasPieceKinged(move.GetTo(), piece))
	if move.IsKingedMove() {
		piece.SetIsPieceKinged(move.IsKingedMove())
	}

	// We send the message to the opponent player.
	msg, err := messages.GenerateMoveMessage(move)
	if err != nil {
		logger.Default.Errorf("(Process Game Moves) - failed to generate move message: %+v, from game with id: %v, from player with id: %v", move, player.GameID, move.GetPlayerID())
	}
	//log.Printf("[%s-%d] - (Process Game Moves) - Message to publish: %v\n", name, pid, string(msg))
	opponent, _ := game.GetOpponentGamePlayer(move.GetPlayerID())
	dw.RedisClient.PublishToGamePlayer(*opponent, string(msg))

	// Since the move was validated and passed to the other player, its time to check for our end turn / end game conditions.
	// This means we can add the move to our game.
	game.Moves = append(game.Moves, move)

	// We check for game Over
	if game.CheckGameOver() {
		msg := fmt.Sprintf("(Process Game Moves) - determined game is over, from game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
		logger.Default.Infof(msg)
		dw.HandleGameEnd(game, "winner", move.GetPlayerID())
		return nil
	}
	// We check for a capture.
	if !move.IsCaptureMove() {
		msg := fmt.Sprintf("(Process Game Moves) - move is not a capture changing turn, from game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
		logger.Default.Infof(msg)
		dw.HandleTurnChange(game)
		return nil
	}
	if move.IsCaptureMove() && !game.Board.CanPieceCaptureNEW(move.GetTo()) {
		msg := fmt.Errorf("(Process Game Moves) - move is capture and cant capture any more pieces, from game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
		logger.Default.Infof(msg.Error())
		dw.HandleTurnChange(game)
		return nil

	}
	if move.IsKingedMove() {
		msg := fmt.Errorf("(Process Game Moves) - move is kinged, handling turn change, from game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
		logger.Default.Infof(msg.Error())
		dw.HandleTurnChange(game)
		return nil

	}
	dw.RedisClient.UpdateGame(game) // we update our game at the end. I guess this probably never happens
	return nil
}
