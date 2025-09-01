package damasworker

import (
	"github.com/Lavizord/checkers-server/gameworker/gameworker"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

type DamasWorker struct {
	*gameworker.GameWorker
}

func New(redis *redisdb.RedisClient, db *postgrescli.PostgresCli) *DamasWorker {
	return &DamasWorker{&gameworker.GameWorker{RedisClient: redis, Db: db, GameName: "BatalhaDasDamas"}}
}
