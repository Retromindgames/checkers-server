package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Transaction holds the schema definition for the Transaction entity.
type Transaction struct {
	ent.Schema
}

// Fields of the Transaction.
func (Transaction) Fields() []ent.Field {
	return []ent.Field{
		field.String("type").Optional(),
		field.Time("deleted_at").
			Optional(), // allow null
		field.Int("amount"),
		field.String("currency"),
		field.String("platform"),
		field.String("operator"),
		field.String("client"),
		field.String("game"),
		field.Int("status"),
		field.String("description"),
		field.Time("timestamp").
			Default(time.Now), // auto-set on create
		field.String("math_profile").Optional(),
		field.Int("denominator"),
		field.Int("final_balance"),
		field.Int("seq_id").Optional(),
		field.Int("multiplier").Optional(),
		field.JSON("game_service", map[string]interface{}{}).Optional(),
		field.String("token"),
		field.Int("original_amount"),
	}
}

// Edges of the Transaction.
func (Transaction) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("rounds", Round.Type).
			Ref("transactions").
			Unique(),
	}
}
