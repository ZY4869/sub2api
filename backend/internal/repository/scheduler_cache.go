package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	schedulerBucketSetKey       = "sched:buckets"
	schedulerOutboxWatermarkKey = "sched:outbox:watermark"
	schedulerAccountPrefix      = "sched:acc:"
	schedulerActivePrefix       = "sched:active:"
	schedulerReadyPrefix        = "sched:ready:"
	schedulerVersionPrefix      = "sched:ver:"
	schedulerSnapshotPrefix     = "sched:"
	schedulerLockPrefix         = "sched:lock:"
)

const (
	// When switching active snapshots, keep the old snapshot key for a short grace period
	// so in-flight readers don't observe a transient miss.
	schedulerSnapshotGraceTTL = 5 * time.Minute
	// If a concurrent writer loses the CAS race, expire its snapshot quickly.
	schedulerSnapshotOrphanTTL = 30 * time.Second
)

var schedulerUnlockLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

var schedulerAllocateSnapshotVersionScript = redis.NewScript(`
local ver = redis.call("GET", KEYS[1])
local active = redis.call("GET", KEYS[2])
local v = tonumber(ver) or 0
local a = tonumber(active) or 0
local next = math.max(v, a) + 1
redis.call("SET", KEYS[1], tostring(next))
return next
`)

var schedulerActivateSnapshotScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
local next = ARGV[1]
local bucket = ARGV[2]
local prev = current or ""
local swapped = 0
local currentNum = tonumber(current) or 0
local nextNum = tonumber(next) or 0
if nextNum > currentNum then
  swapped = 1
  redis.call("SET", KEYS[1], next)
  redis.call("SET", KEYS[2], "1")
  redis.call("SADD", KEYS[3], bucket)
end
return {swapped, prev}
`)

type schedulerCache struct {
	rdb *redis.Client
}

func NewSchedulerCache(rdb *redis.Client) service.SchedulerCache {
	return &schedulerCache{rdb: rdb}
}

func (c *schedulerCache) GetSnapshot(ctx context.Context, bucket service.SchedulerBucket) ([]*service.Account, bool, error) {
	readyKey := schedulerBucketKey(schedulerReadyPrefix, bucket)
	readyVal, err := c.rdb.Get(ctx, readyKey).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if readyVal != "1" {
		return nil, false, nil
	}

	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	activeVal, err := c.rdb.Get(ctx, activeKey).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	snapshotKey := schedulerSnapshotKey(bucket, activeVal)
	ids, err := c.rdb.ZRange(ctx, snapshotKey, 0, -1).Result()
	if err != nil {
		return nil, false, err
	}
	if len(ids) == 0 {
		// 空快照视为缓存未命中，触发数据库回退查询
		// 这解决了新分组创建后立即绑定账号时的竞态条件问题
		return nil, false, nil
	}

	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, schedulerAccountKey(id))
	}
	values, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, false, err
	}

	accounts := make([]*service.Account, 0, len(values))
	for _, val := range values {
		if val == nil {
			return nil, false, nil
		}
		account, err := decodeCachedAccount(val)
		if err != nil {
			return nil, false, err
		}
		accounts = append(accounts, account)
	}

	return accounts, true, nil
}

func (c *schedulerCache) SetSnapshot(ctx context.Context, bucket service.SchedulerBucket, accounts []service.Account) error {
	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)

	versionKey := schedulerBucketKey(schedulerVersionPrefix, bucket)
	version, err := schedulerAllocateSnapshotVersionScript.Run(ctx, c.rdb, []string{versionKey, activeKey}).Int64()
	if err != nil {
		return err
	}

	versionStr := strconv.FormatInt(version, 10)
	snapshotKey := schedulerSnapshotKey(bucket, versionStr)

	pipe := c.rdb.Pipeline()
	for _, account := range accounts {
		payload, err := json.Marshal(account)
		if err != nil {
			return err
		}
		pipe.Set(ctx, schedulerAccountKey(strconv.FormatInt(account.ID, 10)), payload, 0)
	}
	if len(accounts) > 0 {
		// 使用序号作为 score，保持数据库返回的排序语义。
		members := make([]redis.Z, 0, len(accounts))
		for idx, account := range accounts {
			members = append(members, redis.Z{
				Score:  float64(idx),
				Member: strconv.FormatInt(account.ID, 10),
			})
		}
		pipe.ZAdd(ctx, snapshotKey, members...)
	} else {
		pipe.Del(ctx, snapshotKey)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	swapped, previous, err := c.activateSnapshotIfNewer(ctx, bucket, versionStr)
	if err != nil {
		return err
	}
	if swapped && previous != "" && previous != versionStr {
		_ = c.rdb.Expire(ctx, schedulerSnapshotKey(bucket, previous), schedulerSnapshotGraceTTL).Err()
	}
	if !swapped {
		_ = c.rdb.Expire(ctx, snapshotKey, schedulerSnapshotOrphanTTL).Err()
	}

	return nil
}

func (c *schedulerCache) GetAccount(ctx context.Context, accountID int64) (*service.Account, error) {
	key := schedulerAccountKey(strconv.FormatInt(accountID, 10))
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeCachedAccount(val)
}

func (c *schedulerCache) SetAccount(ctx context.Context, account *service.Account) error {
	if account == nil || account.ID <= 0 {
		return nil
	}
	payload, err := json.Marshal(account)
	if err != nil {
		return err
	}
	key := schedulerAccountKey(strconv.FormatInt(account.ID, 10))
	return c.rdb.Set(ctx, key, payload, 0).Err()
}

func (c *schedulerCache) DeleteAccount(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return nil
	}
	key := schedulerAccountKey(strconv.FormatInt(accountID, 10))
	return c.rdb.Del(ctx, key).Err()
}

func (c *schedulerCache) UpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	if len(updates) == 0 {
		return nil
	}

	keys := make([]string, 0, len(updates))
	ids := make([]int64, 0, len(updates))
	for id := range updates {
		keys = append(keys, schedulerAccountKey(strconv.FormatInt(id, 10)))
		ids = append(ids, id)
	}

	values, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	pipe := c.rdb.Pipeline()
	for i, val := range values {
		if val == nil {
			continue
		}
		account, err := decodeCachedAccount(val)
		if err != nil {
			return err
		}
		account.LastUsedAt = ptrTime(updates[ids[i]])
		updated, err := json.Marshal(account)
		if err != nil {
			return err
		}
		pipe.Set(ctx, keys[i], updated, 0)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (c *schedulerCache) TryLockBucket(ctx context.Context, bucket service.SchedulerBucket, ttl time.Duration) (string, bool, error) {
	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	token := uuid.NewString()
	locked, err := c.rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil || !locked {
		return "", locked, err
	}
	return token, true, nil
}

func (c *schedulerCache) UnlockBucket(ctx context.Context, bucket service.SchedulerBucket, lockToken string) error {
	lockToken = strings.TrimSpace(lockToken)
	if lockToken == "" {
		return nil
	}
	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	_, err := schedulerUnlockLockScript.Run(ctx, c.rdb, []string{key}, lockToken).Result()
	return err
}

func (c *schedulerCache) ListBuckets(ctx context.Context) ([]service.SchedulerBucket, error) {
	raw, err := c.rdb.SMembers(ctx, schedulerBucketSetKey).Result()
	if err != nil {
		return nil, err
	}
	out := make([]service.SchedulerBucket, 0, len(raw))
	for _, entry := range raw {
		bucket, ok := service.ParseSchedulerBucket(entry)
		if !ok {
			continue
		}
		out = append(out, bucket)
	}
	return out, nil
}

func (c *schedulerCache) GetOutboxWatermark(ctx context.Context) (int64, error) {
	val, err := c.rdb.Get(ctx, schedulerOutboxWatermarkKey).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (c *schedulerCache) SetOutboxWatermark(ctx context.Context, id int64) error {
	return c.rdb.Set(ctx, schedulerOutboxWatermarkKey, strconv.FormatInt(id, 10), 0).Err()
}

func schedulerBucketKey(prefix string, bucket service.SchedulerBucket) string {
	return fmt.Sprintf("%s%d:%s:%s", prefix, bucket.GroupID, bucket.Platform, bucket.Mode)
}

func schedulerSnapshotKey(bucket service.SchedulerBucket, version string) string {
	return fmt.Sprintf("%s%d:%s:%s:v%s", schedulerSnapshotPrefix, bucket.GroupID, bucket.Platform, bucket.Mode, version)
}

func schedulerAccountKey(id string) string {
	return schedulerAccountPrefix + id
}

func (c *schedulerCache) activateSnapshotIfNewer(ctx context.Context, bucket service.SchedulerBucket, next string) (bool, string, error) {
	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	readyKey := schedulerBucketKey(schedulerReadyPrefix, bucket)
	vals, err := schedulerActivateSnapshotScript.Run(ctx, c.rdb, []string{activeKey, readyKey, schedulerBucketSetKey}, next, bucket.String()).Slice()
	if err != nil {
		return false, "", err
	}
	if len(vals) < 2 {
		return false, "", nil
	}
	swapped := false
	switch v := vals[0].(type) {
	case int64:
		swapped = v == 1
	case int:
		swapped = v == 1
	case string:
		swapped = strings.TrimSpace(v) == "1"
	}

	previous := ""
	switch v := vals[1].(type) {
	case string:
		previous = strings.TrimSpace(v)
	case []byte:
		previous = strings.TrimSpace(string(v))
	}
	return swapped, previous, nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func decodeCachedAccount(val any) (*service.Account, error) {
	var payload []byte
	switch raw := val.(type) {
	case string:
		payload = []byte(raw)
	case []byte:
		payload = raw
	default:
		return nil, fmt.Errorf("unexpected account cache type: %T", val)
	}
	var account service.Account
	if err := json.Unmarshal(payload, &account); err != nil {
		return nil, err
	}
	return &account, nil
}
