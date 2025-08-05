package postgrescli

import (
	"context"
	"fmt"

	"github.com/Lavizord/checkers-server/postgrescli/ent"
)

type OperatorSeed struct {
	Name  string
	Alias string
}

type PlatformSeed struct {
	Name      string
	Hash      string
	HbPayload string
	Operators []OperatorSeed
}

var seed = []PlatformSeed{
	{
		Name:      "SokkerDuel",
		Hash:      "",
		HbPayload: "",
		Operators: []OperatorSeed{
			{Name: "SokkerDuel", Alias: "SokkerPro"},
		},
	},
	{
		Name:      "TestOp",
		Hash:      "",
		HbPayload: "",
		Operators: []OperatorSeed{
			{Name: "TestOp", Alias: "TestOp"},
		},
	},
}

type GameSeed struct {
	Name          string
	TrademarkName string
}
type GameVersionSeed struct {
	Version string
	CanDemo bool
	Game    GameSeed
}
type MathVersionSeed struct {
	Name           string
	Version        string
	UrlReleaseNote string
	Volatility     int
	Rtp            int
	MaxWin         int
}
type CurrencySeed struct {
	Name           string
	Symbol         string
	SymbolPos      string
	ThouSeparator  string
	UnitsSeparator string
}
type CurrencyVersionSeed struct {
	Name        string
	Denominator int
	Currency    CurrencySeed
}
type GameConfigSeed struct {
	OperatorID      int
	GameID          int
	GameVersion     GameVersionSeed
	MathVersion     MathVersionSeed
	CurrencyVersion CurrencyVersionSeed
}

var seed2 = []GameConfigSeed{
	{
		OperatorID: 1,
		GameID:     1,
		GameVersion: GameVersionSeed{
			Version: "SokkerDuel",
			CanDemo: true,
			Game:    GameSeed{Name: "Damas", TrademarkName: "Batalha das Damas"},
		},
		MathVersion: MathVersionSeed{
			Name: "damas", Version: "1", Volatility: 0, UrlReleaseNote: "", Rtp: 0, MaxWin: 0,
		},
		CurrencyVersion: CurrencyVersionSeed{
			Name: "BRL", Denominator: 100,
			Currency: CurrencySeed{Name: "BRL", Symbol: "R$", SymbolPos: "right", ThouSeparator: ".", UnitsSeparator: ","},
		},
	},
}

func (pc *PostgresCli) SeedDb() error {
	ctx := context.Background()

	for _, p := range seed {
		// create platform
		platform, err := pc.EntCli.Platform.
			Create().
			SetName(p.Name).
			SetHash(p.Hash).
			SetHomeButtonPayload(p.HbPayload).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed seeding platform %s: %w", p.Name, err)
		}

		// create related operators
		for _, o := range p.Operators {
			_, err := pc.EntCli.Operator.
				Create().
				SetName(o.Name).
				SetAlias(o.Alias).
				SetPlatformsID(platform.ID). // link to created platform
				Save(ctx)
			if err != nil {
				return fmt.Errorf("failed seeding operator %s: %w", o.Name, err)
			}
		}
	}

	err := pc.SeedGameConfig()
	if err != nil {
		return fmt.Errorf("failed seeding gameConfigs %s", err)
	}
	return nil
}

func (pc *PostgresCli) SeedGameConfig() error {
	ctx := context.Background()

	for _, g := range seed2 {
		// Currency
		currency, err := pc.EntCli.Currency.
			Create().
			SetName(g.CurrencyVersion.Currency.Name).
			SetSymbol(g.CurrencyVersion.Currency.Symbol).
			SetSymbolPosition(g.CurrencyVersion.Currency.SymbolPos).
			SetThousandsSeparator(g.CurrencyVersion.Currency.ThouSeparator).
			SetDenominator(g.CurrencyVersion.Denominator).
			SetUnitsSeparator(g.CurrencyVersion.Currency.UnitsSeparator).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed seeding Currency %s: %w", g.CurrencyVersion.Currency.Name, err)
		}

		// CurrencyVersion
		cv, err := pc.EntCli.CurrencyVersion.
			Create().
			SetName(g.CurrencyVersion.Name).
			SetDenominator(g.CurrencyVersion.Denominator).
			SetCurrencyID(currency.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed seeding CurrencyVersion %s: %w", g.CurrencyVersion.Name, err)
		}

		// Game
		game, err := pc.EntCli.Game.
			Create().
			SetName(g.GameVersion.Game.Name).
			SetTrademarkName(g.GameVersion.Game.TrademarkName).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed seeding Game %s: %w", g.GameVersion.Game.Name, err)
		}

		// GameVersion
		gv, err := pc.EntCli.GameVersion.
			Create().
			SetVersion(g.GameVersion.Version).
			SetCanDemo(g.GameVersion.CanDemo).
			SetGamesID(game.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed seeding GameVersion %s: %w", g.GameVersion.Version, err)
		}

		// MathVersion
		mv, err := pc.EntCli.MathVersion.
			Create().
			SetName(g.MathVersion.Name).
			SetVersion(g.MathVersion.Version).
			SetVolatility(g.MathVersion.Volatility).
			SetURLReleaseNote(g.MathVersion.UrlReleaseNote).
			SetRtp(g.MathVersion.Rtp).
			SetMaxWin(g.MathVersion.MaxWin).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed seeding MathVersion %s: %w", g.MathVersion.Name, err)
		}

		// GameConfig linking everything
		_, err = pc.EntCli.GameConfig.
			Create().
			SetGameVersionsID(gv.ID).
			SetGamesID(g.GameID).
			SetOperatorID(g.OperatorID).
			SetMathVersionsID(mv.ID).
			SetCurrencyVersionsID(cv.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed seeding GameConfig %s: %w", gv.ID, err)
		}
	}

	return nil
}

func rollback(tx *ent.Tx, err error) error {
	_ = tx.Rollback()
	return err
}
