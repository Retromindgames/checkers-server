package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Game holds the schema definition for the Game entity.
type Game struct {
	ent.Schema
}

// Fields of the Game.
func (Game) Fields() []ent.Field {
	return []ent.Field{

		field.String("name"),
		field.String("external_id"),
		field.String("trademark_name"),
	}
}

func (Game) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("studio", Studio.Type).
			Ref("games").
			Unique(), // one studio per game

		edge.From("game_type", GameType.Type).
			Ref("games").
			Unique(), // one game_type per game

		edge.From("serie", Serie.Type).
			Ref("games").
			Unique(), // one serie per game

		edge.To("game_features", GameFeature.Type),
		edge.To("game_versions", GameVersion.Type),
		edge.To("game_configs", GameConfig.Type),
		edge.To("sessions", Session.Type),
	}
}
