package gameworker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
)

func (gw *GameWorker) StartTimer(game *models.Game) {
	switch game.TimerSetting {
	case "reset":
		gw.StartResetEveryTurnTimer(game)
	case "cumulative":
		gw.StartCumulativeTimer(game)
	default:
		log.Printf("Invalid timer setting: %s for game %s", game.TimerSetting, game.ID)
	}
}

func (gw *GameWorker) StartResetEveryTurnTimer(game *models.Game) {
	ctx := context.Background()
	stopChannel := fmt.Sprintf("game:%s:stop_timer", game.ID)
	switchChannel := fmt.Sprintf("game:%s:switch", game.ID) // Channel to listen for switch events

	// Subscribe to both stop and switch channels
	pubsub := gw.RedisClient.Client.Subscribe(ctx, stopChannel, switchChannel)
	defer pubsub.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Fetch the base timer from the config.
	baseTimer := config.Cfg.Services["gameworker"].Timer

	// Find the player with Color == "b" to start the timer
	var activePlayerIndex int
	for i, player := range game.Players {
		if player.Color == "b" {
			activePlayerIndex = i
			break
		}
	}
	//activePlayer := game.Players[activePlayerIndex]	// This was used in the log prints.
	// Initialize the timer for the active player
	timer := baseTimer
	for {
		select {
		case <-ticker.C:
			// Decrement the timer
			timer--
			activePlayerTimer := timer

			// Publish the updated timer to both players
			msg, _ := messages.GenerateGameTimerMessage(*game, activePlayerTimer)
			if isEven(timer) {
				gw.RedisClient.PublishToGamePlayer(game.Players[0], string(msg))
				gw.RedisClient.PublishToGamePlayer(game.Players[1], string(msg))
			}

			// Check if the timer has expired
			if activePlayerTimer <= 0 {
				gw.HandleTurnChange(game)
				//log.Printf("Turn timer expired for player %s in game %s. Switching turns.\n", activePlayer.ID, game.ID)
			}

		case msg := <-pubsub.Channel():
			switch msg.Channel {
			case stopChannel:
				//log.Printf("Timer stopped for game %s\n", game.ID)
				return // Exit the function, stopping the timer

			case switchChannel:
				// Switch the active player when a move is made
				activePlayerIndex = 1 - activePlayerIndex // Toggle between 0 and 1
				//activePlayer = game.Players[activePlayerIndex] 	// This was used in the log prints.
				timer = baseTimer // Reset the timer for the new active player
				//log.Printf("Switched active player to %s in game %s. Timer reset to %d seconds.\n", activePlayer.ID, game.ID, timer)
			}
		}
	}
}

func (gw *GameWorker) StartCumulativeTimer(game *models.Game) {
	ctx := context.Background()
	stopChannel := fmt.Sprintf("game:%s:stop_timer", game.ID)
	switchChannel := fmt.Sprintf("game:%s:switch", game.ID) // Channel to listen for move events

	// Subscribe to both stop and move channels
	pubsub := gw.RedisClient.Client.Subscribe(ctx, stopChannel, switchChannel)
	defer pubsub.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Initialize timers for each player
	playerTimers := make(map[string]int) // Key: Player ID, Value: Remaining time
	for _, player := range game.Players {
		playerTimers[player.ID] = player.Timer
	}

	// Find the player with Color == "b" to start the timer
	var activePlayerIndex int
	for i, player := range game.Players {
		if player.Color == "b" {
			activePlayerIndex = i
			break
		}
	}
	activePlayer := game.Players[activePlayerIndex]

	for {
		select {
		case <-ticker.C:
			// Decrement the active player's timer
			playerTimers[activePlayer.ID]--
			activePlayerTimer := playerTimers[activePlayer.ID]

			// Publish the updated timer to both players
			game, err := gw.RedisClient.GetGame(game.ID)
			if err != nil {
				logger.Default.Errorf("error getting game with id: %v, for active player: %v, with err: %v", game.ID, activePlayer.ID, err.Error())
				continue
			}
			msg, _ := messages.GenerateGameTimerMessage(*game, activePlayerTimer)
			game.UpdatePlayerTimer(activePlayer.ID, activePlayerTimer)
			go gw.RedisClient.UpdateGame(game)
			go gw.RedisClient.PublishToGamePlayer(game.Players[0], string(msg))
			go gw.RedisClient.PublishToGamePlayer(game.Players[1], string(msg))

			// Check if the active player's timer has expired
			if activePlayerTimer <= 0 {
				// The other player wins
				winner := game.Players[1-activePlayerIndex].ID
				//log.Printf("Cumulative timer expired for player %s in game %s. Player %s wins.\n", activePlayer.ID, game.ID, winner)
				gw.HandleGameEnd(game, "timeout", winner)
				return
			}

		case msg := <-pubsub.Channel():
			switch msg.Channel {
			case stopChannel:
				//log.Printf("Timer stopped for game %s\n", game.ID)
				return // Exit the function, stopping the timer

			case switchChannel:
				// Switch the active player when a switch is sent
				activePlayerIndex = 1 - activePlayerIndex // Toggle between 0 and 1
				activePlayer = game.Players[activePlayerIndex]
				//log.Printf("Switched active player to %s in game %s\n", activePlayer.ID, game.ID)
			}
		}
	}
}

func isEven(n int) bool {
	return n&1 == 0 // Last bit = 0 → even
}
