package chessworker

import (
	"fmt"

	"github.com/Lavizord/checkers-server/gameworkers/gameworker"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

type ChessWorker struct {
	*gameworker.GameWorker
}

func New(redis *redisdb.RedisClient, db *postgrescli.PostgresCli) *ChessWorker {
	return &ChessWorker{&gameworker.GameWorker{RedisClient: redis, Db: db, GameName: "BatalhaDoChess"}}
}

func (cw *ChessWorker) Run() {
	go cw.ProcessGameCreationList() // ChessWorker’s version
	go cw.ProcessGameMovesList()    // ChessWorker’s version
	go cw.ProcessLeaveGameList()
	go cw.ProcessDisconnectFromGameList()
	go cw.ProcessReconnectFromGameList()
}

// Checks and processes game move messages.
//func (cw *ChessWorker) ProcessGameMovesList() {
//	listName := fmt.Sprintf("move_piece:{%v}", cw.GameName)
//	for {
//		moveData, err := cw.RedisClient.BLPopGeneric(listName, 0) // Block
//		if err != nil {
//			logger.Default.Infof("(Process Game Moves) - Error retrieving move data: %v", err)
//			continue
//		}
//
//		// We start by getting our move data, player, game and opponentPlayer.
//		var move models.MoveInterface
//		err = json.Unmarshal([]byte(moveData[1]), &move) // Extract second element
//		if err != nil {
//			logger.Default.Infof("(Process Game Moves) - JSON Unmarshal Error: %v", err)
//			continue
//		}
//		player, err := cw.RedisClient.GetPlayer(move.GetPlayerID())
//		if err != nil {
//			player = cw.RedisClient.GetDisconnectedPlayerData(move.GetPlayerID())
//			if player == nil {
//				logger.Default.Errorf("(Process Game Moves) - failed to get data of player with id: %v", move.GetPlayerID())
//				continue
//			}
//			logger.Default.Warnf("(Process Game Moves) - player retrieved from disconnected list: %v", move.GetPlayerID())
//		}
//		game, err := cw.RedisClient.GetGame(player.GameID)
//		if err != nil {
//			logger.Default.Errorf("(Process Game Moves) - failed to get game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
//			continue
//		}
//		if game.CurrentPlayerID != move.GetPlayerID() {
//			logger.Default.Errorf("(Process Game Moves) - incorrect current player to process move: %+v from game with id: %v, from player with id: %v", move, player.GameID, move.GetPlayerID())
//			continue
//		}
//
//		// TODO: THIS WHOLE THING SHOULD BE HANDLED BY THE BOARD I FEEL.
//		//piece, _ := game.Board.GetPieceByID(move.GetPieceID())
//		//if !gw.ValidMove(game, move, piece) {
//		//	logger.Default.Errorf("(Process Game Moves) - invalid move: %+v from game with id: %v, from player with id: %v", move, player.GameID, move.GetPlayerID())
//		//	boardState, _ := messages.GenerateGameBoardState(*game)
//		//	msginv, _ := messages.NewMessage("invalid_move", boardState)
//		//	gw.RedisClient.PublishToPlayer(*player, string(msginv))
//		//	continue
//		//}
//		// We move our piece.
//		//if !game.MovePiece(move) { // TODO: THIS NEEDS TO BE MOVED TO THE BOARD.MovePiece, SINCE THE BOARD IS WHAT CHANGED EACH GAME IT SHOULD BE THE ONE TO MAKE THE MOVE!
//		//	logger.Default.Errorf("(Process Game Moves) - invalid move, board missmatch: %+v from game with id: %v, from player with id: %v", move, player.GameID, move.GetPlayerID())
//		//	boardState, _ := messages.GenerateGameBoardState(*game)
//		//	msginv, _ := messages.NewMessage("invalid_move", boardState)
//		//	gw.RedisClient.PublishToPlayer(*player, string(msginv))
//		//	continue
//		//}
//		//game.UpdatePlayerPieces()
//		//move.SetIsKingedMove(game.Board.WasPieceKinged(move.GetTo(), piece))
//		//if move.IsKingedMove() {
//		//	piece.SetIsPieceKinged(move.IsKingedMove())
//		//}
//
//		// We send the message to the opponent player.
//		msg, err := messages.GenerateMoveMessage(move)
//		if err != nil {
//			logger.Default.Errorf("(Process Game Moves) - failed to generate move message: %+v, from game with id: %v, from player with id: %v", move, player.GameID, move.GetPlayerID())
//		}
//		//log.Printf("[%s-%d] - (Process Game Moves) - Message to publish: %v\n", name, pid, string(msg))
//		opponent, _ := game.GetOpponentGamePlayer(move.GetPlayerID())
//		cw.RedisClient.PublishToGamePlayer(*opponent, string(msg))
//
//		// Since the move was validated and passed to the other player, its time to check for our end turn / end game conditions.
//		// This means we can add the move to our game.
//		game.Moves = append(game.Moves, move)
//
//		// We check for game Over
//		//if game.CheckGameOver() {
//		//	logger.Default.Infof("(Process Game Moves) - determined game is over, from game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
//		//	gw.HandleGameEnd(game, "winner", move.GetPlayerID())
//		//	continue
//		//}
//		// We check for a capture.
//		//if !move.IsCaptureMove() {
//		//	logger.Default.Infof("(Process Game Moves) - move is not a capture changing turn, from game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
//		//	gw.HandleTurnChange(game)
//		//	continue
//		//}
//		//if move.IsCaptureMove() && !game.Board.CanPieceCaptureNEW(move.GetTo()) {
//		//	logger.Default.Infof("(Process Game Moves) - move is capture and cant capture any more pieces, from game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
//		//	gw.HandleTurnChange(game)
//		//	continue
//		//}
//		//if move.IsKingedMove() {
//		//	logger.Default.Infof("(Process Game Moves) - move is kinged, handling turn change, from game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
//		//	gw.HandleTurnChange(game)
//		//	continue
//		//}
//		cw.HandleTurnChange(game)
//		cw.RedisClient.UpdateGame(game) // we update our game at the end. I guess this probably never happens
//	}
//}

func (dw *ChessWorker) HandleMove(game *models.Game, move models.MoveInterface, player *models.Player, piece models.PieceInterface) error {
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

	return fmt.Errorf("not implemented")
}
