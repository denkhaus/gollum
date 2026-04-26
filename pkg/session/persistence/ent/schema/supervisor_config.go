package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// SupervisorConfig holds the schema definition for the SupervisorConfig entity.
type SupervisorConfig struct {
	ent.Schema
}

// Fields of the SupervisorConfig.
func (SupervisorConfig) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.New()).
			Default(uuid.New).
			Unique(),
		field.UUID("session_id", uuid.New()).
			Unique(),
		field.String("model").
			Default(""),
		field.Float32("temperature").
			Default(0.7),
		field.Int("max_tokens").
			Default(4096),
		field.Text("system_prompt").
			Optional(),
		field.JSON("config_json", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the SupervisorConfig.
func (SupervisorConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("session", Session.Type).
			Ref("supervisor").
			Field("session_id").
			Unique().
			Required(),
	}
}
