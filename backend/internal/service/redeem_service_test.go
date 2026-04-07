//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveNegativeRedeemSubscriptionUpdate_ShortensActiveSubscription(t *testing.T) {
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		ID:        1,
		ExpiresAt: now.AddDate(0, 0, 10),
		Status:    SubscriptionStatusActive,
	}

	expiresAt, status := resolveNegativeRedeemSubscriptionUpdate(sub, -3, now)

	require.Equal(t, now.AddDate(0, 0, 7), expiresAt)
	require.Equal(t, SubscriptionStatusActive, status)
}

func TestResolveNegativeRedeemSubscriptionUpdate_ExpiresWhenAdjustmentConsumesRemainingDays(t *testing.T) {
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		ID:        1,
		ExpiresAt: now.AddDate(0, 0, 2),
		Status:    SubscriptionStatusActive,
	}

	expiresAt, status := resolveNegativeRedeemSubscriptionUpdate(sub, -5, now)

	require.Equal(t, now, expiresAt)
	require.Equal(t, SubscriptionStatusExpired, status)
}

func TestAppendRedeemSubscriptionNote_AppendsWithNewline(t *testing.T) {
	got := appendRedeemSubscriptionNote("existing note", "new note")
	require.Equal(t, "existing note\nnew note", got)
}
