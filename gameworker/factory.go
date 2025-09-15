package main

import (
	"github.com/Lavizord/checkers-server/gameworkers/chessworker"
	"github.com/Lavizord/checkers-server/gameworkers/damasworker"
	"github.com/Lavizord/checkers-server/gameworkers/gameworker"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

func NewWorker(gameName string, redis *redisdb.RedisClient, db *postgrescli.PostgresCli) gameworker.Worker {
	switch gameName {
	case "BatalhaDasDamas":
		return damasworker.New(redis, db)
	case "BatalhaDoChess":
		return chessworker.New(redis, db)
	default:
		return nil
	}
}
