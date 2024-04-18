package schema

import (
	"time"

	"gitlab.calendaria.team/services/notifications/ent/enum"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
		field.Int64("task_id").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Int64("notification_data_id").Optional().Nillable(),
	}
}

// Edges of the Notification.
func (Notification) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("notification_data", NotificationData.Type).
			Ref("notification").
			Unique().
			Field("notification_data_id"),
	}
}

// Indexes of the Notification.
func (Notification) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "type"), // for the case of count unread notifications of type
	}
}
