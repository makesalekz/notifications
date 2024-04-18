package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Device holds the schema definition for the Device entity.
type Device struct {
	ent.Schema
}

// Fields of the Device.
func (Device) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Positive(),
		field.String("token").Immutable().Unique().MinLen(1),
		field.Time("created_at").Immutable().Default(time.Now),
		field.String("language").Optional().Default("en"),
	}
}

// Edges of the Device.
func (Device) Edges() []ent.Edge {
	return nil
}
