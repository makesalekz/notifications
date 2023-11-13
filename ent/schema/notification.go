package schema

import (
	"time"

	"gitlab.calendaria.team/services/notifications/ent/enum"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Notification holds the schema definition for the Notification entity.
type Notification struct {
	ent.Schema
}

// Fields of the Notification.
func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Positive(),
		field.String("type").GoType(enum.NotificationType("")).Default(enum.Common.Value()),
		field.String("title").NotEmpty(),
		field.String("text").NotEmpty(),
		field.Int64("event_id").Optional().Nillable(),
		field.Int64("contact_id").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the Notification.
func (Notification) Edges() []ent.Edge {
	return nil
}
