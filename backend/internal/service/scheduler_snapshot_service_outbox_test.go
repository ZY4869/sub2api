//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type schedulerOutboxRepoStub struct {
	events []SchedulerOutboxEvent
}

func (r *schedulerOutboxRepoStub) ListAfter(context.Context, int64, int) ([]SchedulerOutboxEvent, error) {
	return append([]SchedulerOutboxEvent(nil), r.events...), nil
}

func (r *schedulerOutboxRepoStub) MaxID(context.Context) (int64, error) {
	if len(r.events) == 0 {
		return 0, nil
	}
	return r.events[len(r.events)-1].ID, nil
}

type schedulerSnapshotCacheSpy struct {
	snapshotBuckets    []SchedulerBucket
	watermarkCalls     int
	watermarkDeadlines []time.Time
	watermarkSucceedOn int
	lastWatermark      int64
}

func (c *schedulerSnapshotCacheSpy) GetSnapshot(context.Context, SchedulerBucket) ([]*Account, bool, error) {
	return nil, false, nil
}

func (c *schedulerSnapshotCacheSpy) SetSnapshot(_ context.Context, bucket SchedulerBucket, _ []Account) error {
	c.snapshotBuckets = append(c.snapshotBuckets, bucket)
	return nil
}

func (c *schedulerSnapshotCacheSpy) GetAccount(context.Context, int64) (*Account, error) {
	return nil, nil
}
func (c *schedulerSnapshotCacheSpy) SetAccount(context.Context, *Account) error { return nil }
func (c *schedulerSnapshotCacheSpy) DeleteAccount(context.Context, int64) error { return nil }
func (c *schedulerSnapshotCacheSpy) UpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}

func (c *schedulerSnapshotCacheSpy) TryLockBucket(context.Context, SchedulerBucket, time.Duration) (string, bool, error) {
	return "lock", true, nil
}

func (c *schedulerSnapshotCacheSpy) UnlockBucket(context.Context, SchedulerBucket, string) error {
	return nil
}

func (c *schedulerSnapshotCacheSpy) ListBuckets(context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}

func (c *schedulerSnapshotCacheSpy) GetOutboxWatermark(context.Context) (int64, error) {
	return 0, nil
}

func (c *schedulerSnapshotCacheSpy) SetOutboxWatermark(ctx context.Context, id int64) error {
	c.watermarkCalls++
	if deadline, ok := ctx.Deadline(); ok {
		c.watermarkDeadlines = append(c.watermarkDeadlines, deadline)
	}
	if c.watermarkSucceedOn > 0 && c.watermarkCalls < c.watermarkSucceedOn {
		return context.DeadlineExceeded
	}
	c.lastWatermark = id
	return nil
}

func TestSchedulerSnapshotService_PollOutboxDeduplicatesGroupPlatformRebuilds(t *testing.T) {
	accountID := int64(101)
	groupID := int64(7)
	account := Account{
		ID:          accountID,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{groupID},
	}
	repo := &mockAccountRepoForPlatform{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{accountID: &account},
	}
	cache := &schedulerSnapshotCacheSpy{}
	outbox := &schedulerOutboxRepoStub{
		events: []SchedulerOutboxEvent{
			{ID: 1, EventType: SchedulerOutboxEventAccountChanged, AccountID: &accountID, Payload: map[string]any{"group_ids": []any{groupID}}},
			{ID: 2, EventType: SchedulerOutboxEventAccountChanged, AccountID: &accountID, Payload: map[string]any{"group_ids": []any{groupID}}},
		},
	}

	svc := NewSchedulerSnapshotService(cache, outbox, repo, nil, nil)
	svc.pollOutbox()

	require.Len(t, cache.snapshotBuckets, 2)
	require.ElementsMatch(t, []SchedulerBucket{
		{GroupID: groupID, Platform: PlatformOpenAI, Mode: SchedulerModeSingle},
		{GroupID: groupID, Platform: PlatformOpenAI, Mode: SchedulerModeForced},
	}, cache.snapshotBuckets)
	require.Equal(t, int64(2), cache.lastWatermark)
}

func TestSchedulerSnapshotService_PollOutboxUsesFreshContextForWatermarkRetries(t *testing.T) {
	cache := &schedulerSnapshotCacheSpy{watermarkSucceedOn: 3}
	outbox := &schedulerOutboxRepoStub{
		events: []SchedulerOutboxEvent{
			{ID: 9, EventType: "noop", CreatedAt: time.Now()},
		},
	}

	svc := NewSchedulerSnapshotService(cache, outbox, nil, nil, nil)
	svc.pollOutbox()

	require.Equal(t, 3, cache.watermarkCalls)
	require.Equal(t, int64(9), cache.lastWatermark)
	require.Len(t, cache.watermarkDeadlines, 3)
	require.False(t, cache.watermarkDeadlines[0].Equal(cache.watermarkDeadlines[1]))
	require.False(t, cache.watermarkDeadlines[1].Equal(cache.watermarkDeadlines[2]))
}
