package damasworker

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

type DamasWorker struct {
	*gameworker.GameWorker
}

func New(redis *redisdb.RedisClient, db *postgrescli.PostgresCli) *DamasWorker {
	return &DamasWorker{&gameworker.GameWorker{RedisClient: redis, Db: db, GameName: "BatalhaDasDamas"}}
}

// Process elements in the create game message list. A room is serialzied into the list.
//
// This method validates and created a game, the game creation is based on the game of the room.
func (gw *DamasWorker) ProcessGameCreationMessages() {
	listName := fmt.Sprintf("create_game:{%v}", gw.GameName)
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

// Does some checks to see if the move is valid.
//
// TODO: This needs to be adapted acording to the game. Maybe implement it in the game object.
func (gw *DamasWorker) ValidMove(game *models.Game, move models.Move, piece *models.DamasPiece) bool {
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
	if piece.IsKinged {
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
