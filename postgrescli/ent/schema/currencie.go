package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Currencie holds the schema definition for the Currencie entity.
type Currencie struct {
	ent.Schema
}

// Fields of the Currencie.
func (Currencie) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("symbol"),
		field.String("thousands_separator"),
		field.String("units_separator"),
		field.String("symbol_position"),
		field.Int("denominator"),
	}
}

func (Currencie) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("currency_versions", CurrencyVersion.Type),
	}
}
