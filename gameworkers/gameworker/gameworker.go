package gameworker

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

type Worker interface {
	Run()
	GetGameName() string
}

type GameWorker struct {
	RedisClient      *redisdb.RedisClient
	Db               *postgrescli.PostgresCli
	GameName         string
	QueueBetAmmounts []float64
}

// Processes a set of redis queues and
func New(redis *redisdb.RedisClient, db *postgrescli.PostgresCli, gn string) *GameWorker {
	return &GameWorker{
		RedisClient: redis,
		GameName:    gn,
		Db:          db,
	}
}

func (gw *GameWorker) Run() {
	go gw.ProcessGameCreationMessages()
	go gw.ProcessGameMoves()
	go gw.ProcessLeaveGame()
	go gw.ProcessDisconnectFromGame()
	go gw.ProcessReconnectFromGame()
}

func (gw *GameWorker) GetGameName() string {
	return gw.GameName
}

// Process elements in the create game message list. A room is serialzied into the list.
//
// This method validates and created a game, the game creation is based on the game of the room.
func (gw *GameWorker) ProcessGameCreationMessages() {
	listName := fmt.Sprintf("create_game:{%v}", gw.GameName)
	logger.Default.Infof("starting processing gameworker queue: %v", listName)

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

// Checks and processes game move messages.
//
// TODO: This should be moved into a game pub sub, since we already have game timers running.
// doing so will allow us to better controll the flow of the game.
func (gw *GameWorker) ProcessGameMoves() {
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

func (gw *GameWorker) ProcessLeaveGame() {
	listName := fmt.Sprintf("leave_game:{%v}", gw.GameName)

	for {
		// Block until there is a game over message
		playerData, err := gw.RedisClient.BLPop(listName, 0)
		if err != nil {
			logger.Default.Errorf("(Process Leave Game) - Error retrieving player data from leave game queue: %v", err)
			continue
		}
		logger.Default.Infof("(Process Leave Game) - processing leave game for player: %v, from game with id", playerData.ID, playerData.GameID)
		playerData, err = gw.RedisClient.GetPlayer(playerData.ID)
		if err != nil {
			logger.Default.Errorf("(Process Leave Game) - error fetching playerData: %v, from game with id", playerData.ID, playerData.GameID)
			continue
		}
		game, err := gw.RedisClient.GetGame(playerData.GameID)
		if err != nil {
			logger.Default.Errorf("(Process Leave Game) - error retrieving game: %v, for player with id: %v", playerData.GameID, playerData.ID)
			continue
		}
		winnrID, _ := game.GetOpponentPlayerID(playerData.ID)
		gw.HandleGameEnd(game, "player_left", winnrID)
	}
}

func (gw *GameWorker) ProcessDisconnectFromGame() {
	listName := fmt.Sprintf("disconnect_game:{%v}", gw.GameName)
	for {
		// Block until there is a game over message
		playerData, err := gw.RedisClient.BLPop(listName, 0)
		if err != nil {
			logger.Default.Errorf("(processDisconnectFromGame) - error retrieving data from disconnect_game queue: %v", err)
			continue
		}
		game, err := gw.RedisClient.GetGame(playerData.GameID)
		if err != nil {
			logger.Default.Errorf("(processDisconnectFromGame) - error retrieving game with id: %v, from player: %v, with err: %v", playerData.GameID, playerData.ID, err)
			continue
		}
		logger.Default.Infof("(processDisconnectFromGame) - processing disconnect from game game with id: %v, from player: %v", playerData.GameID, playerData.ID)
		gw.RedisClient.SaveDisconnectSessionPlayerData(*playerData, *game)
		gamePlayer, _ := game.GetGamePlayer(playerData.ID)
		// Now we notify the other player that this happened
		msg, _ := messages.NewMessage("opponent_disconnected_game", "disconnected")
		opponent, _ := game.GetOpponentGamePlayer(gamePlayer.ID)
		gw.RedisClient.PublishToGamePlayer(*opponent, string(msg))
	}
}

// TODO:  Review / refactor
func (gw *GameWorker) ProcessReconnectFromGame() {
	listName := fmt.Sprintf("reconnect_game:{%v}", gw.GameName)
	for {
		// Block until there is a game over message
		playerData, err := gw.RedisClient.BLPop(listName, 0)
		if err != nil {
			log.Printf("(Process Reconnect Game) - Error retrieving player data from queue: %v", err)
			continue
		}
		log.Printf("[%s-%d]  (Process Reconnect Game) - Processing the reconnect game: %+v", playerData)
		game, err := gw.RedisClient.GetGame(playerData.GameID)
		if err != nil {
			log.Printf("(Process Reconnect Game) - Error retrieving game data from redis: %v", err)
			continue
		}
		// We send a message to the reconnected player with the board state.
		msg, err := messages.GenerateGameReconnectMessage(*game)
		if err != nil {
			log.Printf("(Process Reconnect Game) - Error generating game reconnect message: %v", err)
			continue
		}
		err = gw.RedisClient.PublishToPlayer(*playerData, string(msg))
		if err != nil {
			log.Printf("(Process Reconnect Game) - Error publishing game reconnect message: %v", err)
			continue
		}
		// We notify the opponent that the player reconnected.
		opponent, _ := game.GetOpponentGamePlayer(playerData.ID)
		msg, _ = messages.NewMessage("opponent_disconnected_game", "reconnected")
		gw.RedisClient.PublishToGamePlayer(*opponent, string(msg))
		gw.RedisClient.DeleteDisconnectedPlayerSession(playerData.SessionID)
	}
}

// Cleans up any redis data on disconected players of a certain game.
//
// Usually used when the game ends.
func (gw *GameWorker) CleanUpGameDisconnectedPlayers(game models.Game) {
	p1SessionId := game.Players[0].SessionID
	p2SessionId := game.Players[1].SessionID

	discPlayer1 := gw.RedisClient.GetDisconnectedPlayerData(p1SessionId)
	if discPlayer1 != nil {
		gw.RedisClient.DeleteDisconnectedPlayerSession(p1SessionId)
		gw.RedisClient.RemovePlayer(discPlayer1.ID)
	}
	discPlayer2 := gw.RedisClient.GetDisconnectedPlayerData(p2SessionId)
	if discPlayer2 != nil {
		gw.RedisClient.DeleteDisconnectedPlayerSession(p2SessionId)
		gw.RedisClient.RemovePlayer(discPlayer2.ID)
	}
}

// Does some checks to see if the move is valid.
//
// TODO: This needs to be adapted acording to the game. Maybe implement it in the game object.
func (gw *GameWorker) ValidMove(game *models.Game, move models.Move, piece models.PieceInterface) bool {
	if piece == nil {
		log.Println("Error: piece is nil")
		return false
	}
	capturers := game.Board.PiecesThatCanCapture(game.CurrentPlayerID)
	// If captures are available, and this piece can't capture, reject the move
	if len(capturers) > 0 {
		canCapture := false
		for _, p := range capturers {
			if p.GetID() == piece.GetID() {
				canCapture = true
				break
			}
		}
		if !canCapture {
			log.Println("Error: there are player pieces that can capture, must move one of those.")
			return false
		}
	}

	var valid bool
	var err error
	game.Board.PiecesThatCanCapture(game.CurrentPlayerID)
	if piece.IsPieceKinged() {
		valid, err = game.Board.IsValidMoveKing(move)
	} else {
		valid, err = game.Board.IsValidMove(move)
	}
	if err != nil {
		log.Printf("Error: %v", err)
		return false
	}
	if !valid {
		return false
	}
	return true
}
