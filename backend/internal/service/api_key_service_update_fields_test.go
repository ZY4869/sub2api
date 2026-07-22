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

func TestApplyAPIKeyUpdateFields_PreservesIPListsWhenFieldsOmitted(t *testing.T) {
	apiKey := &APIKey{
		IPWhitelist: []string{"10.0.0.0/8"},
		IPBlacklist: []string{"192.0.2.1"},
	}

	svc := &APIKeyService{}
	svc.applyAPIKeyUpdateFields(nil, apiKey, UpdateAPIKeyRequest{})

	require.Equal(t, []string{"10.0.0.0/8"}, apiKey.IPWhitelist)
	require.Equal(t, []string{"192.0.2.1"}, apiKey.IPBlacklist)
}

func TestApplyAPIKeyUpdateFields_ClearsIPListsWhenExplicitEmptyArrays(t *testing.T) {
	apiKey := &APIKey{
		IPWhitelist: []string{"10.0.0.0/8"},
		IPBlacklist: []string{"192.0.2.1"},
	}

	svc := &APIKeyService{}
	svc.applyAPIKeyUpdateFields(nil, apiKey, UpdateAPIKeyRequest{
		IPWhitelistSet: true,
		IPBlacklistSet: true,
		IPWhitelist:    []string{},
		IPBlacklist:    []string{},
	})

	require.Empty(t, apiKey.IPWhitelist)
	require.Empty(t, apiKey.IPBlacklist)
}
