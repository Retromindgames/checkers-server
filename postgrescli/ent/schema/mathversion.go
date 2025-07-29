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

		field.Int("volatility"),
		field.Int("rtp"),
		field.Int("max_win"),

		field.Bool("can_buy_bonus"),
		field.String("url_release_note"),
		field.Bool("deprecated"),
		field.Bool("can_ante_bet"),
	}
}

func (MathVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sessions", Session.Type),
		edge.To("game_configs", GameConfig.Type),
	}
}
