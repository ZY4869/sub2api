//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 覆盖回归："暂停使用"(Schedulable=false) 的账号仍被调度的三类根因：
// 1) 请求期旧对象整体回写污染调度缓存（sched:acc:<id> 无 TTL，污染后无法自愈）；
// 2) 并发槽等待返回后不复查账号状态；
// 3) bucket 重建抢锁失败被静默跳过导致快照长期陈旧。

type runtimeUpdateCacheSpy struct {
	schedulerSnapshotCacheSpy
	setAccounts []*Account
}

func (c *runtimeUpdateCacheSpy) SetAccount(_ context.Context, account *Account) error {
	c.setAccounts = append(c.setAccounts, account)
	return nil
}

func TestUpdateAccountRuntimeInCache_DoesNotResurrectPausedAccount(t *testing.T) {
	paused := Account{ID: 9, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: false}
	repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{9: &paused}}
	cache := &runtimeUpdateCacheSpy{}
	svc := NewSchedulerSnapshotService(cache, nil, repo, nil, nil)

	// 模拟：请求期持有的旧对象仍认为账号可调度，期间管理员已在 DB 中暂停该账号。
	err := svc.UpdateAccountRuntimeInCache(context.Background(), 9, func(fresh *Account) {
		if fresh.Extra == nil {
			fresh.Extra = make(map[string]any)
		}
		fresh.Extra["model_rate_limits"] = map[string]any{
			"claude": map[string]any{"rate_limit_reset_at": time.Now().UTC().Format(time.RFC3339)},
		}
	})

	require.NoError(t, err)
	require.Len(t, cache.setAccounts, 1)
	written := cache.setAccounts[0]
	require.False(t, written.Schedulable, "写回缓存的账号必须保持 DB 中的暂停状态")
	require.Contains(t, written.Extra, "model_rate_limits", "运行时字段的变更应生效")
	require.Equal(t, 1, repo.getByIDCalls, "必须从 DB 重读而非直接使用调用方对象")
}

func TestUpdateAccountRuntimeInCache_RepoErrorSkipsCacheWrite(t *testing.T) {
	repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{}}
	cache := &runtimeUpdateCacheSpy{}
	svc := NewSchedulerSnapshotService(cache, nil, repo, nil, nil)

	err := svc.UpdateAccountRuntimeInCache(context.Background(), 9, nil)

	require.Error(t, err)
	require.Empty(t, cache.setAccounts, "重读失败时不得写缓存")
}

type lockRefusingCacheSpy struct {
	schedulerSnapshotCacheSpy
}

func (c *lockRefusingCacheSpy) TryLockBucket(context.Context, SchedulerBucket, time.Duration) (string, bool, error) {
	return "", false, nil
}

func TestRebuildBucket_LockContentionReturnsRetryableError(t *testing.T) {
	cache := &lockRefusingCacheSpy{}
	svc := NewSchedulerSnapshotService(cache, nil, &mockAccountRepoForPlatform{}, nil, nil)

	err := svc.rebuildBucket(context.Background(), SchedulerBucket{GroupID: 1, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}, "test")

	require.ErrorIs(t, err, errSchedulerBucketRebuildLocked)
}

func TestPollOutbox_LockContentionDoesNotAdvanceWatermark(t *testing.T) {
	accountID := int64(7)
	account := Account{ID: accountID, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: true, GroupIDs: []int64{1}}
	repo := &mockAccountRepoForPlatform{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{accountID: &account},
	}
	cache := &lockRefusingCacheSpy{}
	outbox := &schedulerOutboxRepoStub{
		events: []SchedulerOutboxEvent{
			{ID: 1, EventType: SchedulerOutboxEventAccountChanged, AccountID: &accountID, Payload: map[string]any{"group_ids": []any{int64(1)}}},
		},
	}
	svc := NewSchedulerSnapshotService(cache, outbox, repo, nil, nil)

	svc.pollOutbox()

	require.Zero(t, cache.watermarkCalls, "重建被锁挡下时不得推进 watermark，须留待下轮重试")
	require.Empty(t, outbox.deleteThrough, "事件未处理完成时不得清理 outbox")
}

func TestRevalidateSelectionAccount(t *testing.T) {
	t.Run("账号已暂停返回nil触发重新选号", func(t *testing.T) {
		paused := Account{ID: 3, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: false}
		repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{3: &paused}}
		svc := &GatewayService{accountRepo: repo}

		got := svc.RevalidateSelectionAccount(context.Background(), &Account{ID: 3, Schedulable: true})

		require.Nil(t, got, "等槽期间被暂停的账号必须触发重新选号")
	})

	t.Run("账号仍可调度返回最新对象", func(t *testing.T) {
		active := Account{ID: 4, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: true}
		repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{4: &active}}
		svc := &GatewayService{accountRepo: repo}

		got := svc.RevalidateSelectionAccount(context.Background(), &Account{ID: 4, Schedulable: true})

		require.Same(t, &active, got)
	})

	t.Run("读取失败时放行原账号", func(t *testing.T) {
		repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{}}
		svc := &GatewayService{accountRepo: repo}
		original := &Account{ID: 5, Schedulable: true}

		got := svc.RevalidateSelectionAccount(context.Background(), original)

		require.Same(t, original, got, "基础设施读取失败不应放大为选号失败")
	})
}

func TestResolveFreshSelectionAccount_PausedReturnsNil(t *testing.T) {
	svc := &GatewayService{}
	paused := &Account{ID: 6, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: false}

	require.Nil(t, svc.resolveFreshSelectionAccount(context.Background(), paused, PlatformAnthropic, false, "", false))
}

func TestSelectAccountForModelWithPlatform_StickyPausedFallsThrough(t *testing.T) {
	ctx := context.Background()
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{ID: 1, Platform: PlatformAnthropic, Priority: 2, Status: StatusActive, Schedulable: false},
			{ID: 2, Platform: PlatformAnthropic, Priority: 1, Status: StatusActive, Schedulable: true},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}
	cache := &mockGatewayCacheForPlatform{sessionBindings: map[string]int64{"session-paused": 1}}
	svc := &GatewayService{accountRepo: repo, cache: cache, cfg: testConfig()}

	acc, err := svc.selectAccountForModelWithPlatform(ctx, nil, "session-paused", "claude-3-5-sonnet-20241022", nil, PlatformAnthropic)

	require.NoError(t, err)
	require.NotNil(t, acc)
	require.Equal(t, int64(2), acc.ID, "暂停账号不得因粘性会话绑定被继续选中")
}
