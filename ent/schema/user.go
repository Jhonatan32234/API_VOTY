package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("email").Unique(),
		field.String("name"),
		field.String("password").Sensitive(),
		field.Bool("active").Default(true),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (User) Edges() []ent.Edge {
    return []ent.Edge{
        // Añade esto para que User sepa que tiene muchos votos
        edge.To("votes", Vote.Type),
    }
}