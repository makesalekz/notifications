package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// LastReadNotification holds the schema definition for the LastReadNotification entity.
type LastReadNotification struct {
	ent.Schema
}

// Fields of the LastReadNotification.
func (LastReadNotification) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Immutable().Unique(),
		field.Int64("last_read_id"),
	}
}

// Edges of the LastReadNotification.
func (LastReadNotification) Edges() []ent.Edge {
	return nil
}

func (LastReadNotification) Indexes() []ent.Index {
	return nil
}
