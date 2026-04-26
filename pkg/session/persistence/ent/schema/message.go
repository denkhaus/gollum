package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Message holds the schema definition for the Message entity.
type Message struct {
	ent.Schema
}

// Fields of the Message.
func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.New()).
			Default(uuid.New).
			Unique(),
		field.UUID("session_id", uuid.New()),
		field.Enum("type").
			Values("user_chat", "agent_chat", "tool_request", "tool_response", "thinking", "system_info", "error"),
		field.Text("content"),
		field.Time("timestamp").
			Default(time.Now).
			Immutable(),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the Message.
func (Message) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("session", Session.Type).
			Ref("messages").
			Field("session_id").
			Unique().
			Required(),
	}
}

// Indexes of the Message.
func (Message) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id"),
		index.Fields("timestamp"),
	}
}
