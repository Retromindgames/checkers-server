package main

import (
	"log"
	"time"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/redisdb"
)

var redisClient *redisdb.RedisClient

func init() {
	config.LoadConfig()
	redisConData := config.Cfg.Redis
	client, err := redisdb.NewRedisClient(redisConData.Addr, redisConData.User, redisConData.Password, redisConData.Tls)
	if err != nil {
		logger.Default.Fatalf("[BroadcastWorker][Redis] Error initializing Redis client: %v", err)
	}
	redisClient = client
	logger.Default.Infof("broadcastworker initialized...")
}

func main() {
	logger.Default.Infof("broadcastworker starting up...")
	ticker := time.NewTicker(time.Duration(config.Cfg.Services["broadcastworker"].Timer) * time.Second)
	logger.Default.Infof("broadcastworker loaded timer configuration...")
	defer func() {
		ticker.Stop()
		if redisClient != nil {
			redisClient.CloseRedisClient()
		}
	}()

	logger.Default.Infof("broadcastworker loaded timer configuration...")
	for range ticker.C {
		keyCheckers := "game_info:{BatalhaDasDamas}"
		keyChess := "game_info:{BatalhaDoChess}"
		msg, _ := messages.GenerateGameInfoMessageBytes(redisClient, "BatalhaDasDamas")
		// Publish the message
		err := redisClient.Publish(keyCheckers, msg)
		if err != nil {
			log.Printf("[BroadcastWorker] Error publishing message: %v\n", err)
		} else {
			//log.Printf("[BroadcastWorker] Published room aggregates")
		}

		msgChess, _ := messages.GenerateGameInfoMessageBytes(redisClient, "BatalhaDoChess")
		err = redisClient.Publish(keyChess, msgChess)
		if err != nil {
			log.Printf("[BroadcastWorker] Error publishing message: %v\n", err)
		} else {
			//log.Printf("[BroadcastWorker] Published room aggregates")
		}
	}
}
