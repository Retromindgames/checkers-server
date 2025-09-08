package chessworker

import (
	"github.com/Lavizord/checkers-server/gameworkers/gameworker"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

type ChessWorker struct {
	*gameworker.GameWorker
}

func New(redis *redisdb.RedisClient, db *postgrescli.PostgresCli) *ChessWorker {
	return &ChessWorker{&gameworker.GameWorker{RedisClient: redis, Db: db, GameName: "BatalhaDoChess"}}
}

func (cw *ChessWorker) Run() {
	go cw.ProcessGameCreationList() // ChessWorker’s version
	go cw.ProcessGameMovesList()    // ChessWorker’s version
	go cw.ProcessLeaveGameList()
	go cw.ProcessDisconnectFromGameList()
	go cw.ProcessReconnectFromGameList()
}

func (cw *ChessWorker) ProcessGameMovesList() {
	cw.ProcessMovesLoop(cw)
}

func (cw *ChessWorker) HandleMove(game *models.Game, move models.MoveInterface, player *models.Player, piece models.PieceInterface) error {
	msg, err := messages.GenerateMoveMessage(move)
	if err != nil {
		logger.Default.Errorf("(Process Game Moves) - failed to generate move message: %+v, from game with id: %v, from player with id: %v", move, player.GameID, move.GetPlayerID())
	}
	//log.Printf("[%s-%d] - (Process Game Moves) - Message to publish: %v\n", name, pid, string(msg))
	opponent, _ := game.GetOpponentGamePlayer(move.GetPlayerID())
	cw.RedisClient.PublishToGamePlayer(*opponent, string(msg))
	cw.HandleTurnChange(game)
	// Since the move was validated and passed to the other player, its time to check for our end turn / end game conditions.
	// This means we can add the move to our game.
	game.Moves = append(game.Moves, move)

	return nil
}
