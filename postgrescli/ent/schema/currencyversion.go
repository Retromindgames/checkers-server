package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// CurrencyVersion holds the schema definition for the CurrencyVersion entity.
type CurrencyVersion struct {
	ent.Schema
}

// Fields of the CurrencyVersion.
func (CurrencyVersion) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.Int("min_bet").Optional(),
		field.Int("max_exp").Optional(),
		field.Int("denominator"),
		field.Int("default_multiplier").Optional(),
		field.Bool("deprecated").Default(false),
		field.Int("crash_bet_increment").Optional(),
		field.Ints("slots_bet_multipliers").Optional(),
	}
}

func (CurrencyVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("Currency", Currency.Type).
			Ref("currency_versions").
			Unique(),
		edge.From("game_types", GameType.Type).
			Ref("currency_versions").
			Unique(),

		edge.To("sessions", Session.Type),
		edge.To("game_configs", GameConfig.Type),
	}
}
