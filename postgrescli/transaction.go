package postgrescli

import (
	"context"
	"fmt"
	"log"

	"github.com/Lavizord/checkers-server/models"
)

func (pc *PostgresCli) SaveTransaction(session models.Session, gc models.GameConfigDTO, apiError error, tranType string, status, betAmount, originalAmount, finalBalance int) error {
	ctx := context.Background()
	_, err := pc.EntCli.Transaction.
		Create().
		SetType(tranType).
		SetAmount(betAmount).
		SetCurrency(session.Currency).
		SetPlatform(session.PlatformName).
		SetOperator(session.OperatorName).
		SetClient(session.ClientID).
		SetGame(gc.GameName).
		SetStatus(status).
		SetDescription(apiError.Error()).
		SetMathProfile("PLACEHOLDER").
		SetDenominator(gc.CurrencyDenominator).
		SetFinalBalance(finalBalance).
		SetSeqID(0).
		SetMultiplier(gc.CurrencyDefaultMultiplier).
		SetGameService(nil).
		SetToken(session.Token).
		SetOriginalAmount(originalAmount).
		Save(ctx)

	if err != nil {
		log.Printf("[PostgresCli] - error saving session: %v", err)
		return fmt.Errorf("ent: save session: %w", err)
	}
	return nil
}
