//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveAnthropicEffort_UsesTopLevelMaxWhenNativeMissing(t *testing.T) {
	resolution := ResolveAnthropicEffort("", "max")
	require.NotNil(t, resolution.Raw)
	require.NotNil(t, resolution.Effective)
	require.Equal(t, "max", *resolution.Raw)
	require.Equal(t, "max", *resolution.Effective)
	require.Equal(t, effortSourceTopLevel, resolution.Source)
}

func TestResolveAnthropicEffort_PrefersNativeHighOverTopLevelMax(t *testing.T) {
	resolution := ResolveAnthropicEffort("high", "max")
	require.NotNil(t, resolution.Raw)
	require.NotNil(t, resolution.Effective)
	require.Equal(t, "high", *resolution.Raw)
	require.Equal(t, "high", *resolution.Effective)
	require.Equal(t, effortSourceAnthropicField, resolution.Source)
}

func TestResolveAnthropicEffortFromBody_PrefersOutputConfigEffort(t *testing.T) {
	resolution := ResolveAnthropicEffortFromBody(`{"effortLevel":"max","output_config":{"effort":"high"}}`)
	require.NotNil(t, resolution.Raw)
	require.NotNil(t, resolution.Effective)
	require.Equal(t, "high", *resolution.Raw)
	require.Equal(t, "high", *resolution.Effective)
	require.Equal(t, effortSourceAnthropicField, resolution.Source)
}
