package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestGroupImageProtocolModeMigrationKeepsInheritDefaultAndBackfill(t *testing.T) {
	t.Parallel()

	content, err := migrations.FS.ReadFile("101_add_group_image_protocol_mode.sql")
	require.NoError(t, err)

	sql := strings.ToLower(string(content))
	require.Contains(t, sql, "add column if not exists image_protocol_mode")
	require.Contains(t, sql, "default 'inherit'")
	require.Contains(t, sql, "set image_protocol_mode = 'inherit'")
	require.Contains(t, sql, "where image_protocol_mode is null")
	require.Contains(t, sql, "or btrim(image_protocol_mode) = ''")
}
