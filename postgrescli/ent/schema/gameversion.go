package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// GameVersion holds the schema definition for the GameVersion entity.
type GameVersion struct {
	ent.Schema
}

// Fields of the GameVersion.
func (GameVersion) Fields() []ent.Field {
	return []ent.Field{

		field.String("version"),
		field.Bool("can_demo"),
		field.Bool("can_tournament"),
		field.Bool("can_free_bets"),
		field.Bool("can_drop_and_wins"),
		field.Bool("can_turbo"),
		field.String("url_media_pack"),
		field.String("url_release_note"),

		field.Bool("deprecated"),
		field.Ints("available_math_versions"),
		field.Bool("can_auto_bet"),

		field.String("url_game_manual"),
		field.Bool("can_auto_cashout"),
		field.Bool("can_buy_bonus"),
		field.Bool("can_ante_bet"),
	}
}

func (GameVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("games", Game.Type).
			Ref("game_versions").
			Unique(),

		edge.From("game_type", GameType.Type).
			Ref("game_versions").
			Unique(),

		edge.To("sessions", Session.Type),
		edge.To("game_configs", GameConfig.Type),
	}
}
