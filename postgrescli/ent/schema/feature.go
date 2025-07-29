package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Feature struct {
	ent.Schema
}

func (Feature) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),

		field.String("external_id"),
	}
}

func (Feature) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("game_features", GameFeature.Type),
		edge.To("serie_features", SerieFeature.Type),
	}
}
