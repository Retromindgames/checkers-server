package main

import (
	"checkers-server/config"
	"checkers-server/messages"
	"checkers-server/models"
	"checkers-server/redisdb"
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
		err = redisClient.AddPlayer(player1)
		err = redisClient.AddPlayer(player2)
		err = redisClient.AddGame(game)
		msg, err := messages.GenerateGameStartMessage(*game)
		fmt.Printf("[%s-%d] - (Process Game Creation) - Message to publish: %v\n", name, pid, msg)
		redisClient.PublishToGamePlayer(game.Players[0], string(msg))
		redisClient.PublishToGamePlayer(game.Players[1], string(msg))
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
				

		// TODO: Validate move.
		// TODO: Save Move to redis.
		// TODO: I should change the active player
		msg, err := messages.GenerateMoveMessage(move)
		if err != nil {
			fmt.Printf("[%s-%d] - (Process Game Moves) - Failed to generate message: %v\n", name, pid, msg)
		}
		fmt.Printf("[%s-%d] - (Process Game Moves) - Message to publish: %v\n", name, pid, msg)
		//redisClient.PublishToGame(*game, string(msg)) This wasnt working...
		redisClient.PublishToGamePlayer(*opponent, string(msg))
		go startTurnTimer(game) // Restart turn timer
		
	}
}


func startTurnTimer(game *models.Game) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timer := 30 // seconds
	for timer > 0 {
		select {
		case <-ticker.C:
			// TODO: Maybe I'l need to fetch game from redis?
			msg, err := messages.GenerateGameTimerMessage(*game, timer)
			if err != nil {
				fmt.Printf("[%s-%d] - (Process Game Moves) - Failed to generate message: %v\n", name, pid, msg)
			}
			// Publish countdown update to Redis
			redisClient.PublishToGamePlayer(game.Players[0], string(msg))
			redisClient.PublishToGamePlayer(game.Players[1], string(msg))

			timer--
		}
	}

	// TODO: Handle timeout (e.g., force turn switch)
	//timeoutMsg := fmt.Sprintf(`{"game_id":"%s", "timeout":true, "current_player":"%s"}`, game.ID, game.CurrentPlayerID)
	//redisClient.Publish("TIME_OUT", timeoutMsg)
}
