package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
)

type SerieFeature struct {
	ent.Schema
}

func (SerieFeature) Fields() []ent.Field {
	return []ent.Field{}
}

func (SerieFeature) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("features", Feature.Type).
			Ref("serie_features").
			Unique(),
		edge.From("series", Serie.Type).
			Ref("serie_features").
			Unique(),
	}
}
