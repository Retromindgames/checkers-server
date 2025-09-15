package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Lavizord/checkers-server/api/routes"
	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/postgrescli/ent"
	"github.com/Lavizord/checkers-server/redisdb"
)

var postgresClient *postgrescli.PostgresCli
var redisClient *redisdb.RedisClient
var name = "restapi"

func init() {
	config.LoadConfig()

	redisConData := config.Cfg.Redis
	client, err := redisdb.NewRedisClient(redisConData.Addr, redisConData.User, redisConData.Password, redisConData.Tls)
	if err != nil {
		log.Fatalf("[%s-Redis] Error initializing Redis client: %v\n", name, err)
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
		logger.Default.Fatalf("[PostgreSQL] Error initializing POSTGRES client: %v\n", err)
	}
	postgresClient = sqlcliente

	err = postgresClient.CreateDb()
	if err != nil {
		logger.Default.Fatalf("error creating db: %v", err)
	} else {
		logger.Default.Infof("created db...")
	}
	err = postgresClient.SeedDb()
	if err != nil {
		logger.Default.Fatalf("error seeding db: %v", err)
	} else {
		logger.Default.Infof("seeded db...")
	}

	_, err = CacheOperators()
	if err != nil {
		logger.Default.Fatalf("error caching operators in redis: %v", err)
	} else {
		logger.Default.Infof("cached operators in redis...")
	}

	_, err = CacheGameConfigs()
	if err != nil {
		logger.Default.Fatalf("error caching game configs in redis: %v", err)
	} else {
		logger.Default.Infof("cached game configs in redis...")
	}

	logger.Default.Info("initialized api...")
}

func CacheOperators() ([]*ent.Operator, error) {
	op, err := postgresClient.GetOperators()
	if err != nil {
		return nil, err
	}
	if len(op) == 0 || op == nil {
		return nil, fmt.Errorf("failed to cache operators, none found in database.")
	}

	err = redisClient.AddOperators(op)
	if err != nil {
		return nil, fmt.Errorf("failed adding operators in redis.")
	}

	return op, nil
}

func CacheGameConfigs() ([]*ent.GameConfig, error) {
	gc, err := postgresClient.GetAllGameConfigs()
	if err != nil {
		return nil, err
	}
	if len(gc) == 0 || gc == nil {
		return nil, fmt.Errorf("failed to cache game configs, none found in database.")
	}

	err = redisClient.AddGameConfigs(gc)
	if err != nil {
		return nil, fmt.Errorf("failed adding game configs in redis.")
	}

	return gc, nil
}

func main() {
	defer func() {
		if redisClient != nil {
			redisClient.CloseRedisClient()
		}
		if postgresClient != nil {
			postgresClient.Close()
		}
	}()

	router := routes.RegisterRoutes(postgresClient, redisClient)

	port := config.FirstPortFromConfig(name)
	addrs := fmt.Sprintf(":%d", port)

	log.Printf("[API] - HTTP server starting on %d...", port)
	log.Fatal(http.ListenAndServe(addrs, router))

}
