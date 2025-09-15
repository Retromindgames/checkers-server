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
		field.Bool("can_demo").Default(false),
		field.Bool("can_tournament").Default(false),
		field.Bool("can_free_bets").Default(false),
		field.Bool("can_drop_and_wins").Default(false),
		field.Bool("can_turbo").Default(false),
		field.String("url_media_pack").Optional(),
		field.String("url_release_note").Optional(),

		field.Bool("deprecated").Default(false),
		field.Ints("available_math_versions").Optional(),
		field.Bool("can_auto_bet").Default(false),

		field.String("url_game_manual").Optional(),
		field.Bool("can_auto_cashout").Default(false),
		field.Bool("can_buy_bonus").Default(false),
		field.Bool("can_ante_bet").Default(false),
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
