package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Serie struct {
	ent.Schema
}

func (Serie) Fields() []ent.Field {
	return []ent.Field{

		field.String("name"),

		field.Time("created_at").
			Default(time.Now), // auto-set on create
		field.Time("deleted_at").
			Optional(). // allow null
			Nillable(), // pointer in Go struct

		field.String("external_id"),
	}
}

func (Serie) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("games", Game.Type),
		edge.To("serie_features", SerieFeature.Type),
	}
}
