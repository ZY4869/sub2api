//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPeriodicJobLeaderGate_SimpleModeRunsWithoutRedis(t *testing.T) {
	gate := NewPeriodicJobLeaderGate(nil, &config.Config{RunMode: config.RunModeSimple})
	calls := 0

	ok := gate.RunIfLeader(context.Background(), "unit_job", time.Minute, func(context.Context) {
		calls++
	})

	require.True(t, ok)
	require.Equal(t, 1, calls)
}

func TestPeriodicJobLeaderGate_StandardModeSkipsWithoutRedis(t *testing.T) {
	gate := NewPeriodicJobLeaderGate(nil, &config.Config{RunMode: config.RunModeStandard})
	calls := 0

	ok := gate.RunIfLeader(context.Background(), "unit_job", time.Minute, func(context.Context) {
		calls++
	})

	require.False(t, ok)
	require.Zero(t, calls)
}
