package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Platform holds the schema definition for the Platform entity.
type Platform struct {
	ent.Schema
}

// Fields of the Platform.
func (Platform) Fields() []ent.Field {
	return []ent.Field{

		field.String("name"),
		field.String("hash"),
		field.Time("created_at").
			Default(time.Now), // auto-set on create
		field.Time("deleted_at").
			Optional(). // allow null
			Nillable(),
		field.String("home_button_payload"),
	}
}

func (Platform) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("Operator", Operator.Type),
	}
}
