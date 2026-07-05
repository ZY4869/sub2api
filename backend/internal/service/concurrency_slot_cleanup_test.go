package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type slotCleanupCache struct {
	calls atomic.Int64
}

var _ ConcurrencyCache = (*slotCleanupCache)(nil)

func (c *slotCleanupCache) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}

func (c *slotCleanupCache) ReleaseAccountSlot(context.Context, int64, string) error {
	return nil
}

func (c *slotCleanupCache) GetAccountConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *slotCleanupCache) GetAccountConcurrencyBatch(_ context.Context, accountIDs []int64) (map[int64]int, error) {
	return make(map[int64]int, len(accountIDs)), nil
}

func (c *slotCleanupCache) IncrementAccountWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}

func (c *slotCleanupCache) DecrementAccountWaitCount(context.Context, int64) error {
	return nil
}

func (c *slotCleanupCache) GetAccountWaitingCount(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *slotCleanupCache) AcquireUserSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}

func (c *slotCleanupCache) ReleaseUserSlot(context.Context, int64, string) error {
	return nil
}

func (c *slotCleanupCache) GetUserConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *slotCleanupCache) IncrementWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}

func (c *slotCleanupCache) DecrementWaitCount(context.Context, int64) error {
	return nil
}

func (c *slotCleanupCache) GetAccountsLoadBatch(context.Context, []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	return map[int64]*AccountLoadInfo{}, nil
}

func (c *slotCleanupCache) GetUsersLoadBatch(context.Context, []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	return map[int64]*UserLoadInfo{}, nil
}

func (c *slotCleanupCache) CleanupExpiredAccountSlots(context.Context, int64) error {
	return nil
}

func (c *slotCleanupCache) CleanupExpiredAccountSlotKeys(context.Context) error {
	c.calls.Add(1)
	return nil
}

func (c *slotCleanupCache) CleanupStaleProcessSlots(context.Context, string) error {
	return nil
}

func TestStartSlotCleanupWorker_UsesCacheWideCleanupWithoutAccountRepo(t *testing.T) {
	cache := &slotCleanupCache{}
	svc := NewConcurrencyService(cache)

	svc.StartSlotCleanupWorker(nil, time.Hour)

	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if cache.calls.Load() > 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatal("cleanup worker did not call cache-wide account slot cleanup")
		case <-ticker.C:
		}
	}
}
