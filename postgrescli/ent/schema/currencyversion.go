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
		field.Int("min_bet"),
		field.Int("max_exp"),
		field.Int("denominator"),
		field.Int("currency_id"),
		field.Int("default_multiplier"),
		field.Bool("deprecated"),
		field.Int("crash_bet_increment"),
		field.Ints("slots_bet_multipliers"),
	}
}

func (CurrencyVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("Currencie", Currencie.Type).
			Ref("currency_versions").
			Unique(),
		edge.From("game_types", GameType.Type).
			Ref("currency_versions").
			Unique(),

		edge.To("sessions", Session.Type),
		edge.To("game_configs", GameConfig.Type),
	}
}
