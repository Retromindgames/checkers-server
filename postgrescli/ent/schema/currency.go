package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Currency holds the schema definition for the Currency entity.
type Currency struct {
	ent.Schema
}

// Fields of the Currency.
func (Currency) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("symbol"),
		field.String("thousands_separator"),
		field.String("units_separator"),
		field.String("symbol_position"),
		field.Int("denominator"),
	}
}

func (Currency) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("currency_versions", CurrencyVersion.Type),
	}
}
