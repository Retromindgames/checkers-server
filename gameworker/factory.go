package main

import (
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

func NewWorker(gameName string, redis *redisdb.RedisClient, db *postgrescli.PostgresCli) core.Worker {
	switch gameName {
	case "BatalhaDasDamas":
		return damas.New(redis, db)
	case "BatalhaDoChess":
		return chess.New(redis, db)
	default:
		return nil
	}
}
