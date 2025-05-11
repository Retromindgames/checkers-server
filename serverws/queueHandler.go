package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"
)

type QueueHandler struct {
	Player      *models.Player
	RedisClient *redisdb.RedisClient
	Msg         *messages.Message[json.RawMessage]

	// Track changes for cleanup
	initialValidationsFailed bool
	statusUpdated            bool
	addedToQueue             bool
	queueCountIncr           bool
}

func NewQueueHandler(player *models.Player, redis *redisdb.RedisClient, msg *messages.Message[json.RawMessage]) *QueueHandler {
	return &QueueHandler{
		Player:      player,
		RedisClient: redis,
		Msg:         msg,
	}
}

func (qh *QueueHandler) Process() {
	defer qh.cleanup()

	if !qh.validateStatusTransition() {
		qh.initialValidationsFailed = true
		log.Print("QueueHandler process invalid status transition.")
		return
	}

	betValue, err := qh.parseBetValue()
	if err != nil {
		qh.initialValidationsFailed = true
		log.Print("QueueHandler failed to parse bet.")
		return
	}

	found := false
	for _, v := range models.DamasValidBetAmounts {
		if v == betValue {
			found = true
			break
		}
	}
	if !found {
		qh.initialValidationsFailed = true
		log.Print("Bet is not valid for the configured ValidBetAmounts")
		return
	}

	qh.updatePlayerState(betValue)
	qh.addToRedisQueue()
	qh.updateQueueCount()
	qh.sendConfirmation()
}

func (qh *QueueHandler) cleanup() {
	var sendFailedQueueConfirmation = false

	if qh.initialValidationsFailed {
		qh.Player.UpdatePlayerStatus(models.StatusOnline)
		qh.RedisClient.UpdatePlayer(qh.Player)
		sendFailedQueueConfirmation = true
	}

	if qh.statusUpdated {
		qh.Player.UpdatePlayerStatus(models.StatusOnline)
		qh.RedisClient.UpdatePlayer(qh.Player)
		sendFailedQueueConfirmation = true
	}

	if qh.addedToQueue {
		queueName := fmt.Sprintf("queue:%f", qh.Player.SelectedBet)
		qh.RedisClient.Client.LRem(context.Background(), queueName, 1, qh.Player)
		sendFailedQueueConfirmation = true
	}

	if qh.queueCountIncr {
		qh.RedisClient.DecrementQueueCount(qh.Player.SelectedBet)
		sendFailedQueueConfirmation = true
	}

	if sendFailedQueueConfirmation {
		msg, _ := messages.GenerateQueueConfirmationMessage(false)
		qh.Player.WriteChan <- msg
	}
}

func (qh *QueueHandler) validateStatusTransition() bool {
	if qh.Player.UpdatePlayerStatus(models.StatusInQueue) != nil {
		msgBytes, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'queue'")
		qh.Player.WriteChan <- msgBytes
		return false
	}
	qh.statusUpdated = true
	return true
}

func (qh *QueueHandler) parseBetValue() (float64, error) {
	var betValue float64
	err := json.Unmarshal(qh.Msg.Value, &betValue)
	if err != nil {
		log.Printf("Error determining player bet value: %v\n", err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "Error determining player bet value")
		qh.Player.WriteChan <- msgBytes
		return 0, err
	}
	return betValue, nil
}

func (qh *QueueHandler) updatePlayerState(betValue float64) {
	qh.Player.SelectedBet = betValue
	qh.Player.Status = models.StatusInQueue
	qh.RedisClient.UpdatePlayersInQueueSet(qh.Player.ID, models.StatusInQueue)
	qh.RedisClient.UpdatePlayer(qh.Player)
}

func (qh *QueueHandler) addToRedisQueue() error {
	queueName := fmt.Sprintf("queue:%f", qh.Player.SelectedBet)
	err := qh.RedisClient.RPush(queueName, qh.Player)
	if err != nil {
		log.Printf("Error pushing player to Redis queue: %v\n", err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "error adding player to queue")
		qh.Player.WriteChan <- msgBytes
		return err
	}
	qh.addedToQueue = true
	return nil
}

func (qh *QueueHandler) updateQueueCount() {
	exists, err := qh.RedisClient.CheckQueueCountExists(qh.Player.SelectedBet)
	if err == nil {
		if !exists {
			qh.RedisClient.CreateQueueCount(qh.Player.SelectedBet)
		} else {
			qh.RedisClient.IncrementQueueCount(qh.Player.SelectedBet)
		}
		qh.queueCountIncr = true
	}
}

func (qh *QueueHandler) sendConfirmation() {
	m, err := messages.GenerateQueueConfirmationMessage(true)
	if err != nil {
		fmt.Println("Error generating queue confirmation:", err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "error generating confirmation")
		qh.Player.WriteChan <- msgBytes
		return
	}
	qh.Player.WriteChan <- m

	// Success - disable cleanup
	qh.statusUpdated = false
	qh.addedToQueue = false
	qh.queueCountIncr = false
}
