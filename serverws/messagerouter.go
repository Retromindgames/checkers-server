package main

import (
	"encoding/json"
	"fmt"

	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"
)

// Directly route to the right handler based on the command.
//
// The valid command sent will be processsed and router to the right handler. Some of those handlers
// might send the message to redis.
func RouteMessages(message *messages.Message[json.RawMessage], client *Client, redis *redisdb.RedisClient) {
	switch message.Command {
	case "queue":
		logger.Default.Infof("[wsapi] - RouteMessages - received queue message, sending it to handleQueue for session id: %v", client.player.ID)
		handleQueue(message, client, redis)
		return

	case "create_room":
		logger.Default.Infof("[wsapi] - RouteMessages - received create_room message, sending it to handleQueue for session id: %v", client.player.ID)
		if client.player.Status != models.StatusOnline || client.player.RoomID != "" || client.player.GameID != "" {
			logger.Default.Warnf("[wsapi] - RouteMessages -session with id: %v, cant issue a create_room on status : %v", client.player.ID, client.player.Status)
			msg, _ := messages.GenerateGenericMessage("invalid", fmt.Sprintf("Can't issue a create_room when in status: %v.", client.player.Status))
			client.send <- msg
			return
		}
		handleCreateRoom(message, client, redis)
		return

	case "leave_queue":
		logger.Default.Infof("[wsapi] - RouteMessages - received leave_queue message, sending it to handleLeaveQueue for session id: %v", client.player.ID)
		handleLeaveQueue(client, redis)
		return

	case "ready_queue":
		logger.Default.Infof("[wsapi] - RouteMessages - received ready_queue message, sending it to handleReadyQueue for session id: %v", client.player.ID)
		handleReadyQueue(message, client, redis)
		return

	case "leave_room":
		logger.Default.Infof("[wsapi] - RouteMessages - received leave_room message, sending it to handleLeaveRoom for session id: %v", client.player.ID)
		if client.player.Status != models.StatusInRoom && client.player.Status != models.StatusInRoomReady {
			logger.Default.Warnf("[wsapi] - RouteMessages - cant issue a leave room when not in a room for session id: %v", client.player.ID)
			msg, _ := messages.GenerateGenericMessage("invalid", "Can't issue a leave room when not in a Room.")
			client.send <- msg
			return
		}
		handleLeaveRoom(client, redis)
		return

	case "leave_game":
		logger.Default.Infof("[wsapi] - RouteMessages - received leave_game message, sending it to handleLeaveGame for session id: %v", client.player.ID)
		if client.player.Status != models.StatusInGame {
			logger.Default.Warnf("[wsapi] - RouteMessages - cant issue a leave game when not in a game for session id: %v", client.player.ID)
			msg, _ := messages.GenerateGenericMessage("invalid", "Can't issue a leave game when not in a game.")
			client.send <- msg
			return
		}
		handleLeaveGame(client, redis)
		return

	case "move_piece":
		logger.Default.Infof("[wsapi] - RouteMessages - received move_piece message, sending it to handleMovePiece for session id: %v", client.player.ID)
		if client.player.Status != models.StatusInGame {
			logger.Default.Warnf("[wsapi] - RouteMessages - cant issue a move_piece when not in a Game for session id: %v", client.player.ID)
			msg, _ := messages.GenerateGenericMessage("invalid_state_goto_menu", "Can't issue a move when not in a Game.") // This tells FE to go back to main menu.
			client.send <- msg
			return
		}
		handleMovePiece(message, client, redis)
		return
	}
}

func handleQueue(msg *messages.Message[json.RawMessage], client *Client, redis *redisdb.RedisClient) {
	qh := NewQueueHandler(client, redis, msg)
	qh.Process()
}

func handleLeaveQueue(client *Client, redis *redisdb.RedisClient) {
	if client.player.UpdatePlayerStatus(models.StatusOnline) != nil {
		logger.Default.Warnf("[wsapi] - handleLeaveQueue - invalid status transition to 'Online' for session id: %v", client.player.ID)
		msg, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'Online'")
		client.send <- msg
		return
	}
	client.player.SetStatusOnline()
	redis.UpdatePlayer(client.player) // This is important, we will only re-add players to a queue that are in queue.
	//redis.UpdatePlayersInQueueSet(client.player.ID, models.StatusOnline)
	queueName := fmt.Sprintf("queue:%f", client.player.SelectedBet)
	err := redis.RemovePlayerFromQueue(queueName, client.player)
	if err != nil {
		//log.Printf("Error removing player from Redis queue: %v\n", err)	//! This was commented, it fails when there is only 1 player in queue.
		//client.send <- []byte("Error removing player to queue")
		return
	}
	// we update out player status, send a confirmation message back to the player
	m, err := messages.GenerateQueueConfirmationMessage(false)
	if err != nil {
		msg, _ := messages.GenerateGenericMessage("error", "Error generating queue confirmation false:"+err.Error())
		logger.Default.Errorf("[wsapi] - handleLeaveQueue - Error generating queue confirmation false for session id: %v", client.player.ID)
		client.send <- msg
		return
	}
	logger.Default.Infof("[wsapi] - handleLeaveQueue - player status set to removed from queue, sending confirmation message for session id: %v", client.player.ID)
	client.send <- m
}

func handleReadyQueue(msg *messages.Message[json.RawMessage], client *Client, redis *redisdb.RedisClient) {
	// We have to check if the message for readyqueue true or false.
	var value bool
	json.Unmarshal(msg.Value, &value)
	if value {
		// update the player status to ready / awaiting opponent.
		if client.player.UpdatePlayerStatus(models.StatusInRoomReady) != nil {
			logger.Default.Errorf("[wsapi] - handleReadyQueue - Invalid status transition to 'ready_queue true for session id: %v", client.player.ID)
			msg, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'ready_queue true'")
			client.send <- msg
			return
		}
		logger.Default.Infof("[wsapi] - handleReadyQueue - status transition to 'ready_queue true for session id: %v", client.player.ID)
	} else {
		// update the player status, to unready / waiting ready.
		if client.player.UpdatePlayerStatus(models.StatusInRoom) != nil {
			logger.Default.Errorf("[wsapi] - handleReadyQueue - Invalid status transition to 'ready_queue' false for session id: %v", client.player.ID)
			msg, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'ready_queue' false'")
			client.send <- msg
			return
		}
		logger.Default.Infof("[wsapi] - handleReadyQueue - status transition to 'ready_queue' false for session id: %v", client.player.ID)
	}
	msgBytes, _ := messages.GenerateGenericMessage("info", "Processing 'ready_queue'")
	client.send <- msgBytes
	err := redis.UpdatePlayer(client.player)
	if err != nil {
		logger.Default.Error("[wsapi] - handleReadyQueue - error updating player to Redis for session id: %v, error: %v", client.player.ID, err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "Error Updating player: "+err.Error())
		client.send <- msgBytes
		return
	}
	err = redis.RPush("ready_queue", client.player) // now we tell roomworker to process this player ready.
	if err != nil {
		logger.Default.Error("[wsapi] - handleReadyQueue - error pushing player to roomworker Redis ready queue for session id: %v, error: %v", client.player.ID, err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "Pushing player to queue: "+err.Error())
		client.send <- msgBytes
		return
	}
	logger.Default.Infof("[wsapi] - handleReadyQueue - pushed ready queue for roomworker for session id: %v", client.player.ID)
}

func handleLeaveRoom(client *Client, redis *redisdb.RedisClient) {
	// update the player status
	if client.player.UpdatePlayerStatus(models.StatusOnline) != nil {
		logger.Default.Warnf("[wsapi] - handleLeaveRoom - status transition to 'online' for session id: %v", client.player.ID)
		msgBytes, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'leave_room'")
		client.send <- msgBytes
		return
	}
	redis.PublishToRoomPubSub(client.player.RoomID, "leave_room:"+client.player.ID)
	logger.Default.Infof("[wsapi] - handleLeaveRoom - published to room pubsub for session id: %v", client.player.ID)
}

func handleLeaveGame(client *Client, redis *redisdb.RedisClient) {
	msgBytes, _ := messages.GenerateGenericMessage("info", "Processing 'leave_game'")
	client.send <- msgBytes
	err := redis.RPush("leave_game", client.player)
	if err != nil {
		logger.Default.Error("[wsapi] - handleReadyQueue - error pushing player to Redis ready queue for session id: %v, error: %v", client.player.ID, err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "Error adding player to leave_game")
		client.send <- msgBytes
		return
	}
	logger.Default.Infof("[wsapi] - handleLeaveGame - leave_game message sent to gameworker for session id: %v", client.player.ID)
}

func handleMovePiece(message *messages.Message[json.RawMessage], client *Client, redis *redisdb.RedisClient) {
	var move models.Move
	err := json.Unmarshal([]byte(message.Value), &move)
	if err != nil {
		logger.Default.Errorf("[wsapi] - handleMovePiece - JSON Unmarshal Error for session id: %v", client.player.ID)
		msg, _ := messages.GenerateGenericMessage("invalid", "Handle Move Piece - JSON Unmarshal Error.")
		client.send <- msg
		return
	}
	if move.PlayerID != client.player.ID {
		logger.Default.Errorf("[wsapi] - handleMovePiece - move.PlayerID != player.ID for session id: %v", client.player.ID)
		msg, _ := messages.GenerateGenericMessage("invalid", "Handle Move Piece - move.PlayerID != player.ID.")
		client.send <- msg
		return
	}
	// movement message is sent to the game worker
	err = redis.RPushGeneric("move_piece", message.Value)
	if err != nil {
		logger.Default.Errorf("[wsapi] - handleMovePiece - error pushing move to redis for session id: %v, err: %v", client.player.ID, err)
		msg, _ := messages.GenerateGenericMessage("error", "error pushing move to gameworker.")
		client.send <- msg
		return
	}
	logger.Default.Infof("[wsapi] - handleMovePiece - sent move_piece to gameworker for session id: %v", client.player.ID)
}

func handleCreateRoom(message *messages.Message[json.RawMessage], client *Client, redis *redisdb.RedisClient) {
	// We will update the player status and send the notification to the roomworker to create the room.

	var bet float64
	err := json.Unmarshal([]byte(message.Value), &bet)
	if err != nil {
		logger.Default.Errorf("[wsapi] - handleCreateRoom - JSON Unmarshal Error for session id: %v, received invalid bet format.", client.player.ID)
		msg, _ := messages.GenerateGenericMessage("error", "Failed to parse bet value data from command.")
		client.send <- msg
		return
	}
	if !IsValidBet(bet) {
		logger.Default.Errorf("Invalid bet for session id: %v, with bet value: %v", client.player.ID, bet)
		msg, _ := messages.GenerateGenericMessage("error", "invalid bet value.")
		client.send <- msg
	}

	client.player.SelectedBet = bet
	err = client.player.UpdatePlayerStatus(models.StatusInRoom)
	if err != nil {
		logger.Default.Errorf("Falied to update player status for session id: %v, with bet value: %v, and error: %v", client.player.ID, bet, err.Error())
		msg, _ := messages.GenerateGenericMessage("error", "Handle Create room - Failed to update player status.")
		client.send <- msg
	}
	redis.UpdatePlayer(client.player)

	// create room message is sent to the room worker
	err = redis.RPush("create_room", client.player)
	if err != nil {
		logger.Default.Errorf("Handle Create room - error pushing create_room to redis for session id: %v, err: %v", client.player.ID, err)
		msg, _ := messages.GenerateGenericMessage("error", "error pushing create_room to roomworker.")
		client.send <- msg
		return
	}
	logger.Default.Infof("Handle Create room - sent create_room to roomworker for session id: %v", client.player.ID)
}
