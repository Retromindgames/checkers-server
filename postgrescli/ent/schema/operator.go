package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Operator holds the schema definition for the Operator entity.
type Operator struct {
	ent.Schema
}

// Fields of the Operator.
func (Operator) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.Time("created_at").
			Default(time.Now), // auto-set on create
		field.Time("deleted_at").
			Optional(). // allow null
			Nillable(),
		field.String("alias"),
	}
}

func (Operator) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("platforms", Platform.Type).
			Ref("Operator").
			Unique(),

		edge.To("sessions", Session.Type),
		edge.To("game_configs", GameConfig.Type),
	}
}
