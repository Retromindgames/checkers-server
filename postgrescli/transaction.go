package postgrescli

import (
	"context"
	"fmt"
	"log"

	"github.com/Lavizord/checkers-server/models"
)


func (pc *PostgresCli) SaveTransaction(session models.Session, betData models.SokkerDuelBet, apiError error, gameID string) error {
	ctx := context.Background()
	_, err := pgs.EntCli.Transaction.
		Create().
		SetType("bet").
		SetAmount(int(betData.Amount)).
		SetCurrency(session.Currency).
		SetPlatform(session.).
		SetOperator(session.).
		SetClient(session.).
		SetGame(session.).
		SetStatus().
		SetDescription().
		SetMathProfile("").
		SetDenominator().
		SetFinalBalance().
		SetSeqID().
		SetMultiplier().
		SetGameService("PLACEHOLDER").
		SetToken(session.Token).
		SetOriginalAmount().
		Save(ctx)

	if err != nil {
		log.Printf("[PostgresCli] - error saving session: %v", err)
		return fmt.Errorf("ent: save session: %w", err)
	}
	return nil
}
