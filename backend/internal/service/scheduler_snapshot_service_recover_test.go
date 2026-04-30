//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type schedulerSnapshotCacheRecorder struct {
	snapshots        map[string][]Account
	setSnapshotCalls int
	lastBucket       SchedulerBucket
	lastAccounts     []Account
}

func (c *schedulerSnapshotCacheRecorder) GetSnapshot(_ context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	if c.snapshots == nil {
		return nil, false, nil
	}
	items, ok := c.snapshots[bucket.String()]
	if !ok {
		return nil, false, nil
	}
	out := make([]*Account, 0, len(items))
	for i := range items {
		cloned := items[i]
		out = append(out, &cloned)
	}
	return out, true, nil
}

func (c *schedulerSnapshotCacheRecorder) SetSnapshot(_ context.Context, bucket SchedulerBucket, accounts []Account) error {
	if c.snapshots == nil {
		c.snapshots = make(map[string][]Account)
	}
	c.setSnapshotCalls++
	c.lastBucket = bucket
	c.lastAccounts = append([]Account(nil), accounts...)
	c.snapshots[bucket.String()] = append([]Account(nil), accounts...)
	return nil
}

func (c *schedulerSnapshotCacheRecorder) GetAccount(_ context.Context, _ int64) (*Account, error) {
	return nil, nil
}
func (c *schedulerSnapshotCacheRecorder) SetAccount(_ context.Context, _ *Account) error { return nil }
func (c *schedulerSnapshotCacheRecorder) DeleteAccount(_ context.Context, _ int64) error { return nil }
func (c *schedulerSnapshotCacheRecorder) UpdateLastUsed(_ context.Context, _ map[int64]time.Time) error {
	return nil
}
func (c *schedulerSnapshotCacheRecorder) TryLockBucket(_ context.Context, _ SchedulerBucket, _ time.Duration) (string, bool, error) {
	return "lock", true, nil
}
func (c *schedulerSnapshotCacheRecorder) UnlockBucket(_ context.Context, _ SchedulerBucket, _ string) error {
	return nil
}
func (c *schedulerSnapshotCacheRecorder) ListBuckets(_ context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}
func (c *schedulerSnapshotCacheRecorder) GetOutboxWatermark(_ context.Context) (int64, error) {
	return 0, nil
}
func (c *schedulerSnapshotCacheRecorder) SetOutboxWatermark(_ context.Context, _ int64) error {
	return nil
}

type snapshotAccountRepoStub struct {
	*mockAccountRepoForPlatform
}

func (r *snapshotAccountRepoStub) ListByPlatform(_ context.Context, platform string) ([]Account, error) {
	out := make([]Account, 0)
	for _, acc := range r.accounts {
		if acc.Platform == platform && acc.Status == StatusActive {
			out = append(out, acc)
		}
	}
	return out, nil
}

func TestSchedulerSnapshotService_AutoRecoversRateLimitedAccountWithoutSnapshotRebuild(t *testing.T) {
	ctx := context.Background()

	resetAt := time.Now().Add(200 * time.Millisecond)
	account := Account{
		ID:               90001,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Priority:         10,
		Concurrency:      1,
		RateLimitResetAt: &resetAt,
	}

	repo := &snapshotAccountRepoStub{
		mockAccountRepoForPlatform: &mockAccountRepoForPlatform{
			accounts: []Account{account},
		},
	}
	cache := &schedulerSnapshotCacheRecorder{}
	svc := NewSchedulerSnapshotService(cache, nil, repo, nil, nil)

	accounts, _, err := svc.ListSchedulableAccounts(ctx, nil, PlatformOpenAI, false)
	require.NoError(t, err)
	require.Len(t, accounts, 0, "rate-limited account should be filtered at runtime")

	require.Equal(t, 1, cache.setSnapshotCalls, "first call should populate snapshot cache")
	require.Equal(t, cache.lastBucket.Platform, PlatformOpenAI)
	require.Len(t, cache.lastAccounts, 1, "snapshot should store pool members (including rate-limited)")
	require.Equal(t, int64(90001), cache.lastAccounts[0].ID)

	// Wait until resetAt passes; no data change and no snapshot rebuild should be required.
	time.Sleep(300 * time.Millisecond)

	accountsAfter, _, err := svc.ListSchedulableAccounts(ctx, nil, PlatformOpenAI, false)
	require.NoError(t, err)
	require.Len(t, accountsAfter, 1, "account should auto-recover into schedulable candidates after reset time")
	require.Equal(t, int64(90001), accountsAfter[0].ID)

	require.Equal(t, 1, cache.setSnapshotCalls, "second call should hit cache and not rebuild snapshot")
}
