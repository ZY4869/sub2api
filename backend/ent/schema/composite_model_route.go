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

// CompositeModelRoute stores model-level routing rules for composite groups.
type CompositeModelRoute struct {
	ent.Schema
}

func (CompositeModelRoute) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "composite_model_routes"},
	}
}

func (CompositeModelRoute) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (CompositeModelRoute) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("parent_group_id"),
		field.String("display_model_id").
			MaxLen(191).
			NotEmpty(),
		field.Int64("target_group_id"),
		field.String("target_model_id").
			MaxLen(191).
			Default(""),
		field.Int("priority").
			Default(50),
		field.Bool("enabled").
			Default(true),
		field.String("notes").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
	}
}

func (CompositeModelRoute) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("parent_group", Group.Type).
			Unique().
			Required().
			Field("parent_group_id"),
		edge.To("target_group", Group.Type).
			Unique().
			Required().
			Field("target_group_id"),
	}
}

func (CompositeModelRoute) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("parent_group_id"),
		index.Fields("target_group_id"),
		index.Fields("parent_group_id", "display_model_id", "enabled", "priority"),
	}
}
