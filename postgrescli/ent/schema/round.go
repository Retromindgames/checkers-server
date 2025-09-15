package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Round holds the schema definition for the Round entity.
type Round struct {
	ent.Schema
}

// Fields of the Round.
func (Round) Fields() []ent.Field {
	return []ent.Field{
		field.String("platform"),
		field.String("operator"),
		field.JSON("reels", map[string]interface{}{}).Optional(),
		field.JSON("multipliers", map[string]interface{}{}).Optional(),
		field.String("bonus_type").Optional(),
		field.Int("bonus_symbol").Optional(),
		field.Int("bonus_multiplier").Optional(),
		field.Time("timestamp").
			Default(time.Now), // auto-set on create
		field.String("round_type").Optional(),
		field.JSON("play", map[string]interface{}{}),
		field.Int("free_spins_remaining").Optional(),
		field.String("math_output").Optional(),
		field.JSON("game_service", map[string]interface{}{}).Optional(),
		field.Int("free_spins_count").Optional(),
		field.Bool("ante_bet").Optional(),
		field.String("buy_bonus").Optional(),
		field.Int("character").Optional(),
	}
}

// Edges of the Round.
func (Round) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("transactions", Transaction.Type),
	}
}
