package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// 并发控制缓存常量定义
//
// 性能优化说明：
// 原实现使用 SCAN 命令遍历独立的槽位键（concurrency:account:{id}:{requestID}），
// 在高并发场景下 SCAN 需要多次往返，且遍历大量键时性能下降明显。
//
// 新实现改用 Redis 有序集合（Sorted Set）：
// 1. 每个账号/用户只有一个键，成员为 requestID，分数为时间戳
// 2. 使用 ZCARD 原子获取并发数，时间复杂度 O(1)
// 3. 使用 ZREMRANGEBYSCORE 清理过期槽位，避免手动管理 TTL
// 4. 单次 Redis 调用完成计数，减少网络往返
const (
	// 并发槽位键前缀（有序集合）
	// 格式: concurrency:account:{accountID}
	accountSlotKeyPrefix = "concurrency:account:"
	accountActiveSetKey  = "concurrency:account:active"
	// 格式: concurrency:user:{userID}
	userSlotKeyPrefix = "concurrency:user:"
	userActiveSetKey  = "concurrency:user:active"
	// 格式: concurrency:api_key:{apiKeyID}
	apiKeySlotKeyPrefix = "concurrency:api_key:"
	apiKeyActiveSetKey  = "concurrency:api_key:active"
	// 等待队列计数器格式: concurrency:wait:{userID}
	waitQueueKeyPrefix   = "concurrency:wait:"
	userWaitActiveSetKey = "concurrency:wait:active"
	// 账号级等待队列计数器格式: wait:account:{accountID}
	accountWaitKeyPrefix    = "wait:account:"
	accountWaitActiveSetKey = "wait:account:active"

	// 默认槽位过期时间（分钟），可通过配置覆盖
	defaultSlotTTLMinutes    = 15
	legacyWaitSweepMarkerKey = "concurrency:startup:legacy_wait_sweep:v1"
)

var (
	// acquireScript 使用有序集合计数并在未达上限时添加槽位
	// 使用 Redis TIME 命令获取服务器时间，避免多实例时钟不同步问题
	// KEYS[1] = 有序集合键 (concurrency:account:{id} / concurrency:user:{id})
	// ARGV[1] = maxConcurrency
	// ARGV[2] = TTL（秒）
	// ARGV[3] = requestID
	acquireScript = redis.NewScript(`
		local key = KEYS[1]
		local activeSet = KEYS[2]
		local maxConcurrency = tonumber(ARGV[1])
		local ttl = tonumber(ARGV[2])
		local requestID = ARGV[3]
		local accountID = ARGV[4]

		-- 使用 Redis 服务器时间，确保多实例时钟一致
		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - ttl

		-- 清理过期槽位
		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)

		-- 检查是否已存在（支持重试场景刷新时间戳）
		local exists = redis.call('ZSCORE', key, requestID)
		if exists ~= false then
			redis.call('ZADD', key, now, requestID)
			redis.call('EXPIRE', key, ttl)
			if activeSet ~= false and activeSet ~= nil and accountID ~= nil and accountID ~= '' then
				redis.call('SADD', activeSet, accountID)
			end
			return 1
		end

		-- 检查是否达到并发上限
		local count = redis.call('ZCARD', key)
		if count < maxConcurrency then
			redis.call('ZADD', key, now, requestID)
			redis.call('EXPIRE', key, ttl)
			if activeSet ~= false and activeSet ~= nil and accountID ~= nil and accountID ~= '' then
				redis.call('SADD', activeSet, accountID)
			end
			return 1
		end

		return 0
	`)

	// getCountScript 统计有序集合中的槽位数量并清理过期条目
	// 使用 Redis TIME 命令获取服务器时间
	// KEYS[1] = 有序集合键
	// ARGV[1] = TTL（秒）
	getCountScript = redis.NewScript(`
		local key = KEYS[1]
		local activeSet = KEYS[2]
		local ttl = tonumber(ARGV[1])
		local accountID = ARGV[2]

		-- 使用 Redis 服务器时间
		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - ttl

		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)
		local count = redis.call('ZCARD', key)
		if count == 0 then
			redis.call('DEL', key)
			if activeSet ~= false and activeSet ~= nil and accountID ~= nil and accountID ~= '' then
				redis.call('SREM', activeSet, accountID)
			end
		else
			redis.call('EXPIRE', key, ttl)
			if activeSet ~= false and activeSet ~= nil and accountID ~= nil and accountID ~= '' then
				redis.call('SADD', activeSet, accountID)
			end
		end
		return count
	`)

	// incrementWaitScript - refreshes TTL on each increment to keep queue depth accurate
	// KEYS[1] = wait queue key
	// ARGV[1] = maxWait
	// ARGV[2] = TTL in seconds
	incrementWaitScript = redis.NewScript(`
		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local current = redis.call('GET', KEYS[1])
		if current == false then
			current = 0
		else
			current = tonumber(current)
		end

		if current >= tonumber(ARGV[1]) then
			return {0, now}
		end

		local newVal = redis.call('INCR', KEYS[1])

		-- Refresh TTL so long-running traffic doesn't expire active queue counters.
		redis.call('EXPIRE', KEYS[1], ARGV[2])

			return {1, now}
		`)

	// incrementAccountWaitScript - account-level wait queue count (refresh TTL on each increment)
	incrementAccountWaitScript = redis.NewScript(`
			local timeResult = redis.call('TIME')
			local now = tonumber(timeResult[1])
			local current = redis.call('GET', KEYS[1])
			if current == false then
				current = 0
			else
				current = tonumber(current)
			end

			if current >= tonumber(ARGV[1]) then
				return {0, now}
			end

			local newVal = redis.call('INCR', KEYS[1])

			-- Refresh TTL so long-running traffic doesn't expire active queue counters.
			redis.call('EXPIRE', KEYS[1], ARGV[2])

			return {1, now}
		`)

	// decrementWaitScript - same as before
	decrementWaitScript = redis.NewScript(`
			local current = redis.call('GET', KEYS[1])
			if current ~= false and tonumber(current) > 0 then
				current = redis.call('DECR', KEYS[1])
				if tonumber(current) <= 0 then
					redis.call('DEL', KEYS[1])
					return 0
				end
				return tonumber(current)
			end
			return 0
		`)

	// trackSlotScript tracks a stats-only slot without enforcing a limit.
	// KEYS[1] = 有序集合键
	// KEYS[2] = active set key
	// ARGV[1] = TTL（秒）
	// ARGV[2] = requestID
	// ARGV[3] = id
	trackSlotScript = redis.NewScript(`
		local key = KEYS[1]
		local activeSet = KEYS[2]
		local ttl = tonumber(ARGV[1])
		local requestID = ARGV[2]
		local id = ARGV[3]

		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - ttl

		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)
		redis.call('ZADD', key, now, requestID)
		redis.call('EXPIRE', key, ttl)
		if activeSet ~= false and activeSet ~= nil and id ~= nil and id ~= '' then
			redis.call('SADD', activeSet, id)
		end
		return 1
	`)

	// cleanupExpiredSlotsScript 清理单个账号/用户有序集合中过期槽位
	// KEYS[1] = 有序集合键
	// ARGV[1] = TTL（秒）
	cleanupExpiredSlotsScript = redis.NewScript(`
		local key = KEYS[1]
		local activeSet = KEYS[2]
		local ttl = tonumber(ARGV[1])
		local accountID = ARGV[2]
		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - ttl
		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)
		if redis.call('ZCARD', key) == 0 then
			redis.call('DEL', key)
			if activeSet ~= false and activeSet ~= nil and accountID ~= nil and accountID ~= '' then
				redis.call('SREM', activeSet, accountID)
			end
		else
			redis.call('EXPIRE', key, ttl)
			if activeSet ~= false and activeSet ~= nil and accountID ~= nil and accountID ~= '' then
				redis.call('SADD', activeSet, accountID)
			end
		end
		return 1
	`)

	// startupCleanupScript 清理非当前进程前缀的槽位成员。
	// KEYS 是有序集合键列表，ARGV[1] 是当前进程前缀，ARGV[2] 是槽位 TTL。
	// 遍历每个 KEYS[i]，移除前缀不匹配的成员，清空后删 key，否则刷新 EXPIRE。
	startupCleanupScript = redis.NewScript(`
		local activePrefix = ARGV[1]
		local slotTTL = tonumber(ARGV[2])
		local removed = 0
		for i = 1, #KEYS do
			local key = KEYS[i]
			local members = redis.call('ZRANGE', key, 0, -1)
			for _, member in ipairs(members) do
				if string.sub(member, 1, string.len(activePrefix)) ~= activePrefix then
					removed = removed + redis.call('ZREM', key, member)
				end
			end
			if redis.call('ZCARD', key) == 0 then
				redis.call('DEL', key)
			else
				redis.call('EXPIRE', key, slotTTL)
			end
		end
		return removed
	`)
)

type concurrencyCache struct {
	rdb                 *redis.Client
	slotTTLSeconds      int // 槽位过期时间（秒）
	waitQueueTTLSeconds int // 等待队列过期时间（秒）
}

// NewConcurrencyCache 创建并发控制缓存
// slotTTLMinutes: 槽位过期时间（分钟），0 或负数使用默认值 15 分钟
// waitQueueTTLSeconds: 等待队列过期时间（秒），0 或负数使用 slot TTL
func NewConcurrencyCache(rdb *redis.Client, slotTTLMinutes int, waitQueueTTLSeconds int) service.ConcurrencyCache {
	if slotTTLMinutes <= 0 {
		slotTTLMinutes = defaultSlotTTLMinutes
	}
	if waitQueueTTLSeconds <= 0 {
		waitQueueTTLSeconds = slotTTLMinutes * 60
	}
	return &concurrencyCache{
		rdb:                 rdb,
		slotTTLSeconds:      slotTTLMinutes * 60,
		waitQueueTTLSeconds: waitQueueTTLSeconds,
	}
}

// Helper functions for key generation
func accountSlotKey(accountID int64) string {
	return fmt.Sprintf("%s%d", accountSlotKeyPrefix, accountID)
}

func userSlotKey(userID int64) string {
	return fmt.Sprintf("%s%d", userSlotKeyPrefix, userID)
}

func apiKeySlotKey(apiKeyID int64) string {
	return fmt.Sprintf("%s%d", apiKeySlotKeyPrefix, apiKeyID)
}

func waitQueueKey(userID int64) string {
	return fmt.Sprintf("%s%d", waitQueueKeyPrefix, userID)
}

func accountWaitKey(accountID int64) string {
	return fmt.Sprintf("%s%d", accountWaitKeyPrefix, accountID)
}

func parseTrackedAccountIDs(rawMembers []string) []int64 {
	if len(rawMembers) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(rawMembers))
	for _, raw := range rawMembers {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func (c *concurrencyCache) syncActiveSetState(ctx context.Context, activeSetKey string, keyPrefix string, counts map[int64]int) error {
	if len(counts) == 0 {
		return nil
	}
	pipe := c.rdb.Pipeline()
	for id, count := range counts {
		key := keyPrefix + strconv.FormatInt(id, 10)
		member := strconv.FormatInt(id, 10)
		if count > 0 {
			pipe.SAdd(ctx, activeSetKey, member)
			pipe.Expire(ctx, key, time.Duration(c.slotTTLSeconds)*time.Second)
			continue
		}
		pipe.SRem(ctx, activeSetKey, member)
		pipe.Del(ctx, key)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return nil
}

func (c *concurrencyCache) syncAccountActiveState(ctx context.Context, counts map[int64]int) error {
	return c.syncActiveSetState(ctx, accountActiveSetKey, accountSlotKeyPrefix, counts)
}

func (c *concurrencyCache) syncUserActiveState(ctx context.Context, counts map[int64]int) error {
	return c.syncActiveSetState(ctx, userActiveSetKey, userSlotKeyPrefix, counts)
}

func (c *concurrencyCache) syncAPIKeyActiveState(ctx context.Context, counts map[int64]int) error {
	return c.syncActiveSetState(ctx, apiKeyActiveSetKey, apiKeySlotKeyPrefix, counts)
}

func (c *concurrencyCache) syncWaitActiveState(ctx context.Context, activeSetKey string, keyPrefix string, id int64, count int) error {
	member := strconv.FormatInt(id, 10)
	key := keyPrefix + member
	if count > 0 {
		if err := c.rdb.SAdd(ctx, activeSetKey, member).Err(); err != nil {
			return err
		}
		return c.rdb.Expire(ctx, key, time.Duration(c.waitQueueTTLSeconds)*time.Second).Err()
	}
	pipe := c.rdb.Pipeline()
	pipe.SRem(ctx, activeSetKey, member)
	pipe.Del(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return nil
}

func (c *concurrencyCache) GetTrackedActiveAccountIDs(ctx context.Context) ([]int64, error) {
	members, err := c.rdb.SMembers(ctx, accountActiveSetKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	return parseTrackedAccountIDs(members), nil
}

// Account slot operations

func (c *concurrencyCache) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error) {
	key := accountSlotKey(accountID)
	// 时间戳在 Lua 脚本内使用 Redis TIME 命令获取，确保多实例时钟一致
	result, err := acquireScript.Run(ctx, c.rdb, []string{key, accountActiveSetKey}, maxConcurrency, c.slotTTLSeconds, requestID, strconv.FormatInt(accountID, 10)).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *concurrencyCache) ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error {
	key := accountSlotKey(accountID)
	pipe := c.rdb.Pipeline()
	pipe.ZRem(ctx, key, requestID)
	countCmd := pipe.ZCard(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return c.syncAccountActiveState(ctx, map[int64]int{accountID: int(countCmd.Val())})
}

func (c *concurrencyCache) GetAccountConcurrency(ctx context.Context, accountID int64) (int, error) {
	key := accountSlotKey(accountID)
	// 时间戳在 Lua 脚本内使用 Redis TIME 命令获取
	result, err := getCountScript.Run(ctx, c.rdb, []string{key, accountActiveSetKey}, c.slotTTLSeconds, strconv.FormatInt(accountID, 10)).Int()
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (c *concurrencyCache) GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error) {
	if len(accountIDs) == 0 {
		return map[int64]int{}, nil
	}
	result, err := c.getSlotConcurrencyBatch(ctx, accountSlotKeyPrefix, accountIDs)
	if err != nil {
		return nil, err
	}
	if err := c.syncAccountActiveState(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *concurrencyCache) getSlotConcurrencyBatch(ctx context.Context, keyPrefix string, ids []int64) (map[int64]int, error) {
	if len(ids) == 0 {
		return map[int64]int{}, nil
	}

	now, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis TIME: %w", err)
	}
	cutoffTime := now.Unix() - int64(c.slotTTLSeconds)

	pipe := c.rdb.Pipeline()
	type slotCmd struct {
		id       int64
		zcardCmd *redis.IntCmd
	}
	cmds := make([]slotCmd, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		slotKey := keyPrefix + strconv.FormatInt(id, 10)
		pipe.ZRemRangeByScore(ctx, slotKey, "-inf", strconv.FormatInt(cutoffTime, 10))
		cmds = append(cmds, slotCmd{
			id:       id,
			zcardCmd: pipe.ZCard(ctx, slotKey),
		})
	}

	if len(cmds) == 0 {
		return map[int64]int{}, nil
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	result := make(map[int64]int, len(cmds))
	for _, cmd := range cmds {
		result[cmd.id] = int(cmd.zcardCmd.Val())
	}
	return result, nil
}

// User slot operations

func (c *concurrencyCache) AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error) {
	key := userSlotKey(userID)
	// 时间戳在 Lua 脚本内使用 Redis TIME 命令获取，确保多实例时钟一致
	result, err := acquireScript.Run(ctx, c.rdb, []string{key, userActiveSetKey}, maxConcurrency, c.slotTTLSeconds, requestID, strconv.FormatInt(userID, 10)).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *concurrencyCache) ReleaseUserSlot(ctx context.Context, userID int64, requestID string) error {
	key := userSlotKey(userID)
	pipe := c.rdb.Pipeline()
	pipe.ZRem(ctx, key, requestID)
	countCmd := pipe.ZCard(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return c.syncUserActiveState(ctx, map[int64]int{userID: int(countCmd.Val())})
}

func (c *concurrencyCache) GetUserConcurrency(ctx context.Context, userID int64) (int, error) {
	key := userSlotKey(userID)
	// 时间戳在 Lua 脚本内使用 Redis TIME 命令获取
	result, err := getCountScript.Run(ctx, c.rdb, []string{key, userActiveSetKey}, c.slotTTLSeconds, strconv.FormatInt(userID, 10)).Int()
	if err != nil {
		return 0, err
	}
	return result, nil
}

// API key slot operations are stats-only and do not enforce a concurrency limit.

func (c *concurrencyCache) TrackAPIKeySlot(ctx context.Context, apiKeyID int64, requestID string) error {
	_, err := trackSlotScript.Run(ctx, c.rdb, []string{apiKeySlotKey(apiKeyID), apiKeyActiveSetKey}, c.slotTTLSeconds, requestID, strconv.FormatInt(apiKeyID, 10)).Result()
	return err
}

func (c *concurrencyCache) ReleaseAPIKeySlot(ctx context.Context, apiKeyID int64, requestID string) error {
	key := apiKeySlotKey(apiKeyID)
	pipe := c.rdb.Pipeline()
	pipe.ZRem(ctx, key, requestID)
	countCmd := pipe.ZCard(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return c.syncAPIKeyActiveState(ctx, map[int64]int{apiKeyID: int(countCmd.Val())})
}

func (c *concurrencyCache) GetAPIKeyConcurrencyBatch(ctx context.Context, apiKeyIDs []int64) (map[int64]int, error) {
	result, err := c.getSlotConcurrencyBatch(ctx, apiKeySlotKeyPrefix, apiKeyIDs)
	if err != nil {
		return nil, err
	}
	if err := c.syncAPIKeyActiveState(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Wait queue operations

func (c *concurrencyCache) IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error) {
	key := waitQueueKey(userID)
	result, err := incrementWaitScript.Run(ctx, c.rdb, []string{key}, maxWait, c.waitQueueTTLSeconds).Result()
	if err != nil {
		return false, err
	}
	allowed, err := redisScriptInt64At(result, 0)
	if err != nil {
		return false, err
	}
	if allowed == 1 {
		if err := c.syncWaitActiveState(ctx, userWaitActiveSetKey, waitQueueKeyPrefix, userID, 1); err != nil {
			return false, err
		}
	}
	return allowed == 1, nil
}

func (c *concurrencyCache) DecrementWaitCount(ctx context.Context, userID int64) error {
	key := waitQueueKey(userID)
	count, err := decrementWaitScript.Run(ctx, c.rdb, []string{key}).Int()
	if err != nil {
		return err
	}
	return c.syncWaitActiveState(ctx, userWaitActiveSetKey, waitQueueKeyPrefix, userID, count)
}

// Account wait queue operations

func (c *concurrencyCache) IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error) {
	key := accountWaitKey(accountID)
	result, err := incrementAccountWaitScript.Run(ctx, c.rdb, []string{key}, maxWait, c.waitQueueTTLSeconds).Result()
	if err != nil {
		return false, err
	}
	allowed, err := redisScriptInt64At(result, 0)
	if err != nil {
		return false, err
	}
	if allowed == 1 {
		if err := c.syncWaitActiveState(ctx, accountWaitActiveSetKey, accountWaitKeyPrefix, accountID, 1); err != nil {
			return false, err
		}
	}
	return allowed == 1, nil
}

func (c *concurrencyCache) DecrementAccountWaitCount(ctx context.Context, accountID int64) error {
	key := accountWaitKey(accountID)
	count, err := decrementWaitScript.Run(ctx, c.rdb, []string{key}).Int()
	if err != nil {
		return err
	}
	return c.syncWaitActiveState(ctx, accountWaitActiveSetKey, accountWaitKeyPrefix, accountID, count)
}

func (c *concurrencyCache) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	key := accountWaitKey(accountID)
	val, err := c.rdb.Get(ctx, key).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	}
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, nil
}

func (c *concurrencyCache) GetAccountsLoadBatch(ctx context.Context, accounts []service.AccountWithConcurrency) (map[int64]*service.AccountLoadInfo, error) {
	if len(accounts) == 0 {
		return map[int64]*service.AccountLoadInfo{}, nil
	}

	// 使用 Pipeline 替代 Lua 脚本，兼容 Redis Cluster（Lua 内动态拼 key 会 CROSSSLOT）。
	// 每个账号执行 3 个命令：ZREMRANGEBYSCORE（清理过期）、ZCARD（并发数）、GET（等待数）。
	now, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis TIME: %w", err)
	}
	cutoffTime := now.Unix() - int64(c.slotTTLSeconds)

	pipe := c.rdb.Pipeline()

	type accountCmds struct {
		id             int64
		maxConcurrency int
		zcardCmd       *redis.IntCmd
		getCmd         *redis.StringCmd
	}
	cmds := make([]accountCmds, 0, len(accounts))
	for _, acc := range accounts {
		slotKey := accountSlotKeyPrefix + strconv.FormatInt(acc.ID, 10)
		waitKey := accountWaitKeyPrefix + strconv.FormatInt(acc.ID, 10)
		pipe.ZRemRangeByScore(ctx, slotKey, "-inf", strconv.FormatInt(cutoffTime, 10))
		ac := accountCmds{
			id:             acc.ID,
			maxConcurrency: acc.MaxConcurrency,
			zcardCmd:       pipe.ZCard(ctx, slotKey),
			getCmd:         pipe.Get(ctx, waitKey),
		}
		cmds = append(cmds, ac)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	loadMap := make(map[int64]*service.AccountLoadInfo, len(accounts))
	for _, ac := range cmds {
		currentConcurrency := int(ac.zcardCmd.Val())
		waitingCount := 0
		if v, err := ac.getCmd.Int(); err == nil {
			waitingCount = v
		}
		loadRate := 0
		if ac.maxConcurrency > 0 {
			loadRate = (currentConcurrency + waitingCount) * 100 / ac.maxConcurrency
		}
		loadMap[ac.id] = &service.AccountLoadInfo{
			AccountID:          ac.id,
			CurrentConcurrency: currentConcurrency,
			WaitingCount:       waitingCount,
			LoadRate:           loadRate,
		}
	}

	return loadMap, nil
}

func (c *concurrencyCache) GetUsersLoadBatch(ctx context.Context, users []service.UserWithConcurrency) (map[int64]*service.UserLoadInfo, error) {
	if len(users) == 0 {
		return map[int64]*service.UserLoadInfo{}, nil
	}

	// 使用 Pipeline 替代 Lua 脚本，兼容 Redis Cluster。
	now, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis TIME: %w", err)
	}
	cutoffTime := now.Unix() - int64(c.slotTTLSeconds)

	pipe := c.rdb.Pipeline()

	type userCmds struct {
		id             int64
		maxConcurrency int
		zcardCmd       *redis.IntCmd
		getCmd         *redis.StringCmd
	}
	cmds := make([]userCmds, 0, len(users))
	for _, u := range users {
		slotKey := userSlotKeyPrefix + strconv.FormatInt(u.ID, 10)
		waitKey := waitQueueKeyPrefix + strconv.FormatInt(u.ID, 10)
		pipe.ZRemRangeByScore(ctx, slotKey, "-inf", strconv.FormatInt(cutoffTime, 10))
		uc := userCmds{
			id:             u.ID,
			maxConcurrency: u.MaxConcurrency,
			zcardCmd:       pipe.ZCard(ctx, slotKey),
			getCmd:         pipe.Get(ctx, waitKey),
		}
		cmds = append(cmds, uc)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	loadMap := make(map[int64]*service.UserLoadInfo, len(users))
	for _, uc := range cmds {
		currentConcurrency := int(uc.zcardCmd.Val())
		waitingCount := 0
		if v, err := uc.getCmd.Int(); err == nil {
			waitingCount = v
		}
		loadRate := 0
		if uc.maxConcurrency > 0 {
			loadRate = (currentConcurrency + waitingCount) * 100 / uc.maxConcurrency
		}
		loadMap[uc.id] = &service.UserLoadInfo{
			UserID:             uc.id,
			CurrentConcurrency: currentConcurrency,
			WaitingCount:       waitingCount,
			LoadRate:           loadRate,
		}
	}

	return loadMap, nil
}

func (c *concurrencyCache) CleanupExpiredAccountSlots(ctx context.Context, accountID int64) error {
	key := accountSlotKey(accountID)
	_, err := cleanupExpiredSlotsScript.Run(ctx, c.rdb, []string{key, accountActiveSetKey}, c.slotTTLSeconds, strconv.FormatInt(accountID, 10)).Result()
	return err
}

func (c *concurrencyCache) CleanupExpiredAccountSlotKeys(ctx context.Context) error {
	if err := c.cleanupTrackedSlotKeys(ctx, accountActiveSetKey, accountSlotKeyPrefix); err != nil {
		return err
	}
	if err := c.cleanupTrackedSlotKeys(ctx, userActiveSetKey, userSlotKeyPrefix); err != nil {
		return err
	}
	return c.cleanupTrackedSlotKeys(ctx, apiKeyActiveSetKey, apiKeySlotKeyPrefix)
}

func (c *concurrencyCache) CleanupStaleProcessSlots(ctx context.Context, activeRequestPrefix string) error {
	if activeRequestPrefix == "" {
		return nil
	}

	if err := c.cleanupStaleProcessSlotsForActiveSet(ctx, accountActiveSetKey, accountSlotKeyPrefix, activeRequestPrefix); err != nil {
		return err
	}
	if err := c.cleanupStaleProcessSlotsForActiveSet(ctx, userActiveSetKey, userSlotKeyPrefix, activeRequestPrefix); err != nil {
		return err
	}
	if err := c.cleanupStaleProcessSlotsForActiveSet(ctx, apiKeyActiveSetKey, apiKeySlotKeyPrefix, activeRequestPrefix); err != nil {
		return err
	}

	if err := c.deleteWaitKeysForActiveSet(ctx, accountWaitActiveSetKey, accountWaitKeyPrefix); err != nil {
		return err
	}
	if err := c.deleteWaitKeysForActiveSet(ctx, userWaitActiveSetKey, waitQueueKeyPrefix); err != nil {
		return err
	}

	return c.sweepLegacyWaitKeysOnce(ctx)
}

func (c *concurrencyCache) cleanupTrackedSlotKeys(ctx context.Context, activeSetKey string, keyPrefix string) error {
	members, err := c.rdb.SMembers(ctx, activeSetKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("read active set %s: %w", activeSetKey, err)
	}
	for _, member := range members {
		id, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			if remErr := c.rdb.SRem(ctx, activeSetKey, member).Err(); remErr != nil {
				return fmt.Errorf("remove invalid active member %s from %s: %w", member, activeSetKey, remErr)
			}
			continue
		}
		key := keyPrefix + strconv.FormatInt(id, 10)
		if _, err := cleanupExpiredSlotsScript.Run(ctx, c.rdb, []string{key, activeSetKey}, c.slotTTLSeconds, strconv.FormatInt(id, 10)).Result(); err != nil {
			return fmt.Errorf("cleanup expired slots %s: %w", key, err)
		}
	}
	return nil
}

func (c *concurrencyCache) cleanupStaleProcessSlotsForActiveSet(ctx context.Context, activeSetKey string, keyPrefix string, activePrefix string) error {
	members, err := c.rdb.SMembers(ctx, activeSetKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("read active set %s: %w", activeSetKey, err)
	}
	for _, member := range members {
		id, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			if remErr := c.rdb.SRem(ctx, activeSetKey, member).Err(); remErr != nil {
				return fmt.Errorf("remove invalid active member %s from %s: %w", member, activeSetKey, remErr)
			}
			continue
		}
		key := keyPrefix + strconv.FormatInt(id, 10)
		if _, err := startupCleanupScript.Run(ctx, c.rdb, []string{key}, activePrefix, c.slotTTLSeconds).Result(); err != nil {
			return fmt.Errorf("cleanup slots %s: %w", key, err)
		}
		count, err := c.rdb.ZCard(ctx, key).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return fmt.Errorf("count slots %s: %w", key, err)
		}
		if count == 0 {
			pipe := c.rdb.Pipeline()
			pipe.SRem(ctx, activeSetKey, member)
			pipe.Del(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
				return fmt.Errorf("remove inactive slot %s: %w", key, err)
			}
		}
	}
	return nil
}

func (c *concurrencyCache) deleteWaitKeysForActiveSet(ctx context.Context, activeSetKey string, keyPrefix string) error {
	members, err := c.rdb.SMembers(ctx, activeSetKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("read wait active set %s: %w", activeSetKey, err)
	}
	if len(members) == 0 {
		return nil
	}
	pipe := c.rdb.Pipeline()
	for _, member := range members {
		if _, err := strconv.ParseInt(member, 10, 64); err != nil {
			continue
		}
		pipe.Del(ctx, keyPrefix+member)
	}
	pipe.Del(ctx, activeSetKey)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("delete indexed wait keys %s: %w", activeSetKey, err)
	}
	return nil
}

func (c *concurrencyCache) sweepLegacyWaitKeysOnce(ctx context.Context) error {
	exists, err := c.rdb.Exists(ctx, legacyWaitSweepMarkerKey).Result()
	if err != nil {
		return fmt.Errorf("check legacy wait sweep marker: %w", err)
	}
	if exists > 0 {
		return nil
	}
	for _, pattern := range []string{accountWaitKeyPrefix + "*", waitQueueKeyPrefix + "*"} {
		if err := c.deleteKeysByPattern(ctx, pattern); err != nil {
			return err
		}
	}
	if err := c.rdb.Set(ctx, legacyWaitSweepMarkerKey, "1", 0).Err(); err != nil {
		return fmt.Errorf("set legacy wait sweep marker: %w", err)
	}
	return nil
}

// deleteKeysByPattern is retained only for one-time legacy wait counter migration.
func (c *concurrencyCache) deleteKeysByPattern(ctx context.Context, pattern string) error {
	const scanCount = 200
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, pattern, scanCount).Result()
		if err != nil {
			return fmt.Errorf("scan %s: %w", pattern, err)
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("del %s: %w", pattern, err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
