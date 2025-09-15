package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
)

type GameFeature struct {
	ent.Schema
}

func (GameFeature) Fields() []ent.Field {
	return []ent.Field{}
}

func (GameFeature) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("features", Feature.Type).
			Ref("game_features").
			Unique(),
		edge.From("games", Game.Type).
			Ref("game_features").
			Unique(),
	}
}
