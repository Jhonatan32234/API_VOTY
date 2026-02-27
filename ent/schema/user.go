package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").
            Unique().
            Immutable().
            DefaultFunc(func() string {
                return uuid.New().String() // Genera el ID automáticamente
            }),
        field.String("email").Unique(),
        field.String("name"),
        field.String("password").Sensitive(),
        field.Bool("active").Default(true),
        field.Time("created_at").
            Default(time.Now). // Fecha automática al crear
            Immutable(),
        field.Time("updated_at").
            Default(time.Now). // Fecha inicial
            UpdateDefault(time.Now), // Se actualiza sola
    }
}


func (User) Edges() []ent.Edge {
    return []ent.Edge{
        // Añade esto para que User sepa que tiene muchos votos
        edge.To("votes", Vote.Type),
    }
}