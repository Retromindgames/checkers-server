package gameworker

import (
	"log"

	"github.com/Lavizord/checkers-server/interfaces"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
)

// TODO: The logs in here should be reviewed. More info and logger.Default..

func (gw *GameWorker) HandleGameEnd(game *models.Game, reason string, winnerID string) {
	//log.Printf("Handling Game End for game [%v] - reason: [%v]", game.ID, reason)

	gw.PublishStopToTimerChannel(game.ID)
	game.FinishGame(winnerID)

	winAmount := interfaces.CalculateWinAmount(int64(game.BetValue*100), game.OperatorIdentifier.WinFactor)
	gameOverMsg, err := messages.GenerateGameOverMessage(reason, *game, winAmount)
	if err != nil {
		logger.Default.Errorf("[gameworker] - (Handle Game Over) - Failed to generate game over message!: %v", err)
		return
	}
	p1 := game.Players[0]
	p2 := game.Players[1]
	go gw.CleanUpGameDisconnectedPlayers(*game) // This was at the end, was moved up here, might make the reconect when the game is over more smooth...?
	go gw.HandleGameEndForPlayer(winnerID, game, p1, reason, winAmount, gameOverMsg)
	go gw.HandleGameEndForPlayer(winnerID, game, p2, reason, winAmount, gameOverMsg)

	// since the game is Over, we remove it from redis.
	if gw.RedisClient.RemoveGame(game.ID) != nil {
		log.Printf("[gameworker] - (Handle Game Over) - Failed to remove game!: %v", err)
	} else {
		log.Printf("[gameworker] - (Handle Game Over) - Removed game!: %v", err)
	}
	go gw.Db.SaveGame(*game, reason)
}

func (gw *GameWorker) HandleGameEndForPlayer(winnerID string, game *models.Game, gamePlayer models.GamePlayer, reason string, winAmount int64, gameOverMsg []byte) {
	var interfaceModule interfaces.OperatorInterface
	interfaceModule = interfaces.OperatorModules[game.OperatorIdentifier.OperatorName]
	var balanceUpdateMsg []byte

	// 1. The Winner needs to have a post to the wallet.
	if winnerID == gamePlayer.ID {
		// Get the session from the ID, since they share the same ID.
		winnerSession, err := gw.RedisClient.GetSessionByID(winnerID)
		if err != nil {
			log.Printf("[GameWorker] - error -> handleGameEnd - HandleGameEndForPlayer: fetching winner player session:%s\n", err)
			return
		}
		if winnerSession == nil {
			log.Printf("[GameWorker] - error -> handleGameEnd - HandleGameEndForPlayer: session id is nill!:%s\n", err)
			return
		}
		// we use our winner session here, because this way the winner will be payed out even if offline.
		var newBalance int64
		newBalance, _, err = interfaceModule.HandlePostWin(gw.Db, gw.RedisClient, *winnerSession, int64(game.BetValue*100), game.ID)
		if err != nil {
			log.Printf("[GameWorker] - Error posting the win :%s\n", err)
		} else {
			// We then generate the balance update message and send it over to the game Player. The player can be offline, but I guess the message just wont get delivered.
			// I could try to fetch the player from redis...? Is it worth it?... I fetch the player down the line...
			balanceUpdateMsg, _ = messages.NewMessage("balance_update", float64(newBalance)/100)
			gw.RedisClient.PublishToGamePlayer(gamePlayer, string(balanceUpdateMsg))
		}
	}

	// 2. Update player data, if it exists. If not prolly offline.
	player, err := gw.RedisClient.GetPlayer(gamePlayer.ID)
	if err != nil {
		log.Printf("(Handle Game Over) - processGameEndForPlayer - Failed to get player!: %v", err)
		return
	}
	if player == nil {
		log.Printf("(Handle Game Over) - processGameEndForPlayer - player is nill !: %v", err)
		return
	}
	player.GameID = ""
	player.UpdatePlayerStatus(models.StatusOnline)
	gw.RedisClient.UpdatePlayer(player)
	gw.RedisClient.PublishToGamePlayer(gamePlayer, string(gameOverMsg))
}

func (gw *GameWorker) HandleTurnChange(game *models.Game) {
	// publishStopToTimerChannel(game.ID)
	game.NextPlayer()
	gw.RedisClient.UpdateGame(game)
	msg, err := messages.NewMessage("turn_switch", game.CurrentPlayerID)
	if err != nil {
		log.Printf("(Handle Turn Change) - Failed to generate for turn change: %v", msg)
	}
	gw.BroadCastToGamePlayers(msg, *game)
	gw.PublishSwitchToTimerChannel(game.ID) // Start a fresh timer or switch player timer.
}
