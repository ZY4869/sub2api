package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PasskeyUserHandle struct {
	ent.Schema
}

func (PasskeyUserHandle) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "passkey_user_handles"},
	}
}

func (PasskeyUserHandle) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (PasskeyUserHandle) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Unique(),
		field.Bytes("user_handle").
			NotEmpty().
			Unique(),
	}
}

func (PasskeyUserHandle) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("passkey_user_handles").
			Field("user_id").
			Unique().
			Required(),
	}
}
