package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PasskeyCredential struct {
	ent.Schema
}

func (PasskeyCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "passkey_credentials"},
	}
}

func (PasskeyCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (PasskeyCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Bytes("credential_id").
			NotEmpty().
			Unique(),
		field.String("name").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.JSON("credential_data", map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("last_used_at").
			Optional().
			Nillable(),
	}
}

func (PasskeyCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("passkey_credentials").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (PasskeyCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("created_at"),
	}
}
