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
		field.String("token").Unique(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the Device.
func (Device) Edges() []ent.Edge {
	return nil
}
