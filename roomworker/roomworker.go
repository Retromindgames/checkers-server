package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"
)

type RoomWorker struct {
	RedisClient      *redisdb.RedisClient
	GameName         string
	QueueBetAmmounts []float64
}

func NewRoomWorker(redis *redisdb.RedisClient, gn string, betAmounts []float64) *RoomWorker {
	return &RoomWorker{
		RedisClient:      redis,
		GameName:         gn,
		QueueBetAmmounts: betAmounts,
	}
}

func (rw *RoomWorker) Run() {
	go rw.ProcessQueue()
	go rw.ProcessReadyQueue()
}

func (rw *RoomWorker) ProcessQueue() {
	// Launch a goroutine for each bet queue
	for _, bet := range rw.QueueBetAmmounts {
		go rw.ProcessQueueForBet(bet)
	}
	// Block forever or wait on a channel (to prevent the main goroutine from exiting)
	select {}
}

func (rw *RoomWorker) ProcessQueueForBet(bet float64) {
	queueName := fmt.Sprintf("queue:{%v}:%f", rw.GameName, bet)
	for {
		// Block indefinitely for player1 (this goroutine is dedicated to this queue)
		player1, err := rw.RedisClient.BLPop(queueName, 0)
		if err != nil {
			logger.Default.Errorf("error retrieving player 1 from %s: %v", queueName, err)
			continue
		}
		var p1disc, p2disc bool

		// log.Printf("[RoomWorker-%d] - Retrieved player 1 from %s: %v\n", pid, queueName, player1)
		player1Details, err := rw.RedisClient.GetPlayer(player1.ID)
		if err != nil {
			// We will make a check if the player is one of the disconnected players.
			player1Details = rw.RedisClient.GetDisconnectedInQueuePlayerData(player1.ID)
			if player1Details == nil {
				logger.Default.Warnf("error retrieving player 1 details from online and offline list, with id: %v, player removed from queue, with error: %v", player1.ID, err)
				rw.RedisClient.DecrementQueueCount(rw.GameName, bet)
				continue
			}
			if time.Now().Unix()-player1Details.DisconnectedAt > 3600 { // Note: the TTL of the playerDisconneceted is 2h, this is 1h.
				logger.Default.Infof("player was disconnected for longer than 3600 minutes, discarding player from queue, with id: %v", player1Details.ID)
				rw.RedisClient.DecrementQueueCount(rw.GameName, bet)
				rw.RedisClient.DeleteDisconnectedInQueuePlayerData(player1.ID)
				continue
			}
			p1disc = true
			logger.Default.Infof("player 1 details with id: %v, retrieved from offline queue list.", player1.ID)
		}

		// We check to see if the player is eligible to be processed.
		if !player1Details.IsEligibleForQueue(bet) {
			logger.Default.Warnf("player1 with id: %v, and with status %v not eligible to be processed by the queue, player removed from queue: %v", player1Details.ID, player1Details.Status, queueName)
			rw.RedisClient.DecrementQueueCount(rw.GameName, bet)
			continue
		}

		// Try fetching the second player with a timeout
		player2, err := rw.RedisClient.BLPop(queueName, config.Cfg.Services["roomworker"].Timer)
		if err != nil {
			logger.Default.Infof("No second player found in queue: %s, re-queueing player 1 with id: %v", player1.ID, queueName)
			// Since we failed to get the player2, we will requeue the player1.
			time.Sleep(time.Second * 1)
			rw.RedisClient.RPush(queueName, player1)
			continue
		}

		// log.Printf("[RoomWorker-%d] - Retrieved player 2 from %s: %v\n", pid, queueName, player2)
		player2Details, err := rw.RedisClient.GetPlayer(player2.ID)
		if err != nil {
			// We will make a check if the player is one of the disconnected players.
			player2Details = rw.RedisClient.GetDisconnectedInQueuePlayerData(player2.ID)
			if player2Details == nil {
				logger.Default.Warnf("error retrieving player 2 details from online and offline list, with id: %v, player 2 removed from queue, with error: %v", player2.ID, err)
				rw.RedisClient.DecrementQueueCount(rw.GameName, bet)
				continue
			}
			if time.Now().Unix()-player2Details.DisconnectedAt > 3600 { // Note: the TTL of the playerDisconneceted is 2h, this is 1h.
				logger.Default.Infof("player 2 was disconnected for longer than 3600 minutes, discarding player from queue, with id: %v", player2Details.ID)
				rw.RedisClient.DecrementQueueCount(rw.GameName, bet)
				rw.RedisClient.DeleteDisconnectedInQueuePlayerData(player2.ID)
				continue
			}
			p2disc = true
			logger.Default.Infof("player 2 details with id: %v, retrieved from offline queue list.", player2.ID)

		}

		if player1Details.ID == player2Details.ID ||
			(player1Details.Name == player2Details.Name &&
				player1Details.OperatorIdentifier.OperatorName == player2Details.OperatorIdentifier.OperatorName &&
				player1Details.Currency == player2Details.Currency) {
			logger.Default.Infof("same player detected with id: %v, player2 removed from queue: %v, retrieved from offline queue list.", player2.ID, queueName)
			rw.RedisClient.RPush(queueName, player1)
			rw.RedisClient.DecrementQueueCount(rw.GameName, bet)
			continue
		}

		// Before we handle the paired, we will do a final check to make sure the players2 is still online / valid.
		if !player2Details.IsEligibleForQueue(bet) {
			// If it is not valid, we will add player 1 back to the queue.
			logger.Default.Warnf("player2 with id: %v, and with status %v not eligible to be processed by the queue, player removed from queue: %v", player2Details.ID, player2Details.Status, queueName)
			rw.RedisClient.RPush(queueName, player1)
			rw.RedisClient.DecrementQueueCount(rw.GameName, bet)
			continue
		}

		// Process both players
		// log.Printf("[RoomWorker-%d] - Pairing players: %s and %s from %s\n", pid, player1, player2, queueName)
		rw.HandleQueuePaired(player1, player2, p1disc, p2disc)
	}
}

func (rw *RoomWorker) ProcessReadyQueue() {
	key := fmt.Sprintf("ready_queue:{%v}", rw.GameName)
	for {
		playerData, err := rw.RedisClient.BLPop(key, 0) // Block
		if err != nil {
			log.Printf("[RoomWorker-%d] - Error retrieving player:%v\n", pid, err)
			continue
		}
		log.Printf("[RoomWorker-%d] - processing ready room!: %+v\n", pid, playerData)
		// Aqui ou damos handle do ready queue ou handle do unreadyqueue
		if playerData.Status == models.StatusInRoomReady {
			rw.RedisClient.PublishToRoomPubSub(playerData.RoomID, "player_ready:"+playerData.ID)
			//handleReadyRoom(playerData)
			continue
		}
		if playerData.Status == models.StatusInRoom {
			rw.RedisClient.PublishToRoomPubSub(playerData.RoomID, "player_unready:"+playerData.ID)
			//handleUnReadyRoom(playerData)
			continue
		}
		log.Printf("Player is neither InRoomReady neither InRoom?!")
	}
}

func (rw *RoomWorker) AddPlayerToQueue(player *models.Player, incrementQueueCount, notify bool) {
	// Reset both player data.
	player.RoomID = ""
	player.GameID = ""
	player.Status = models.StatusInQueue
	err := rw.RedisClient.UpdatePlayer(player)
	if err != nil {
		rw.RedisClient.SaveDisconnectInQueuePlayerData(player)
	}

	// Pushing the player to the "queue" Redis list
	queueName := fmt.Sprintf("queue:{%v}:%f", rw.GameName, player.SelectedBet)
	err2 := rw.RedisClient.RPush(queueName, player)
	if err2 != nil {
		log.Printf("[RoomWorker-%d] - Error adding player to queue:%v\n", pid, err2)
		return
	}
	if incrementQueueCount {
		rw.RedisClient.IncrementQueueCount(rw.GameName, player.SelectedBet)
	}
	if notify {
		queueMsg, _ := messages.GenerateQueueConfirmationMessage(true)
		rw.RedisClient.PublishPlayerEvent(player, string(queueMsg))
	}
}
