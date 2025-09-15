package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Lavizord/checkers-server/interfaces"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"
)

func (rw *RoomWorker) HandleQueuePaired(player1, player2 *models.Player, p1disc, p2disc bool) {
	room := &models.Room{
		ID:                 models.GenerateUUID(),
		Player1:            player1,
		Player2:            player2,
		StartDate:          time.Now(),
		Currency:           player1.Currency,
		BetValue:           player1.SelectedBet,
		OperatorIdentifier: player1.OperatorIdentifier,
	}

	player1.RoomID = room.ID
	player2.RoomID = room.ID
	player1.Status = models.StatusInRoom
	player2.Status = models.StatusInRoom

	if p1disc == true {
		rw.RedisClient.SaveDisconnectInQueuePlayerData(player1)
	} else {
		rw.RedisClient.UpdatePlayer(player1)
	}
	if p2disc == true {
		rw.RedisClient.SaveDisconnectInQueuePlayerData(player2)
	} else {
		rw.RedisClient.UpdatePlayer(player2)
	}

	cleanup := true
	defer func() {
		if cleanup {
			player1.RoomID = ""
			player2.RoomID = ""
			player1.Status = models.StatusOnline
			player2.Status = models.StatusOnline
			if p1disc == true {
				rw.RedisClient.SaveDisconnectInQueuePlayerData(player1)
			} else {
				rw.RedisClient.UpdatePlayer(player1)
			}
			if p2disc == true {
				rw.RedisClient.SaveDisconnectInQueuePlayerData(player2)
			} else {
				rw.RedisClient.UpdatePlayer(player2)
			}
			rw.RedisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(room.ID))
			rw.RedisClient.DecrementQueueCount(rw.GameName, player1.SelectedBet)
			rw.RedisClient.DecrementQueueCount(rw.GameName, player2.SelectedBet)
			msg, _ := messages.GenerateGenericMessage("error", "failed to handle queue paired.")
			rw.RedisClient.PublishToPlayer(*player1, string(msg))
			rw.RedisClient.PublishToPlayer(*player2, string(msg))
		}
	}()

	colorp1 := rand.Intn(2)
	colorp2 := 1
	if colorp1 == 1 {
		room.CurrentPlayerID = player1.ID
		colorp2 = 0
	} else {
		room.CurrentPlayerID = player2.ID
	}

	err := rw.RedisClient.AddRoom(rw.GameName, room)
	if err != nil {
		log.Printf("Failed to add room to Redis: %v\n", err)
		return
	}

	message1, err := messages.GeneratePairedMessage(player1, player2, room.ID, colorp1, interfaces.CalculateWinAmount(int64(room.BetValue*100), room.OperatorIdentifier.WinFactor), 30)
	if err != nil {
		log.Printf("Error generating message for player1: %v\n", err)
		return
	}

	message2, err := messages.GeneratePairedMessage(player2, player1, room.ID, colorp2, interfaces.CalculateWinAmount(int64(room.BetValue*100), room.OperatorIdentifier.WinFactor), 30)
	if err != nil {
		log.Printf("Error generating message for player2: %v\n", err)
		return
	}

	err = rw.RedisClient.PublishToPlayer(*player1, string(message1))
	if err != nil {
		log.Printf("Failed to publish message to player1: %v\n", err)
		return
	}
	err = rw.RedisClient.PublishToPlayer(*player2, string(message2))
	if err != nil {
		log.Printf("Failed to publish message to player2: %v\n", err)
		return
	}

	// This will start a pubsub tied to a timer.
	rw.ListenRoom(context.Background(), redisClient, room)

	cleanup = false
	rw.RedisClient.DecrementQueueCount(rw.GameName, player1.SelectedBet)
	rw.RedisClient.DecrementQueueCount(rw.GameName, player2.SelectedBet)
}

func (rw *RoomWorker) HandleReadyRoomNew(player *models.Player, proom *models.Room) {
	//log.Printf("[RoomWorker-%d] - Handling player (READY QUEUE): %s (Session: %s, Currency: %s)\n",
	//	pid, player.ID, player.SessionID, player.Currency)

	var errorWithOpponent bool
	errorWithOpponent = false
	defer func() {
		if errorWithOpponent {
			rw.AddPlayerToQueue(player, true, true)
		}
	}()
	player2, err := proom.GetOpponentPlayer(player.ID)
	// We will always notify the opponent the we are ready.
	msg, err := messages.GenerateOpponentReadyMessage(true)
	if err != nil {
		log.Printf("[RoomWorker-%d] - Error handleReadyRoom getting GenerateOpponentReadyMessage(true) for opponent:%s\n", pid, err)
		return
	}
	rw.RedisClient.PublishPlayerEvent(player2, string(msg))
	// now we tell our player that is ready if the opponent is ready or not.
	if player2.Status != models.StatusInRoomReady {
		//log.Printf("[RoomWorker-%d] - handleReadyRoom Opponent aint ready yet!:%s\n", pid, err)
		msg, err := messages.GenerateOpponentReadyMessage(false)
		if err != nil {
			log.Printf("[RoomWorker-%d] - Error handleReadyRoom getting GenerateOpponentReadyMessage(false) for player:%s\n", pid, err)
		}
		rw.RedisClient.PublishPlayerEvent(player, string(msg))
		return
	}
	// Now! If both players are ready...!!
	// Before we start the game, we will need to post to the wallet api of the bet, we will use our api interface for that.
	module, exists := interfaces.OperatorModules[proom.OperatorIdentifier.OperatorName]
	if !exists {
		log.Printf("[RoomWorker-%d] - Error handleReadyRoom getting getting interfaces.OperatorModules[%v]:%s\n", pid, proom.OperatorIdentifier.OperatorName, err)
		return
	}

	session1, err := rw.RedisClient.GetSessionByID(player.SessionID)
	if err != nil {
		log.Printf("[RoomWorker-%d] - Error handleReadyRoom fetching player1 sessionID:%s\n", pid, err)
		return
	}
	session2, err := rw.RedisClient.GetSessionByID(player2.SessionID)
	if err != nil {
		log.Printf("[RoomWorker-%d] - Error handleReadyRoom fetching player2 sessionID:%s\n", pid, err)
		return
	}

	newBalance1, err := module.HandlePostBet(postgresClient, redisClient, *session1, int64(proom.BetValue*100), proom.ID)
	if err != nil {
		log.Printf("[RoomWorker-%d] - Error HandlePostBet failed to bet:%s for sessionid:[%s]\n", pid, err, session1.ID)
		player.SetStatusOnline()
		rw.RedisClient.UpdatePlayer(player)
		msg, _ := messages.GenerateGenericMessage("error", err.Error())
		rw.RedisClient.PublishPlayerEvent(player, string(msg))
		rw.RedisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(proom.ID))

		msg, _ = messages.NewMessage("opponent_left_room", true)
		rw.RedisClient.PublishPlayerEvent(player2, string(msg))

		// since the first player failed the api check, we will queue up the second plyer.
		rw.AddPlayerToQueue(player2, true, true)
		// TODO: CREDITAR VALOR A JOGADOR.
		return
	}
	newBalance2, err := module.HandlePostBet(postgresClient, redisClient, *session2, int64(proom.BetValue*100), proom.ID)
	if err != nil {
		log.Printf("[RoomWorker-%d] - Error HandlePostBet failed to bet:%s for sessionid:[%s]\n", pid, err, session1.ID)
		player2.SetStatusOnline()
		rw.RedisClient.UpdatePlayer(player2)
		msg, _ := messages.GenerateGenericMessage("error", err.Error())
		rw.RedisClient.PublishPlayerEvent(player2, string(msg))
		rw.RedisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(proom.ID))

		// since the second player failed the api check, we will queue up the first player.
		rw.AddPlayerToQueue(player2, true, true)
		// TODO: CREDITAR VALOR A JOGADOR.
		return
	}
	// Now that everything is OK, we will start up the game
	msgP1, _ := messages.NewMessage("balance_update", float64(newBalance1)/100)
	msgP2, _ := messages.NewMessage("balance_update", float64(newBalance2)/100)

	// then notify player and store it in redis.
	rw.RedisClient.UpdatePlayer(player)
	rw.RedisClient.UpdatePlayer(player2)

	rw.RedisClient.PublishPlayerEvent(player, string(msgP1))
	rw.RedisClient.PublishPlayerEvent(player2, string(msgP2))

	rw.RedisClient.PublishToRoomPubSub(proom.ID, "game_start")

	// Then we start a match
	roomdata, _ := json.Marshal(proom)
	key := fmt.Sprintf("create_game:{%v}", rw.GameName)
	err = rw.RedisClient.RPushGeneric(key, roomdata)
	if err != nil {
		log.Printf("[RoomWorker-%d] - Error handleReadyRoom Creating Game RPushGeneric:%s\n", pid, err)
	}

}

func (rw *RoomWorker) HandleUnReadyRoomNew(player *models.Player, proom *models.Room) {
	//log.Printf("[RoomWorker-%d] - Handling player (UN-READY QUEUE): %s (Session: %s, Currency: %s)\n",
	//	pid, player.ID, player.SessionID, player.Currency)

	var errorWithOpponent bool
	errorWithOpponent = false
	defer func() {
		if errorWithOpponent {
			rw.AddPlayerToQueue(player, true, true)
		}
	}()
	player2, err := proom.GetOpponentPlayer(player.ID)
	if err != nil {
		errorWithOpponent = true
		log.Printf("[RoomWorker-%d] - Error handleUnReadyRoom getting opponent player:%s\n", pid, err)
		return
	}
	// We will always notify the opponent the we are no longer ready.
	msg, err := messages.GenerateOpponentReadyMessage(false)
	if err != nil {
		log.Printf("[RoomWorker-%d] - Error handleUnReadyRoom getting GenerateOpponentReadyMessage(false) for opponent:%s\n", pid, err)
		return
	}
	rw.RedisClient.PublishPlayerEvent(player2, string(msg))

	// now we tell our player that is ready if the opponent is ready or not.
	if player2.Status != models.StatusInRoomReady {
		//log.Printf("[RoomWorker-%d] - handleUnReadyRoom Opponent aint ready yet!:%s\n", pid, err)
		msg, err := messages.GenerateOpponentReadyMessage(false)
		if err != nil {
			log.Printf("[RoomWorker-%d] - Error handleUnReadyRoom getting GenerateOpponentReadyMessage(false) for player:%s\n", pid, err)
		}
		rw.RedisClient.PublishPlayerEvent(player, string(msg))
		return
	}
}

func (rw *RoomWorker) HandleEndRoom(room *models.Room) {
	p1, _ := rw.RedisClient.GetPlayer(room.Player1.ID)
	p2, _ := rw.RedisClient.GetPlayer(room.Player2.ID)

	key := fmt.Sprintf("%t_%t", p1 != nil, p2 != nil)
	switch key {
	case "false_false":
		// both nil, no players, we will just remove the room, and remove both players from possible offline lists.
		//log.Printf("[RoomWorker-%d] - handleEndRoom - false_false: %v\n", pid)
		p1 = rw.RedisClient.GetDisconnectedInQueuePlayerData(room.Player1.ID)
		if p1 != nil {
			if p1.Status == models.StatusInRoomReady {
				rw.AddPlayerToQueue(p1, true, true)
			} else {
				rw.RedisClient.DeleteDisconnectedInQueuePlayerData(p1.ID)
			}
		}
		p2 = rw.RedisClient.GetDisconnectedInQueuePlayerData(room.Player2.ID)
		if p2 != nil {
			if p2.Status == models.StatusInRoomReady {
				rw.AddPlayerToQueue(p2, true, true)
			} else {
				rw.RedisClient.DeleteDisconnectedInQueuePlayerData(p2.ID)
			}
		}
		err := rw.RedisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(room.ID))
		if err != nil {
			log.Printf("[RoomWorker-%d] - handleEndRoom - Error removing room: %v\n", pid, err)
			return
		}

	case "true_false":
		// only p1, we will handle the removal of the P2, and requeue the p1.
		// log.Printf("[RoomWorker-%d] - handleEndRoom - false_true: %v\n", pid)
		// Since this guy is offline, we will check if its state was in room ready.
		msg, _ := messages.NewMessage("room_failed_ready_check", true)
		p2 = rw.RedisClient.GetDisconnectedInQueuePlayerData(room.Player2.ID)
		if p2 != nil {
			if p2.Status == models.StatusInRoomReady {
				rw.AddPlayerToQueue(p2, true, true)
			} else {
				rw.RedisClient.DeleteDisconnectedInQueuePlayerData(p2.ID)
			}
		}
		if p1.Status == models.StatusInRoomReady {
			rw.AddPlayerToQueue(p1, true, true)
		} else {
			p1.SetStatusOnline()
			rw.RedisClient.PublishToPlayerID(p1.ID, string(msg))
			rw.RedisClient.UpdatePlayer(p1)
		}
		err := rw.RedisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(room.ID))
		if err != nil {
			log.Printf("[RoomWorker-%d] - handleEndRoom - Error removing room: %v\n", pid, err)
			return
		}
		return

	case "false_true":
		// only p1, we will handle the removal of the P2, and requeue the p1.
		//log.Printf("[RoomWorker-%d] - handleEndRoom - false_true:\n", pid)
		msg, _ := messages.NewMessage("room_failed_ready_check", true)
		// Since this guy is offline, we will check if its state was in room ready.
		p1 = rw.RedisClient.GetDisconnectedInQueuePlayerData(room.Player1.ID)
		if p1 != nil {
			if p1.Status == models.StatusInRoomReady {
				rw.AddPlayerToQueue(p1, true, true)
			} else {
				rw.RedisClient.DeleteDisconnectedInQueuePlayerData(p1.ID)
			}
		}
		if p2.Status == models.StatusInRoomReady {
			rw.AddPlayerToQueue(p2, true, true)
		} else {
			p2.SetStatusOnline()
			rw.RedisClient.PublishToPlayerID(p2.ID, string(msg))
			rw.RedisClient.UpdatePlayer(p2)
		}
		err := rw.RedisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(room.ID))
		if err != nil {
			log.Printf("[RoomWorker-%d] - handleEndRoom - Error removing room:", pid)
			return
		}
		return

	case "true_true":
		// both, this means that the timer ran out...
		//log.Printf("[RoomWorker-%d] - handleEndRoom - true_true:\n", pid)
		msg, _ := messages.NewMessage("room_failed_ready_check", true)
		if p1.Status == models.StatusInRoomReady {
			rw.AddPlayerToQueue(p1, true, true)
		} else {
			p1.SetStatusOnline()
			rw.RedisClient.PublishToPlayerID(p1.ID, string(msg))
			rw.RedisClient.UpdatePlayer(p1)
		}
		if p2.Status == models.StatusInRoomReady {
			rw.AddPlayerToQueue(p2, true, true)
		} else {
			p2.SetStatusOnline()
			rw.RedisClient.PublishToPlayerID(p2.ID, string(msg))
			rw.RedisClient.UpdatePlayer(p2)
		}
		err := rw.RedisClient.RemoveRoom(redisdb.GenerateRoomRedisKeyById(room.ID))
		if err != nil {
			log.Printf("[RoomWorker-%d] - handleEndRoom - Error removing room: %v\n", pid, err)
			return
		}
		return
	}
}
