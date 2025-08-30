package main

import (
	"os"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/gameworker/gameworker"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

var pid int
var redisClient *redisdb.RedisClient
var postgresClient *postgrescli.PostgresCli
var name = "GameWorker"
var gameEngine string

func init() {
	pid = os.Getpid()
	config.LoadConfig()
	redisConData := config.Cfg.Redis
	client, err := redisdb.NewRedisClient(redisConData.Addr, redisConData.User, redisConData.Password, redisConData.Tls)
	if err != nil {
		logger.Default.Fatalf("error initalalizing redis client: %v", err)
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
		logger.Default.Fatalf("error initalalizing postgres client: %v", err)
	}
	postgresClient = sqlcliente
	logger.Default.Info("gameworker service initialized.")
}

func main() {
	defer func() {
		if redisClient != nil {
			redisClient.CloseRedisClient()
		}
	}()

	logger.Default.Infof("creating gameworker...")
	gameEngine := os.Getenv("GAME_ENGINE")
	if gameEngine == "" {
		logger.Default.Fatalf("no GAME_ENGINE env variable defined")
	}
	gw := gameworker.NewGameWorker(redisClient, postgresClient, gameEngine)
	gw.Run()

	logger.Default.Infof("gameworker created, awaiting messages...")

	select {}
}
