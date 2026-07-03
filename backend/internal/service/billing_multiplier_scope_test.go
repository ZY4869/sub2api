package service

import "testing"

import "github.com/stretchr/testify/require"

func TestNormalizeExplicitRateMultiplierDefaultsNonPositiveValues(t *testing.T) {
	require.Equal(t, 1.0, normalizeExplicitRateMultiplier(0))
	require.Equal(t, 1.0, normalizeExplicitRateMultiplier(-0.5))
	require.Equal(t, 1.25, normalizeExplicitRateMultiplier(1.25))
}

func TestBillingLineActualMultiplierScopesTokenAndFlatUnits(t *testing.T) {
	require.Equal(t, 2.0, billingLineActualMultiplier(BillingUnitInputToken, 2.0, 1.5))
	require.Equal(t, 2.0, billingLineActualMultiplier(BillingUnitOutputToken, 2.0, 1.5))
	require.Equal(t, 1.5, billingLineActualMultiplier(BillingUnitImage, 2.0, 1.5))
	require.Equal(t, 1.0, billingLineActualMultiplier(BillingUnitInputToken, 0, 0))
}
