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
	HandleMove(game *models.Game, move models.MoveInterface, player *models.Player, piece models.PieceInterface) error
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
	go gw.ProcessGameCreationList()
	go gw.ProcessGameMovesList()
	go gw.ProcessLeaveGameList()
	go gw.ProcessDisconnectFromGameList()
	go gw.ProcessReconnectFromGameList()
}

func (gw *GameWorker) GetGameName() string {
	return gw.GameName
}

// Method for handling a move message.
//
// Its responsible for handling the validation of the move and sending the necessary messages to
// the players, as well as advancing the updating the game.
func (gw *GameWorker) HandleMove(game *models.Game, move models.MoveInterface, player *models.Player, piece models.PieceInterface) error {
	panic("HandleMove must be implemented by embedding worker")
}

// Process elements in the create game message list. A room is serialzied into the list.
//
// This method validates and created a game, the game creation is based on the game of the room.
func (gw *GameWorker) ProcessGameCreationList() {
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

// Abstracting the games moves to be able to pass it the game worker.
func (gw *GameWorker) ProcessGameMovesList() {
	gw.ProcessMovesLoop(gw)
}

// Shared method to process the moves of the redis list.
//
// Does all the preliminary data validation, and calles the Worker.HandleMove, to
// process the move within the game-
func (gw *GameWorker) ProcessMovesLoop(w Worker) {
	listName := fmt.Sprintf("move_piece:{%v}", gw.GameName)
	for {
		moveData, err := gw.RedisClient.BLPopGeneric(listName, 0)
		if err != nil {
			logger.Default.Infof("(Process Game Moves) - Error retrieving move data: %v", err)
			continue
		}

		move, err := models.UnmarshalMove([]byte(moveData[1]), gw.GameName)
		if err != nil {
			logger.Default.Infof("(Process Game Moves) - JSON Unmarshal Error: %v", err)
			continue
		}

		player, err := gw.RedisClient.GetPlayer(move.GetPlayerID())
		if err != nil {
			player = gw.RedisClient.GetDisconnectedPlayerData(move.GetPlayerID())
			if player == nil {
				logger.Default.Errorf("(Process Game Moves) - failed to get data of player with id: %v", move.GetPlayerID())
				continue
			}
			logger.Default.Warnf("(Process Game Moves) - player retrieved from disconnected list: %v", move.GetPlayerID())
		}

		game, err := gw.RedisClient.GetGame(player.GameID)
		if err != nil {
			logger.Default.Errorf("(Process Game Moves) - failed to get game with id: %v, from player with id: %v", player.GameID, move.GetPlayerID())
			continue
		}
		if game.CurrentPlayerID != move.GetPlayerID() {
			logger.Default.Errorf("(Process Game Moves) - incorrect current player: %+v", move)
			continue
		}

		piece, _ := game.Board.GetPieceByID(move.GetPieceID())
		if piece == nil {
			logger.Default.Errorf("(Process Game Moves) - error getting piece from board")
			continue
		}

		// delegate to worker-specific move handler
		if err := w.HandleMove(game, move, player, piece); err != nil {
			logger.Default.Errorf("(Process Game Moves) - handle move error: %v", err)
			continue
		}
	}
}

func (gw *GameWorker) ProcessLeaveGameList() {
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

func (gw *GameWorker) ProcessDisconnectFromGameList() {
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

func (gw *GameWorker) ProcessReconnectFromGameList() {
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
