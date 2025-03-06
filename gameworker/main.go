package main

import (
	"checkers-server/config"
	"checkers-server/messages"
	"checkers-server/models"
	"checkers-server/redisdb"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

var pid int
var redisClient *redisdb.RedisClient
var name = "GameWorker"

func init() {
	pid = os.Getpid()
	config.LoadConfig()
	redisAddr := config.Cfg.Redis.Addr
	client, err := redisdb.NewRedisClient(redisAddr)
	if err != nil {
		log.Fatalf("[%s-Redis] Error initializing Redis client: %v\n", name, err)
	}
	redisClient = client
}

func main() {
	fmt.Printf("[%s-%d] - Waiting for Game messages...\n", name, pid)
	go processGameCreation()
	go processGameMoves()
	go processGameOverQueue()
	go processLeaveGame()
	select {}
}

func processGameCreation() {
	for {
		roomData, err := redisClient.BLPopGeneric("create_game", 0) // Block
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Creation) - Error retrieving room data: %v\n", name, pid, err)
			continue
		}

		if len(roomData) < 2 {
			fmt.Printf("[%s-%d] - (Process Game Creation) - Unexpected BLPop result: %+v\n", name, pid, roomData)
			continue
		}
		fmt.Printf("[%s-%d] - (Process Game Creation) - create game!: %+v\n", name, pid, roomData)

		var room models.Room
		err = json.Unmarshal([]byte(roomData[1]), &room) // Extract second element
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Creation) - JSON Unmarshal Error: %v\n", name, pid, err)
			continue
		}

		player1, err := redisClient.GetPlayer(room.Player1.ID)
		player2, err := redisClient.GetPlayer(room.Player2.ID)
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
		// Finnally save stuff to redis.
		err = redisClient.UpdatePlayer(player1)
		err = redisClient.UpdatePlayer(player2)
		err = redisClient.AddGame(game)
		msg, err := messages.GenerateGameStartMessage(*game)

		fmt.Printf("[%s-%d] - (Process Game Creation) - Message to publish: %v\n", name, pid, string(msg))
		BroadCastToGamePlayers(msg, *game)
		go startTurnTimer(game) // Start turn timer
	}
}

func processGameMoves() {
	for {
		moveData, err := redisClient.BLPopGeneric("move_piece", 0) // Block
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Moves) - Error retrieving move data: %v\n", name, pid, err)
			continue
		}
		fmt.Printf("[%s-%d] - (Process Game Moves) - processing move DATA!: %+v\n", name, pid, moveData)

		// We start by getting our move data, player, game and opponentPlayer.
		var move models.Move
		err = json.Unmarshal([]byte(moveData[1]), &move) // Extract second element
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Moves) - JSON Unmarshal Error: %v\n", name, pid, err)
			continue
		}
		player, err := redisClient.GetPlayer(move.PlayerID)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Moves) - Failed to get player!: %v\n", name, pid, err)
			continue
		}
		game, err := redisClient.GetGame(player.GameID)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Moves) - Failed to get game!: %v\n", name, pid, err)
			continue
		}
		opponent, err := game.GetOpponentGamePlayer(move.PlayerID)

		// We move our piece.
		game.MovePiece(move)
		game.UpdatePlayerPieces()
		// We send the message to the opponent player.
		msg, err := messages.GenerateMoveMessage(move)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Moves) - Failed to generate message: %v\n", name, pid, string(msg))
		}
		fmt.Printf("[%s-%d] - (Process Game Moves) - Message to publish: %v\n", name, pid, string(msg))
		redisClient.PublishToGamePlayer(*opponent, string(msg))

		// We check for game Over
		if game.CheckGameOver() {
			redisClient.Client.RPush(context.Background(), "game_over_queue", game.ID)
			redisClient.UpdateGame(game) // we update our game
			continue
		}
		if !move.IsCapture {
			handleTurnChange(game)
			redisClient.UpdateGame(game)
			continue
		}

		// we check for a turn change.
		if move.IsCapture && !game.Board.CanPieceCapture(move.To) {
			handleTurnChange(game)
		}
		redisClient.UpdateGame(game) // we update our game at the end.
	}
}

func processGameOverQueue() {
	for {
		// Block until there is a game over message
		gameOverData, err := redisClient.Client.BLPop(context.Background(), 0, "game_over_queue").Result()
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Over) - Error retrieving game over data: %v\n", name, pid, err)
			continue
		}

		if len(gameOverData) < 2 {
			fmt.Printf("[%s-%d] - (Process Game Over) - Unexpected BLPop result", name, pid)
			continue
		}
		gameId := gameOverData[1]                                                                   // Get the message
		fmt.Printf("[%s-%d] - (Process Game Over) - Processing game over: %s\n", name, pid, gameId) // this should be a game ID.
		stopGameTimer(gameId)

		// Now we handle game logic.
		game, err := redisClient.GetGame(gameId)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Moves) - Failed to get game!: %v\n", name, pid, err)
			continue
		}
		game.FinishGame() // This should handle our data side of things.
		msg, err := messages.GenerateGameOverMessage("winner", *game)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Over) - Failed to get game!: %v\n", name, pid, err)
			continue
		}
		redisClient.PublishToGamePlayer(*&game.Players[0], string(msg))
		redisClient.PublishToGamePlayer(*&game.Players[1], string(msg))

		// Now we will process palyer balance for the winner.
		winnerPlayer, err := redisClient.GetPlayer(game.Winner)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Over) - Failed to get winner player!: %v\n", name, pid, err)
			continue
		}
		// Update status and game Id of both players
		winnerPlayer.GameID = ""
		winnerPlayer.UpdatePlayerStatus(models.StatusOnline)
		winnerPlayer.UpdateBalance(game.BetValue * 1.75) // TODO: Move this to own function, maybe read from config?
		loserID, err := game.GetOpponentPlayerID(winnerPlayer.ID)
		loserPlayer, err := redisClient.GetPlayer(loserID)
		loserPlayer.GameID = ""
		loserPlayer.UpdatePlayerStatus(models.StatusOnline)

		msgP1, err := messages.NewMessage("balance_update", winnerPlayer.CurrencyAmount)
		redisClient.PublishPlayerEvent(winnerPlayer, string(msgP1))
		redisClient.UpdatePlayer(winnerPlayer)
		// since the game is Over, we remove it from redis.
		redisClient.RemoveGame(game.ID)
	}
}

func processLeaveGame() {
	for {
		// Block until there is a game over message
		playerData, err := redisClient.BLPop("leave_game", 0)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Leave Game) - Error retrieving player data from queue: %v\n", name, pid, err)
			continue
		}
		//playerData, err = redisClient.GetPlayer(playerData.ID)	// We cant do the get player, because it was already removed...
		fmt.Printf("[%s-%d] - Processing the leave game: %+v\n", name, pid, playerData)
		game, err := redisClient.GetGame(playerData.GameID)
		if err != nil {
			fmt.Printf("[%s-%d] - Error retrieving Game:%v\n", name, pid, err)
			continue
		}
		// if the game is over, lets stop the timers.
		stopGameTimer(playerData.GameID)

		// What happens when a player leaves the game?
		game.FinishGame()                                          // This should handle our data side of things.
		game.Winner, err = game.GetOpponentPlayerID(playerData.ID) // we gotta set our winner.
		msg, err := messages.GenerateGameOverMessage("winner", *game)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Over) - Failed to get game!: %v\n", name, pid, err)
			continue
		}
		redisClient.PublishToGamePlayer(*&game.Players[0], string(msg))
		redisClient.PublishToGamePlayer(*&game.Players[1], string(msg))

		// Now we will process palyer balance for the winner.
		winnerPlayer, err := redisClient.GetPlayer(game.Winner)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Over) - Failed to get winner player!: %v\n", name, pid, err)
			continue
		}
		// Update status and game Id of remaining player
		winnerPlayer.GameID = ""
		winnerPlayer.UpdatePlayerStatus(models.StatusOnline)
		winnerPlayer.UpdateBalance(game.BetValue * 1.75) // TODO: Move this to own function, maybe read from config?
		msgP1, err := messages.NewMessage("balance_update", winnerPlayer.CurrencyAmount)
		redisClient.PublishPlayerEvent(winnerPlayer, string(msgP1))
		redisClient.UpdatePlayer(winnerPlayer)
		// since the game is Over, we remove it from redis.
		if redisClient.RemoveGame(game.ID) != nil {
			fmt.Printf("[%s-%d] - (Process Game Over) - Failed to remove game!: %v\n", name, pid, err)
		} else {
			fmt.Printf("[%s-%d] - (Process Game Over) - Removed game!: %v\n", name, pid, err)
		}
	}
}

func handleTurnChange(game *models.Game) {
	stopGameTimer(game.ID)
	game.NextPlayer()
	msg, err := messages.NewMessage("turn_switch", game.CurrentPlayerID)
	if err != nil {
		fmt.Printf("[%s-%d] - (Handle Turn Change) - Failed to generate for turn change: %v\n", name, pid, msg)
	}
	BroadCastToGamePlayers(msg, *game)
	go startTurnTimer(game) // Start a fresh timer
}

func stopGameTimer(gameID string) {
	stopChannel := fmt.Sprintf("game:%s:stop_timer", gameID)
	redisClient.Client.Publish(context.Background(), stopChannel, "STOP") // Stop the old timer
}

func startTurnTimer(game *models.Game) {
	ctx := context.Background()
	stopChannel := fmt.Sprintf("game:%s:stop_timer", game.ID)
	pubsub := redisClient.Client.Subscribe(ctx, stopChannel)
	defer pubsub.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// fetch the timer from the config.
	timer := config.Cfg.Services["gameworker"].Timer
	for timer > 0 {
		select {
		case <-ticker.C:
			msg, _ := messages.GenerateGameTimerMessage(*game, timer)
			redisClient.PublishToGamePlayer(game.Players[0], string(msg))
			redisClient.PublishToGamePlayer(game.Players[1], string(msg))
			timer--

		case msg := <-pubsub.Channel():
			if msg.Payload == "STOP" {
				fmt.Printf("Timer stopped for game %s\n", game.ID)
				return // Exit the function, stopping the timer
			}
		}
	}
	handleTurnChange(game)
	redisClient.UpdateGame(game)
	fmt.Printf("Turn timer expired for game %s\n", game.ID)
}

func BroadCastToGamePlayers(msg []byte, game models.Game) {
	redisClient.PublishToGamePlayer(game.Players[0], string(msg))
	redisClient.PublishToGamePlayer(game.Players[1], string(msg))
}
