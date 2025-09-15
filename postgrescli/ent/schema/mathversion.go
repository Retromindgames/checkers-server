package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// MathVersion holds the schema definition for the MathVersion entity.
type MathVersion struct {
	ent.Schema
}

// Fields of the MathVersion.
func (MathVersion) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("version"),

		field.Int("volatility").Optional(),
		field.Int("rtp").Optional(),
		field.Int("max_win").Optional(),

		field.Bool("can_buy_bonus").Optional(),
		field.String("url_release_note").Optional(),
		field.Bool("deprecated").Default(false),
		field.Bool("can_ante_bet").Optional(),
	}
}

func (MathVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sessions", Session.Type),
		edge.To("game_configs", GameConfig.Type),
	}
}
