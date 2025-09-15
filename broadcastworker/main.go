package main

import (
	"fmt"
	"log"
	"os"
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

	logger.Default.Infof("creating gameworker...")
	gameEngine := os.Getenv("GAME_ENGINE")
	if gameEngine == "" {
		logger.Default.Fatalf("no GAME_ENGINE env variable defined, exiting")
	}

	logger.Default.Infof("broadcastworker loaded timer configuration...")
	for range ticker.C {
		key := fmt.Sprintf("game_info:{%v}", gameEngine)

		msgChess, _ := messages.GenerateGameInfoMessageBytes(redisClient, gameEngine)
		err := redisClient.Publish(key, msgChess)
		if err != nil {
			log.Printf("[BroadcastWorker] Error publishing message: %v\n", err)
		} else {
			//log.Printf("[BroadcastWorker] Published room aggregates")
		}

	}
}
