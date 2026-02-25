package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// PollOption holds the schema definition for the PollOption entity.
type PollOption struct {
	ent.Schema
}

func (PollOption) Fields() []ent.Field {
    return []ent.Field{
        field.String("text"),
        field.Int("votes_count").Default(0),
    }
}

func (PollOption) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("poll", Poll.Type).Ref("options").Unique(),
        edge.To("votes", Vote.Type),   
	}
}