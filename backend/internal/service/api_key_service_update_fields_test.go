//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyAPIKeyUpdateQuotaFields_RestoresQuotaExhaustedWhenQuotaBecomesUnlimited(t *testing.T) {
	quota := 0.0
	apiKey := &APIKey{
		Status:    StatusAPIKeyQuotaExhausted,
		Quota:     100,
		QuotaUsed: 100,
	}

	applyAPIKeyUpdateQuotaFields(apiKey, UpdateAPIKeyRequest{Quota: &quota})

	require.Equal(t, StatusActive, apiKey.Status)
	require.Equal(t, 0.0, apiKey.Quota)
}

func TestApplyAPIKeyUpdateQuotaFields_RestoresQuotaExhaustedWhenQuotaExceedsUsage(t *testing.T) {
	quota := 120.0
	apiKey := &APIKey{
		Status:    StatusAPIKeyQuotaExhausted,
		Quota:     100,
		QuotaUsed: 100,
	}

	applyAPIKeyUpdateQuotaFields(apiKey, UpdateAPIKeyRequest{Quota: &quota})

	require.Equal(t, StatusActive, apiKey.Status)
	require.Equal(t, 120.0, apiKey.Quota)
}

func TestApplyAPIKeyUpdateQuotaFields_KeepsQuotaExhaustedWhenQuotaStillUsedUp(t *testing.T) {
	quota := 100.0
	apiKey := &APIKey{
		Status:    StatusAPIKeyQuotaExhausted,
		Quota:     100,
		QuotaUsed: 100,
	}

	applyAPIKeyUpdateQuotaFields(apiKey, UpdateAPIKeyRequest{Quota: &quota})

	require.Equal(t, StatusAPIKeyQuotaExhausted, apiKey.Status)
}
