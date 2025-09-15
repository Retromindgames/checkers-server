package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type GameType struct {
	ent.Schema
}

func (GameType) Fields() []ent.Field {
	return []ent.Field{

		field.String("type"),
		field.String("external_type_id"),
	}
}

func (GameType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("games", Game.Type),
		edge.To("game_versions", GameVersion.Type),
		edge.To("currency_versions", CurrencyVersion.Type),
	}
}
