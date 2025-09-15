package gameworker

import (
	"github.com/Lavizord/checkers-server/interfaces"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
)

func (gw *GameWorker) HandleGameEnd(game *models.Game, reason string, winnerID string) {
	//log.Printf("Handling Game End for game [%v] - reason: [%v]", game.ID, reason)

	gw.PublishStopToTimerChannel(game.ID)
	game.FinishGame(winnerID)

	winAmount := interfaces.CalculateWinAmount(int64(game.BetValue*100), game.OperatorIdentifier.WinFactor)
	gameOverMsg, err := messages.GenerateGameOverMessage(reason, *game, winAmount)
	if err != nil {
		logger.Default.Errorf("[gameworker] - (Handle Game Over) - Failed to generate game over message, for game: %v, with player1 session: %v and player2 session: %v, with err: %v", game.ID, game.Players[0].SessionID, game.Players[1].SessionID, err)
		return
	}
	p1 := game.Players[0]
	p2 := game.Players[1]
	go gw.CleanUpGameDisconnectedPlayers(*game)
	go gw.HandleGameEndForPlayer(winnerID, game, p1, reason, winAmount, gameOverMsg)
	go gw.HandleGameEndForPlayer(winnerID, game, p2, reason, winAmount, gameOverMsg)

	// since the game is Over, we remove it from redis.
	if gw.RedisClient.RemoveGame(game.ID) != nil {
		logger.Default.Errorf("failed to remove game with id: %s, for player 1 with session id: %v and player 2 with session id: %s, from redis, with err: %v", game.ID, game.Players[0].SessionID, game.Players[1].SessionID, err)
	}
	go gw.Db.SaveGame(*game, reason)
}

func (gw *GameWorker) HandleGameEndForPlayer(winnerID string, game *models.Game, gamePlayer models.GamePlayer, reason string, winAmount int64, gameOverMsg []byte) {
	interfaceModule := interfaces.OperatorModules[game.OperatorIdentifier.OperatorName]
	var balanceUpdateMsg []byte

	// 1. The Winner needs to have a post to the wallet.
	if winnerID == gamePlayer.ID {
		// Get the session from the ID, since they share the same ID.
		winnerSession, err := gw.RedisClient.GetSessionByID(winnerID)
		if err != nil {
			logger.Default.Errorf("failed to fetch the winner player session for game with id: %s, for player 1 with session id: %v and player 2 with session id: %s, from redis, with err: %v", game.ID, game.Players[0].SessionID, game.Players[1].SessionID, err)
			return
		}
		if winnerSession == nil {
			logger.Default.Errorf("session id of the winner is nil for game with id: %s, for player 1 with session id: %v and player 2 with session id: %s, from redis, with err: %v", game.ID, game.Players[0].SessionID, game.Players[1].SessionID, err)
			return
		}
		// we use our winner session here, because this way the winner will be payed out even if offline.
		var newBalance int64
		newBalance, _, err = interfaceModule.HandlePostWin(gw.Db, gw.RedisClient, *winnerSession, int64(game.BetValue*100), game.ID)
		if err != nil {
			logger.Default.Errorf("error posting the win to the api, for session: %s, for game with id: %s, for player 1 with session id: %v and player 2 with session id: %s, from redis, with err: %v", winnerSession, game.ID, game.Players[0].SessionID, game.Players[1].SessionID, err)
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
		logger.Default.Warnf("failed to get player to update data to redis, player with id: %s, for game with id: %s, for player 1 with session id: %v and player 2 with session id: %s, from redis, with err: %v", gamePlayer.ID, game.ID, game.Players[0].SessionID, game.Players[1].SessionID, err)
		return
	}
	if player == nil {
		logger.Default.Warnf("player to update data to redis is nil, player with id: %s, for game with id: %s, for player 1 with session id: %v and player 2 with session id: %s, from redis", gamePlayer.ID, game.ID, game.Players[0].SessionID, game.Players[1].SessionID)
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
		logger.Default.Errorf("failed to generate turn_swith message for game with id: %s, for player 1 with session id: %s, and player 2 with session id:%s from redis, with err: %v", game.ID, game.Players[0].SessionID, game.Players[1].SessionID, err)
	}
	gw.BroadCastToGamePlayers(msg, *game)
	gw.PublishSwitchToTimerChannel(game.ID) // Start a fresh timer or switch player timer.
}
