package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Session holds the schema definition for the Session entity.
type Session struct {
	ent.Schema
}

// Fields of the Session.
func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.New()).
			Default(uuid.New).
			Unique(),
		field.UUID("session_id", uuid.New()).
			Unique(),
		field.UUID("channel_id", uuid.New()),
		field.UUID("agent_id", uuid.New()),
		field.String("cwd").
			Default("."),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("closed_at").
			Optional().
			Nillable(),
		field.Enum("state").
			Values("active", "closed", "archived").
			Default("active"),
	}
}

// Edges of the Session.
func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("messages", Message.Type),
		edge.To("supervisor", SupervisorConfig.Type).
			Unique(),
	}
}

// Indexes of the Session.
func (Session) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id"),
		index.Fields("channel_id"),
	}
}
