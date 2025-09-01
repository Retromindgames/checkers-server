package chessworker

import (
	"github.com/Lavizord/checkers-server/gameworker/gameworker"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

type ChessWorker struct {
	*gameworker.GameWorker
}

func New(redis *redisdb.RedisClient, db *postgrescli.PostgresCli) *ChessWorker {
	return &ChessWorker{&gameworker.GameWorker{RedisClient: redis, Db: db, GameName: "BatalhaDoChess"}}
}
