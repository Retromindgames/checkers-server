package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// GameConfig holds the schema definition for the GameConfig entity.
type GameConfig struct {
	ent.Schema
}

// Fields of the GameConfig.
func (GameConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Bool("can_demo"),
		field.Bool("can_tournament"),
		field.Bool("can_free_bets"),
		field.Bool("can_drop_and_wins"),
		field.Bool("can_turbo"),
		field.Bool("is_active"),
		field.Bool("can_auto_bet"),
		field.Bool("can_auto_cashout"),
		field.Bool("can_ante_bet"),
		field.Bool("can_home_button"),
	}
}

func (GameConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("math_versions", MathVersion.Type).
			Ref("game_configs").
			Unique(),

		edge.From("game_versions", GameVersion.Type).
			Ref("game_configs").
			Unique(),

		edge.From("games", Game.Type).
			Ref("game_configs").
			Unique(),

		edge.From("Operator", Operator.Type).
			Ref("game_configs").
			Unique(),

		edge.From("currency_versions", CurrencyVersion.Type).
			Ref("game_configs").
			Unique(),
	}
}
