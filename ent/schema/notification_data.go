package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// NotificationData holds the schema definition for the NotificationData entity.
type NotificationData struct {
	ent.Schema
}

// Fields of the NotificationData.
func (NotificationData) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("type").Optional().Nillable(),
		field.String("event").Optional().Nillable(),
		field.String("member").Optional().Nillable(),
		field.String("chat").Optional().Nillable(),
		field.String("message").Optional().Nillable(),
		field.String("contact").Optional().Nillable(),
		field.String("task").Optional().Nillable(),
		field.String("metadata").Optional().Nillable(),
		field.Int64("plural_count").Optional().Nillable(),
	}
}

// Edges of the NotificationData.
func (NotificationData) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("notification", Notification.Type).Unique(),
	}
}
