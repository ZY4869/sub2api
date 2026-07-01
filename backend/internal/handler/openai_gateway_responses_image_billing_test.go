//go:build unit

package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestShouldReserveResponsesImageCountRequiresImageTool(t *testing.T) {
	apiKey := &service.APIKey{
		ImageOnlyEnabled:         true,
		ImageCountBillingEnabled: true,
		ImageMaxCount:            5,
		User:                     &service.User{ID: 1, Role: service.RoleAdmin},
	}

	require.False(t, shouldReserveResponsesImageCount(apiKey, false))
	require.True(t, shouldReserveResponsesImageCount(apiKey, true))
}

func TestShouldReserveResponsesImageCountRequiresBillingEnabled(t *testing.T) {
	apiKey := &service.APIKey{
		ImageOnlyEnabled:         true,
		ImageCountBillingEnabled: false,
		ImageMaxCount:            5,
		User:                     &service.User{ID: 1, Role: service.RoleAdmin},
	}

	require.False(t, shouldReserveResponsesImageCount(apiKey, true))
}
