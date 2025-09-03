package chessworker

import (
	"encoding/json"
	"fmt"
	"log"

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
	go cw.ProcessGameCreationMessages() // ChessWorker’s version
	go cw.ProcessGameMoves()            // ChessWorker’s version
	go cw.ProcessLeaveGame()
	go cw.ProcessDisconnectFromGame()
	go cw.ProcessReconnectFromGame()
}

// Process elements in the create game message list. A room is serialzied into the list.
//
// This method validates and created a game, the game creation is based on the game of the room.
func (gw *ChessWorker) ProcessGameCreationMessages() {
	listName := fmt.Sprintf("create_game:{%v}", gw.GameName)
	logger.Default.Infof("starting processing chessworker queue: %v", listName)
	for {
		roomData, err := gw.RedisClient.BLPopGeneric(listName, 0) // Block
		if err != nil {
			logger.Default.Errorf("(Process Game Creation) - error retrieving room data to create a game: %v", err)
			continue
		}
		if len(roomData) < 2 {
			logger.Default.Errorf("(Process Game Creation) - Unexpected BLPop result: %v", roomData)
			continue
		}

		var room models.Room
		err = json.Unmarshal([]byte(roomData[1]), &room) // Extract second element
		if err != nil {
			logger.Default.Errorf("(Process Game Creation) - JSON Unmarshal Error: %v", err)
			continue
		}

		player1, err := gw.RedisClient.GetPlayer(room.Player1.ID)
		if err != nil {
			player1 = gw.RedisClient.GetDisconnectedInQueuePlayerData(room.Player1.ID)
			logger.Default.Infof("(Process Game Creation) player1 with id: %v retrieved from offline list:")
		}
		player2, err := gw.RedisClient.GetPlayer(room.Player2.ID)
		if err != nil {
			player2 = gw.RedisClient.GetDisconnectedInQueuePlayerData(room.Player2.ID)
			logger.Default.Infof("(Process Game Creation) player2 with id: %v retrieved from offline list:")
		}
		logger.Default.Infof("(Process Game Creation) starting game for players with id: %v and : %v", player1.ID, player2.ID)
		game := room.NewGame()
		// we need to update our players with a game ID.
		player1.GameID = game.ID
		player2.GameID = game.ID
		// We then reset the room id
		player1.RoomID = ""
		player2.RoomID = ""
		// we also change the player status.
		player1.UpdatePlayerStatus(models.StatusInGame)
		player2.UpdatePlayerStatus(models.StatusInGame)

		err = gw.RedisClient.AddGame(game)
		gw.RedisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(room.ID))
		msg, err := messages.GenerateGameStartMessage(*game)
		//log.Printf("[%s-%d] - (Process Game Creation) - Message to publish: %v\n", name, pid, string(msg))
		gw.BroadCastToGamePlayers(msg, *game)
		// Finnally save stuff to redis.
		err = gw.RedisClient.UpdatePlayer(player1)
		if err != nil {
			// since the player is offline, we will move the player over to the offline list.
			gw.RedisClient.SaveDisconnectSessionPlayerData(*player1, *game)
			gw.RedisClient.DeleteDisconnectedInQueuePlayerData(player2.ID)
			// we also notify the opponent that this player is offline.
			msg, _ := messages.NewMessage("opponent_disconnected_game", "disconnected")
			opponent, _ := game.GetGamePlayer(player2.ID)
			gw.RedisClient.PublishToGamePlayer(*opponent, string(msg))
		}
		err = gw.RedisClient.UpdatePlayer(player2)
		if err != nil {
			// since the player is offline, we will move the player over to the offline list.
			gw.RedisClient.SaveDisconnectSessionPlayerData(*player2, *game)
			gw.RedisClient.DeleteDisconnectedInQueuePlayerData(player1.ID)
			// we also notify the opponent that this player is offline.
			msg, _ := messages.NewMessage("opponent_disconnected_game", "disconnected")
			opponent, _ := game.GetGamePlayer(player1.ID)
			gw.RedisClient.PublishToGamePlayer(*opponent, string(msg))
		}
		logger.Default.Infof("(Process Game Creation) game started for players with id: %v and : %v", player1.ID, player2.ID)
		go gw.StartTimer(game) // Start turn timer
	}
}

// TODO: Finish chess implementation.
func (gw *ChessWorker) ValidMove(game *models.Game, move models.Move, piece models.PieceInterface) bool {
	if piece == nil {
		log.Println("Error: piece is nil")
		return false
	}
	return true
}

// Checks and processes game move messages.
func (gw *ChessWorker) ProcessGameMoves() {
	listName := fmt.Sprintf("move_piece:{%v}", gw.GameName)
	for {
		moveData, err := gw.RedisClient.BLPopGeneric(listName, 0) // Block
		if err != nil {
			logger.Default.Infof("(Process Game Moves) - Error retrieving move data: %v", err)
			continue
		}

		// We start by getting our move data, player, game and opponentPlayer.
		var move models.Move
		err = json.Unmarshal([]byte(moveData[1]), &move) // Extract second element
		if err != nil {
			logger.Default.Infof("(Process Game Moves) - JSON Unmarshal Error: %v", err)
			continue
		}
		player, err := gw.RedisClient.GetPlayer(move.PlayerID)
		if err != nil {
			player = gw.RedisClient.GetDisconnectedPlayerData(move.PlayerID)
			if player == nil {
				logger.Default.Errorf("(Process Game Moves) - failed to get data of player with id: %v", move.PlayerID)
				continue
			}
			logger.Default.Warnf("(Process Game Moves) - player retrieved from disconnected list: %v", move.PlayerID)
		}
		game, err := gw.RedisClient.GetGame(player.GameID)
		if err != nil {
			logger.Default.Errorf("(Process Game Moves) - failed to get game with id: %v, from player with id: %v", player.GameID, move.PlayerID)
			continue
		}
		if game.CurrentPlayerID != move.PlayerID {
			logger.Default.Errorf("(Process Game Moves) - incorrect current player to process move: %+v from game with id: %v, from player with id: %v", move, player.GameID, move.PlayerID)
			continue
		}

		piece, _ := game.Board.GetPieceByID(move.PieceID)
		if !gw.ValidMove(game, move, piece) {
			logger.Default.Errorf("(Process Game Moves) - invalid move: %+v from game with id: %v, from player with id: %v", move, player.GameID, move.PlayerID)
			boardState, _ := messages.GenerateGameBoardState(*game)
			msginv, _ := messages.NewMessage("invalid_move", boardState)
			gw.RedisClient.PublishToPlayer(*player, string(msginv))
			continue
		}
		// We move our piece.
		if !game.MovePiece(move) {
			logger.Default.Errorf("(Process Game Moves) - invalid move, board missmatch: %+v from game with id: %v, from player with id: %v", move, player.GameID, move.PlayerID)
			boardState, _ := messages.GenerateGameBoardState(*game)
			msginv, _ := messages.NewMessage("invalid_move", boardState)
			gw.RedisClient.PublishToPlayer(*player, string(msginv))
			continue
		}
		game.UpdatePlayerPieces()
		move.IsKinged = game.Board.WasPieceKinged(move.To, piece)
		if move.IsKinged {
			piece.SetIsPieceKinged(move.IsKinged)
		}

		// We send the message to the opponent player.
		msg, err := messages.GenerateMoveMessage(move)
		if err != nil {
			logger.Default.Errorf("(Process Game Moves) - failed to generate move message: %+v, from game with id: %v, from player with id: %v", move, player.GameID, move.PlayerID)
		}
		//log.Printf("[%s-%d] - (Process Game Moves) - Message to publish: %v\n", name, pid, string(msg))
		opponent, _ := game.GetOpponentGamePlayer(move.PlayerID)
		gw.RedisClient.PublishToGamePlayer(*opponent, string(msg))

		// Since the move was validated and passed to the other player, its time to check for our end turn / end game conditions.
		// This means we can add the move to our game.
		game.Moves = append(game.Moves, move)

		// We check for game Over
		if game.CheckGameOver() {
			logger.Default.Infof("(Process Game Moves) - determined game is over, from game with id: %v, from player with id: %v", player.GameID, move.PlayerID)
			gw.HandleGameEnd(game, "winner", move.PlayerID)
			continue
		}
		// We check for a capture.
		if !move.IsCapture {
			logger.Default.Infof("(Process Game Moves) - move is not a capture changing turn, from game with id: %v, from player with id: %v", player.GameID, move.PlayerID)
			gw.HandleTurnChange(game)
			continue
		}
		if move.IsCapture && !game.Board.CanPieceCaptureNEW(move.To) {
			logger.Default.Infof("(Process Game Moves) - move is capture and cant capture any more pieces, from game with id: %v, from player with id: %v", player.GameID, move.PlayerID)
			gw.HandleTurnChange(game)
			continue
		}
		if move.IsKinged {
			logger.Default.Infof("(Process Game Moves) - move is kinged, handling turn change, from game with id: %v, from player with id: %v", player.GameID, move.PlayerID)
			gw.HandleTurnChange(game)
			continue
		}
		gw.RedisClient.UpdateGame(game) // we update our game at the end. I guess this probably never happens
	}
}
