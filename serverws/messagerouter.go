package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Lavizord/checkers-server/messages"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"

	queueHandler "github.com/Lavizord/checkers-server/serverws/queuehandler"
)

// Directly route to the right handler based on the command.
//
// The valid command sent will be processsed and router to the right handler. Some of those handlers
// might send the message to redis.
func RouteMessages(message *messages.Message[json.RawMessage], player *models.Player, redis *redisdb.RedisClient) {
	switch message.Command {
	case "queue":
		handleQueue(message, player, redis)

	case "leave_queue":
		handleLeaveQueue(player, redis)

	case "ready_queue":
		handleReadyQueue(message, player, redis)

	case "leave_room":
		if player.Status != models.StatusInRoom {
			msg, _ := messages.GenerateGenericMessage("invalid", "Can't issue a leave room when not in a Room.")
			player.WriteChan <- msg
			return
		}
		handleLeaveRoom(player, redis)

	case "leave_game":
		if player.Status != models.StatusInGame {
			msg, _ := messages.GenerateGenericMessage("invalid", "Can't issue a leave game when not in a game.")
			player.WriteChan <- msg
			return
		}
		handleLeaveGame(player, redis)

	case "move_piece":
		if player.Status != models.StatusInGame {
			msg, _ := messages.GenerateGenericMessage("invalid", "Can't issue a move when not in a Game.")
			player.WriteChan <- msg
			return
		}
		handleMovePiece(message, player, redis)
	}
}

func handleQueue(msg *messages.Message[json.RawMessage], player *models.Player, redis *redisdb.RedisClient) {
	qh := queueHandler.NewQueueHandler(player, redis, msg)
	qh.Process()
}

func handleLeaveQueue(player *models.Player, redis *redisdb.RedisClient) {
	if player.UpdatePlayerStatus(models.StatusOnline) != nil {
		msg, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'Online'")
		player.WriteChan <- msg
		return
	}
	player.SetStatusOnline()
	redis.UpdatePlayer(player) // This is important, we will only re-add players to a queue that are in queue.
	redis.UpdatePlayersInQueueSet(player.ID, models.StatusOnline)
	queueName := fmt.Sprintf("queue:%f", player.SelectedBet)
	err := redis.RemovePlayerFromQueue(queueName, player)
	if err != nil {
		//log.Printf("Error removing player from Redis queue: %v\n", err)	//! This was commented, it fails when there is only 1 player in queue.
		//player.WriteChan <- []byte("Error removing player to queue")
		return
	}
	// we update out player status., send a confirmation message back to the player
	m, err := messages.GenerateQueueConfirmationMessage(false)
	if err != nil {
		msg, _ := messages.GenerateGenericMessage("error", "Error generating queue confirmation false:"+err.Error())
		log.Println("Error generating queue confirmation false:", err)
		player.WriteChan <- msg
		return
	}
	player.WriteChan <- m
}

func handleReadyQueue(msg *messages.Message[json.RawMessage], player *models.Player, redis *redisdb.RedisClient) {
	// We have to check if the message for readyqueue true or false.
	var value bool
	json.Unmarshal(msg.Value, &value)
	if value {
		// update the player status to ready / awaiting opponent.
		if player.UpdatePlayerStatus(models.StatusInRoomReady) != nil {
			msg, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'ready_queue true'")
			player.WriteChan <- msg
			return
		}
	} else {
		// update the player status, to unready / waiting ready.
		if player.UpdatePlayerStatus(models.StatusInRoom) != nil {
			msg, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'ready_queue false'")
			player.WriteChan <- msg
			return
		}
	}
	msgBytes, _ := messages.GenerateGenericMessage("info", "Processing 'ready_queue'")
	player.WriteChan <- msgBytes
	// we update our player to redis.
	redis.UpdatePlayersInQueueSet(player.ID, player.Status)
	err := redis.UpdatePlayer(player)
	if err != nil {
		log.Printf("Error updating player to Redis: %v\n", err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "Error Updating player: "+err.Error())
		player.WriteChan <- msgBytes
		return
	}
	err = redis.RPush("ready_queue", player) // now we tell roomworker to process this player ready.
	if err != nil {
		log.Printf("Error pushing player to Redis ready queue: %v\n", err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "Pushing player to queue: "+err.Error())
		player.WriteChan <- msgBytes
		return
	}
}

func handleLeaveRoom(player *models.Player, redis *redisdb.RedisClient) {
	// update the player status
	if player.UpdatePlayerStatus(models.StatusOnline) != nil {
		msgBytes, _ := messages.GenerateGenericMessage("invalid", "Invalid status transition to 'leave_room'")
		player.WriteChan <- msgBytes
		return
	}
	msgBytes, _ := messages.GenerateGenericMessage("info", "Processing 'leave_room'")
	player.WriteChan <- msgBytes
	// we update our player to redis.
	err := redis.UpdatePlayer(player)
	if err != nil {
		log.Printf("Error updagint player to Redis on HandleLeaveRoom: %v\n", err)
		msg, _ := messages.GenerateGenericMessage("error", "error updating player.")
		player.WriteChan <- msg
		return
	}
	err = redis.RPush("leave_room", player)
	if err != nil {
		log.Printf("Error pushing player to Redis leave_room queue: %v\n", err)
		msg, _ := messages.GenerateGenericMessage("error", "error leaving room.")
		player.WriteChan <- msg
		return
	}
}

func handleLeaveGame(player *models.Player, redis *redisdb.RedisClient) {
	msgBytes, _ := messages.GenerateGenericMessage("invalid", "Processing 'leave_game'")
	player.WriteChan <- msgBytes
	err := redis.RPush("leave_game", player)
	if err != nil {
		log.Printf("Error pushing player to Redis leave_game queue: %v\n", err)
		msgBytes, _ := messages.GenerateGenericMessage("error", "Error adding player to leave_game")
		player.WriteChan <- msgBytes
		return
	}
}

func handleMovePiece(message *messages.Message[json.RawMessage], player *models.Player, redis *redisdb.RedisClient) {
	var move models.Move
	err := json.Unmarshal([]byte(message.Value), &move)
	if err != nil {
		log.Printf("[Handlers] - Handle Move Piece - JSON Unmarshal Error: %v\n", err)
		msg, _ := messages.GenerateGenericMessage("invalid", "Handle Move Piece - JSON Unmarshal Error.")
		player.WriteChan <- msg
		return
	}
	if move.PlayerID != player.ID {
		log.Printf("[Handlers] - Handle Move Piece - move.PlayerID != player.ID\n")
		msg, _ := messages.GenerateGenericMessage("invalid", "Handle Move Piece - move.PlayerID != player.ID.")
		player.WriteChan <- msg
		return
	}
	// movement message is sent to the game worker
	err = redis.RPushGeneric("move_piece", message.Value)
	if err != nil {
		log.Printf("Error pushing move to Redis handleMovePiece queue: %v\n", err)
		msg, _ := messages.GenerateGenericMessage("error", "error pushing move to gameworker.")
		player.WriteChan <- msg
		return
	}
}
