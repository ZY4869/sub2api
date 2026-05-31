package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigration126AddsAntigravityOpus48Mapping(t *testing.T) {
	body, err := migrations.FS.ReadFile("126_add_antigravity_opus48_mapping.sql")
	require.NoError(t, err)

	sql := string(body)
	require.Contains(t, sql, "claude-opus-4-8")
	require.Contains(t, sql, "platform = 'antigravity'")
	require.Contains(t, strings.ToLower(sql), "jsonb_set")
	require.Contains(t, sql, "deleted_at IS NULL")
	require.Contains(t, sql, "credentials->'model_mapping'->>'claude-opus-4-8' IS NULL")
}
