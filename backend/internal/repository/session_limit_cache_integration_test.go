//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SessionLimitCacheSuite struct {
	IntegrationRedisSuite
	cache service.SessionLimitCache
}

func TestSessionLimitCacheSuite(t *testing.T) {
	suite.Run(t, new(SessionLimitCacheSuite))
}

func (s *SessionLimitCacheSuite) SetupTest() {
	s.IntegrationRedisSuite.SetupTest()
	s.cache = NewSessionLimitCache(s.rdb, 5)
}

func (s *SessionLimitCacheSuite) TestRegisterSessionTracksActiveAccountSet() {
	reader, ok := s.cache.(interface {
		GetTrackedActiveAccountIDs(context.Context) ([]int64, error)
	})
	require.True(s.T(), ok)

	accountID := int64(61)
	allowed, err := s.cache.RegisterSession(s.ctx, accountID, "session-1", 3, 2*time.Minute)
	require.NoError(s.T(), err)
	require.True(s.T(), allowed)

	activeIDs, err := reader.GetTrackedActiveAccountIDs(s.ctx)
	require.NoError(s.T(), err)
	require.Contains(s.T(), activeIDs, accountID)

	sessionKey := fmt.Sprintf("%s%d", sessionLimitKeyPrefix, accountID)
	expiredTime := time.Now().Add(-3 * time.Minute).Unix()
	require.NoError(s.T(), s.rdb.ZAdd(s.ctx, sessionKey, redis.Z{
		Score:  float64(expiredTime),
		Member: "session-1",
	}).Err())

	counts, err := s.cache.GetActiveSessionCountBatch(s.ctx, []int64{accountID}, map[int64]time.Duration{
		accountID: time.Minute,
	})
	require.NoError(s.T(), err)
	require.Equal(s.T(), 0, counts[accountID])

	activeIDs, err = reader.GetTrackedActiveAccountIDs(s.ctx)
	require.NoError(s.T(), err)
	require.NotContains(s.T(), activeIDs, accountID)
}
