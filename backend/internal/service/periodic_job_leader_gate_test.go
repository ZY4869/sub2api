//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPeriodicJobLeaderTTLBounds(t *testing.T) {
	require.Equal(t, defaultPeriodicJobLeaderLockTTL, periodicJobLeaderTTL(time.Second))
	require.Equal(t, 5*time.Minute, periodicJobLeaderTTL(time.Minute))
	require.Equal(t, maxPeriodicJobLeaderLockTTL, periodicJobLeaderTTL(24*time.Hour))
}
