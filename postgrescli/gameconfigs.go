package postgrescli

import (
	"context"
	"fmt"

	"github.com/Lavizord/checkers-server/postgrescli/ent"
	"github.com/Lavizord/checkers-server/postgrescli/ent/currency"
	"github.com/Lavizord/checkers-server/postgrescli/ent/currencyversion"
	"github.com/Lavizord/checkers-server/postgrescli/ent/game"
	"github.com/Lavizord/checkers-server/postgrescli/ent/gameconfig"
	"github.com/Lavizord/checkers-server/postgrescli/ent/operator"
)

// TODO: Handle deleted and other conditions
func (pc *PostgresCli) GetGameConfig(gameName, operatorName, currencyName string) (*ent.GameConfig, error) {
	ctx := context.Background()
	result, err := pc.EntCli.GameConfig.
		Query().
		Where(
			gameconfig.HasGamesWith(game.NameEQ(gameName)),
			gameconfig.HasOperatorWith(operator.NameEQ(operatorName)),
			gameconfig.HasCurrencyVersionsWith(currencyversion.HasCurrencyWith(currency.NameEQ(currencyName))),
		).
		WithGames().
		WithCurrencyVersions(func(cvq *ent.CurrencyVersionQuery) {
			cvq.WithCurrency()
		}).
		WithOperator(func(oq *ent.OperatorQuery) {
			oq.WithPlatforms()
		}).
		WithGameVersions().
		WithMathVersions().
		WithCurrencyVersions(func(cvq *ent.CurrencyVersionQuery) {
			cvq.WithCurrency()
		}).
		Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("error fetching gameconfig: %v", err)
	}
	return result, nil
}

// TODO: Handle deleted and other conditions
func (pc *PostgresCli) GetAllGameConfigs() ([]*ent.GameConfig, error) {
	ctx := context.Background()
	result, err := pc.EntCli.GameConfig.
		Query().
		Where().
		WithGames().
		WithCurrencyVersions(func(cvq *ent.CurrencyVersionQuery) {
			cvq.WithCurrency()
		}).
		WithOperator(func(oq *ent.OperatorQuery) {
			oq.WithPlatforms()
		}).
		WithGameVersions().
		WithMathVersions().
		WithCurrencyVersions(func(cvq *ent.CurrencyVersionQuery) {
			cvq.WithCurrency()
		}).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("error fetching gameconfig: %v", err)
	}
	return result, nil
}
