package gameworker

import (
	"context"
	"fmt"

	"github.com/Lavizord/checkers-server/models"
)

func (gw *GameWorker) BroadCastToGamePlayers(msg []byte, game models.Game) {
	gw.RedisClient.PublishToGamePlayer(game.Players[0], string(msg))
	gw.RedisClient.PublishToGamePlayer(game.Players[1], string(msg))
}

// TODO: Review stop channel key?
func (gw *GameWorker) PublishStopToTimerChannel(gameID string) {
	stopChannel := fmt.Sprintf("game:%s:stop_timer", gameID)
	gw.RedisClient.Client.Publish(context.Background(), stopChannel, "STOP") // Stop the old timer
}

// TODO: Review stop channel key?
func (gw *GameWorker) PublishSwitchToTimerChannel(gameID string) {
	switchChannel := fmt.Sprintf("game:%s:switch", gameID)
	gw.RedisClient.Client.Publish(context.Background(), switchChannel, "SWITCH") // This will let the timer know there was a change.
}
