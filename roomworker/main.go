package main

import (
	"os"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

var pid int
var redisClient *redisdb.RedisClient
var postgresClient *postgrescli.PostgresCli
var name = "roomworker"

func init() {
	pid = os.Getpid()
	config.LoadConfig()
	redisConData := config.Cfg.Redis
	client, err := redisdb.NewRedisClient(redisConData.Addr, redisConData.User, redisConData.Password, redisConData.Tls)
	if err != nil {
		logger.Default.Fatalf("Error initializing Redis client: %v", err)
	}
	redisClient = client

	sqlcliente, err := postgrescli.NewPostgresCli(
		config.Cfg.Postgres.User,
		config.Cfg.Postgres.Password,
		config.Cfg.Postgres.DBName,
		config.Cfg.Postgres.Host,
		config.Cfg.Postgres.Port,
		config.Cfg.Postgres.Ssl,
	)
	if err != nil {
		logger.Default.Fatalf("Error initializing POSTGRES client: %v", err)
	}
	postgresClient = sqlcliente
}

func main() {
	logger.Default.Info("Waiting for room messages...")

	defer func() {
		if redisClient != nil {
			redisClient.CloseRedisClient()
		}
	}()

	workerCheckers := NewRoomWorker(redisClient, "BatalhaDasDamas", models.DamasValidBetAmounts)
	workerChess := NewRoomWorker(redisClient, "BatalhaDoChess", models.DamasValidBetAmounts)

	workerChess.Run()
	workerCheckers.Run()

	select {}
}
