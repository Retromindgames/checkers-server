package main

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Lavizord/checkers-server/interfaces"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"
)

// Method to listen to room messages.
//
// Initially created to manage the room timer, but was  expanded to be used by a few
// other messages.
func (rw *RoomWorker) ListenRoom(ctx context.Context, rdb *redisdb.RedisClient, room *models.Room) {
	pubsub := rdb.Client.Subscribe(ctx, "roompubsub:"+room.ID)
	ch := pubsub.Channel()

	go func() {
		defer pubsub.Close()

		countdown := 30
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// Main loop of the listen, will check the channel and the tiker channel for messages.
		for {
			select {
			case msg := <-ch:
				//log.Println("Received:", msg.Payload)
				// Main switch to route the channel specific messages.
				switch {
				case msg.Payload == "room_end":
					//log.Println("Timer canceled by message room_end")
					rw.HandleEndRoom(room)
					return

				case msg.Payload == "game_start":
					//log.Println("Timer canceled by message game_start")
					return

				case strings.HasPrefix(msg.Payload, "leave_room:"):
					playerID := strings.TrimPrefix(msg.Payload, "leave_room:")
					// First we will handle the player who left.
					playerWhoLeft, _ := room.GetPlayer(playerID)
					playerWhoLeft.SetStatusOnline()
					rdb.UpdatePlayer(playerWhoLeft)
					// now we handle the player that was in the room, he can be online, offline, ready or unready.
					opponentId, _ := room.GetOpponentPlayerID(playerID)
					// We check if its offline.
					opponentPlayer := rdb.GetDisconnectedInQueuePlayerData(opponentId)
					if opponentPlayer == nil {
						// This means the player should be online:
						msg, _ := messages.NewMessage("opponent_left_room", true)
						opponentPlayer, _ = room.GetPlayer(opponentId)
						redisClient.PublishToPlayer(*opponentPlayer, string(msg))
						rw.AddPlayerToQueue(opponentPlayer, true, true)
					} else {
						rw.AddPlayerToQueue(opponentPlayer, true, true)
					}
					err := rdb.RemoveRoom(redisdb.GenerateRoomRedisKeyById(room.ID))
					if err != nil {
						log.Printf("[RoomWorker-%d] - processRoomEnding - Error removing room: %v\n", pid, err)
					}
					return

				case strings.HasPrefix(msg.Payload, "player_ready:"):
					playerID := strings.TrimPrefix(msg.Payload, "player_ready:")
					room.SetPlayerReady(playerID)
					player, _ := room.GetPlayer(playerID)
					rw.HandleReadyRoomNew(player, room)

				case strings.HasPrefix(msg.Payload, "player_unready:"):
					playerID := strings.TrimPrefix(msg.Payload, "player_unready:")
					room.SetPlayerUnReady(playerID)
					player, _ := room.GetPlayer(playerID)
					rw.HandleUnReadyRoomNew(player, room)

				case strings.HasPrefix(msg.Payload, "player_reconnect:"):
					playerID := strings.TrimPrefix(msg.Payload, "player_reconnect:")
					opponent, _ := room.GetOpponentPlayer(playerID)
					player, _ := room.GetOpponentPlayer(opponent.ID)
					outBoundMsg, _ := messages.GeneratePairedMessage(player, opponent, room.ID, room.DeducePlayerColor(playerID), interfaces.CalculateWinAmount(int64(room.BetValue*100), room.OperatorIdentifier.WinFactor), countdown)
					rdb.PublishToPlayerID(playerID, string(outBoundMsg))
				}

			case <-ticker.C:
				countdown--
				if countdown == 27 || countdown == 10 || countdown == 3 {
					timerMsg, _ := messages.NewMessage("room_timer", strconv.Itoa(countdown))
					rdb.PublishToPlayerID(room.Player1.ID, string(timerMsg))
					rdb.PublishToPlayerID(room.Player2.ID, string(timerMsg))
				}
				//log.Println("Countdown:", countdown)
				if countdown <= 0 {
					//log.Println("Room timed out")
					rw.HandleEndRoom(room)
					return
				}

			case <-ctx.Done():
				return
			}
		}

	}()
}
