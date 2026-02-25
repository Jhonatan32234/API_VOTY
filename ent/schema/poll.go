package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Poll holds the schema definition for the Poll entity.
type Poll struct {
	ent.Schema
}

func (Poll) Fields() []ent.Field {
    return []ent.Field{
        field.String("title"),
        field.Bool("is_open").Default(true),
        field.Time("created_at").Default(time.Now),
    }
}
func (Poll) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("options", PollOption.Type),
        edge.To("votes", Vote.Type),
    }
}