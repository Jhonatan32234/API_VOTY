package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Vote holds the schema definition for the Vote entity.
type Vote struct {
	ent.Schema
}

func (Vote) Indexes() []ent.Index {
    return []ent.Index{
        // Crea una restricción única: Un usuario solo un voto por encuesta
        index.Edges("user", "poll").Unique(),
    }
}

func (Vote) Fields() []ent.Field {
    return []ent.Field{
        // Usamos un string para el ID si vienes usando UUIDs, o deja que Ent use un int
        field.Time("created_at").
            Default(time.Now).
            Immutable(), // El voto no se puede cambiar de fecha
    }
}

// Edges of the Vote.
func (Vote) Edges() []ent.Edge {
    return []ent.Edge{
        // Quién emitió el voto
        edge.From("user", User.Type).
            Ref("votes").
            Unique().
            Required(),
        
        // En qué encuesta participó
        edge.From("poll", Poll.Type).
            Ref("votes").
            Unique().
            Required().
            // AÑADE ESTO:
            Annotations(entsql.Annotation{
                OnDelete: entsql.Cascade,
            }),
        
        // Qué opción específica seleccionó
        edge.From("poll_option", PollOption.Type).
            Ref("votes").
            Unique().
            Required(),
    }
}