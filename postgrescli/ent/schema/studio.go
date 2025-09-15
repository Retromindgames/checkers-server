package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Studio holds the schema definition for the Studio entity.
type Studio struct {
	ent.Schema
}

// Fields of the Studio.
func (Studio) Fields() []ent.Field {
	return []ent.Field{

		field.String("name"),
		field.Time("created_at").
			Default(time.Now), // auto-set on create
		field.Time("deleted_at").
			Optional(). // allow null
			Nillable(), // pointer in Go struct
	}
}

// Edges of the Studio.
func (Studio) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("games", Game.Type),
	}
}
