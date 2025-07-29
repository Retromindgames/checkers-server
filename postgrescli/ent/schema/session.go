package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Session holds the schema definition for the Session entity.
type Session struct {
	ent.Schema
}

// Fields of the Session.
func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.Bool("can_demo"),
		field.String("token"),
		field.String("client_id"),

		field.Bool("demo"),

		field.Time("created_at").
			Default(time.Now), // auto-set on create
		field.Time("deleted_at").
			Optional(). // allow null
			Nillable(),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("games", Game.Type).
			Ref("sessions").
			Unique(),

		edge.From("game_versions", GameVersion.Type).
			Ref("sessions").
			Unique(),

		edge.From("Operator", Operator.Type).
			Ref("sessions").
			Unique(),

		edge.From("currency_versions", CurrencyVersion.Type).
			Ref("sessions").
			Unique(),

		edge.From("math_versions", MathVersion.Type).
			Ref("sessions").
			Unique(),
	}
}
