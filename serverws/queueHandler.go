package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"
)

type QueueHandler struct {
	Client      *Client
	RedisClient *redisdb.RedisClient
	Msg         *messages.Message[json.RawMessage]
	GameName    string
	// Track changes for cleanup
	initialValidationsFailed bool
	statusUpdated            bool
	addedToQueue             bool
	queueCountIncr           bool
}

func NewQueueHandler(client *Client, redis *redisdb.RedisClient, msg *messages.Message[json.RawMessage], gn string) *QueueHandler {
	return &QueueHandler{
		Client:      client,
		RedisClient: redis,
		Msg:         msg,
		GameName:    gn,
	}
}

func (qh *QueueHandler) Process() {
	defer qh.cleanup()

	if !qh.validateStatusTransition() {
		qh.initialValidationsFailed = true
		logger.Default.Errorf("Invalid status transition to queue, for session id: %v", qh.Client.player.ID)
		return
	}

	betValue, err := qh.parseBetValue()
	if err != nil {
		qh.initialValidationsFailed = true
		logger.Default.Errorf("QueueHandler failed to parse bet, for session id: %v, with err: %v", qh.Client.player.ID, err)
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
		logger.Default.Errorf("Invalid bet for session id: %v, with bet value: %v", qh.Client.player.ID, betValue)
		return
	}
	logger.Default.Infof("Valid bet for session id: %v", qh.Client.player.ID)
	qh.updatePlayerState(betValue)
	qh.addToRedisQueue()
	qh.updateQueueCount()
	qh.sendConfirmation()
	session, _ := qh.RedisClient.GetSessionByID(qh.Client.player.ID)
	ttl := 6 * time.Hour
	qh.RedisClient.RefreshSessionTTL(session, ttl)
}

func (qh *QueueHandler) cleanup() {
	var sendFailedQueueConfirmation = false

	if qh.initialValidationsFailed {
		qh.Client.player.UpdatePlayerStatus(models.StatusOnline)
		qh.RedisClient.UpdatePlayer(qh.Client.player)
		sendFailedQueueConfirmation = true
	}

	if qh.statusUpdated {
		qh.Client.player.UpdatePlayerStatus(models.StatusOnline)
		qh.RedisClient.UpdatePlayer(qh.Client.player)
		sendFailedQueueConfirmation = true
	}

	if qh.addedToQueue {
		queueName := fmt.Sprintf("queue:%f", qh.Client.player.SelectedBet)
		qh.RedisClient.Client.LRem(context.Background(), queueName, 1, qh.Client.player)
		sendFailedQueueConfirmation = true
	}

	if qh.queueCountIncr {
		qh.RedisClient.DecrementQueueCount(qh.GameName, qh.Client.player.SelectedBet)
		sendFailedQueueConfirmation = true
	}

	if sendFailedQueueConfirmation {
		msg, _ := messages.GenerateQueueConfirmationMessage(false)
		qh.Client.send <- msg
	}
}

func (qh *QueueHandler) validateStatusTransition() bool {
	if qh.Client.player.UpdatePlayerStatus(models.StatusInQueue) != nil {
		msgBytes, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'queue'")
		qh.Client.send <- msgBytes
		return false
	}
	qh.statusUpdated = true
	return true
}

func (qh *QueueHandler) parseBetValue() (float64, error) {
	var betValue float64
	err := json.Unmarshal(qh.Msg.Value, &betValue)
	if err != nil {
		msgBytes, _ := messages.GenerateGenericMessage("error", "Error determining player bet value")
		qh.Client.send <- msgBytes
		return 0, err
	}
	return betValue, nil
}

func (qh *QueueHandler) updatePlayerState(betValue float64) {
	qh.Client.player.SelectedBet = betValue
	qh.Client.player.Status = models.StatusInQueue
	qh.RedisClient.UpdatePlayer(qh.Client.player)
}

func (qh *QueueHandler) addToRedisQueue() error {
	queueName := fmt.Sprintf("queue:{%v}:%f", qh.GameName, qh.Client.player.SelectedBet)
	err := qh.RedisClient.RPush(queueName, qh.Client.player)
	if err != nil {
		msgBytes, _ := messages.GenerateGenericMessage("error", "error adding player to queue")
		qh.Client.send <- msgBytes
		return err
	}
	qh.addedToQueue = true
	return nil
}

func (qh *QueueHandler) updateQueueCount() {
	exists, err := qh.RedisClient.CheckQueueCountExists(qh.GameName, qh.Client.player.SelectedBet)
	if err == nil {
		if !exists {
			qh.RedisClient.CreateQueueCount(qh.GameName, qh.Client.player.SelectedBet)
		} else {
			qh.RedisClient.IncrementQueueCount(qh.GameName, qh.Client.player.SelectedBet)
		}
		qh.queueCountIncr = true
	}
}

func (qh *QueueHandler) sendConfirmation() {
	m, err := messages.GenerateQueueConfirmationMessage(true)
	if err != nil {
		msgBytes, _ := messages.GenerateGenericMessage("error", "error generating confirmation")
		qh.Client.send <- msgBytes
		return
	}
	qh.Client.send <- m

	// Success - disable cleanup
	qh.statusUpdated = false
	qh.addedToQueue = false
	qh.queueCountIncr = false
}
