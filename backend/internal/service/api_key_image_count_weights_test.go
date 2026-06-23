package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAPIKeyImageCountWeights_DefaultsAndKnownTiers(t *testing.T) {
	got := NormalizeAPIKeyImageCountWeights(map[string]int{
		"1k": 3,
		"2K": 0,
		"4K": 5,
		"8K": 9,
	})

	require.Equal(t, map[string]int{"1K": 3, "2K": 1, "4K": 5}, got)
}

func TestAPIKey_ImageCountUnitsForTier_UsesConfiguredWeights(t *testing.T) {
	key := &APIKey{ImageCountWeights: map[string]int{"1K": 1, "2K": 2, "4K": 4}}

	require.Equal(t, 8, key.ImageCountUnitsForTier(2, "4k"))
	require.Equal(t, 4, key.ImageCountUnitsForTier(2, "auto"))
	require.Equal(t, 0, key.ImageCountUnitsForTier(0, "4K"))
}

func TestAPIKey_EffectiveImageCountBillingRequiresAdminOwner(t *testing.T) {
	base := APIKey{
		ImageOnlyEnabled:         true,
		ImageCountBillingEnabled: true,
		ImageMaxCount:            10,
	}

	userKey := base
	userKey.User = &User{ID: 7, Role: RoleUser}
	require.False(t, userKey.EffectiveImageCountBillingEnabled())

	adminKey := base
	adminKey.User = &User{ID: 1, Role: RoleAdmin}
	require.True(t, adminKey.EffectiveImageCountBillingEnabled())
}
