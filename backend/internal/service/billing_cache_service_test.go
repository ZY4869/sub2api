package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type billingCacheWorkerStub struct {
	balanceUpdates      int64
	subscriptionUpdates int64
}

func (b *billingCacheWorkerStub) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	return 0, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetUserBalance(ctx context.Context, userID int64, balance float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) DeductUserBalance(ctx context.Context, userID int64, amount float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateUserBalance(ctx context.Context, userID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *APIKeyRateLimitCacheData) error {
	return nil
}

func (b *billingCacheWorkerStub) UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error {
	return nil
}

func (b *billingCacheWorkerStub) InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error {
	return nil
}

func TestBillingCacheServiceQueueHighLoad(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	start := time.Now()
	for i := 0; i < cacheWriteBufferSize*2; i++ {
		svc.QueueDeductBalance(1, 1)
	}
	require.Less(t, time.Since(start), 2*time.Second)

	svc.QueueUpdateSubscriptionUsage(1, 2, 1.5)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.balanceUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.subscriptionUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestBillingCacheServiceEnqueueAfterStopReturnsFalse(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, &config.Config{})
	svc.Stop()

	enqueued := svc.enqueueCacheWrite(cacheWriteTask{
		kind:   cacheWriteDeductBalance,
		userID: 1,
		amount: 1,
	})
	require.False(t, enqueued)
}

type billingCacheUserRepoStub struct {
	UserRepository
	user     *User
	getErr   error
	getCalls int
}

func (s *billingCacheUserRepoStub) GetByID(ctx context.Context, id int64) (*User, error) {
	s.getCalls++
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.user != nil {
		return s.user, nil
	}
	return &User{ID: id, Balance: 1}, nil
}

type billingCacheHoldRepoStub struct {
	reserveCalls int
	lastHold     *BillingHold
	err          error
}

func (s *billingCacheHoldRepoStub) Reserve(ctx context.Context, hold *BillingHold) (*BillingHold, error) {
	s.reserveCalls++
	if hold != nil {
		cloned := *hold
		s.lastHold = &cloned
	}
	if s.err != nil {
		return nil, s.err
	}
	if hold == nil {
		return nil, ErrInvalidBillingAmount
	}
	out := *hold
	out.Status = BillingHoldStatusHeld
	return &out, nil
}

func (s *billingCacheHoldRepoStub) Settle(ctx context.Context, requestID string, apiKeyID int64, actualAmount float64) (*BillingHold, error) {
	return nil, ErrBillingHoldNotFound
}

func (s *billingCacheHoldRepoStub) Release(ctx context.Context, requestID string, apiKeyID int64) (*BillingHold, error) {
	return nil, ErrBillingHoldNotFound
}

type billingCacheAPIKeyRepoStub struct {
	APIKeyRepository
	holdRepo      *billingCacheHoldRepoStub
	rateLimitData *APIKeyRateLimitData
	rateCalls     int
}

func (s *billingCacheAPIKeyRepoStub) BillingHoldRepository() BillingHoldRepository {
	return s.holdRepo
}

func (s *billingCacheAPIKeyRepoStub) ResetRateLimitWindows(ctx context.Context, id int64) error {
	return nil
}

func (s *billingCacheAPIKeyRepoStub) GetRateLimitData(ctx context.Context, id int64) (*APIKeyRateLimitData, error) {
	s.rateCalls++
	if s.rateLimitData != nil {
		return s.rateLimitData, nil
	}
	return &APIKeyRateLimitData{}, nil
}

func TestBillingCacheServiceCheckBillingEligibility_RejectsRateLimitBeforeReserve(t *testing.T) {
	now := time.Now()
	user := &User{ID: 10, Balance: 10}
	holdRepo := &billingCacheHoldRepoStub{}
	apiKeyRepo := &billingCacheAPIKeyRepoStub{
		holdRepo:      holdRepo,
		rateLimitData: &APIKeyRateLimitData{Usage5h: 1, Window5hStart: &now},
	}
	svc := NewBillingCacheService(nil, &billingCacheUserRepoStub{user: user}, nil, apiKeyRepo, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(context.Background(), user, &APIKey{
		ID:          20,
		User:        user,
		RateLimit5h: 1,
	}, nil, nil)

	require.ErrorIs(t, err, ErrAPIKeyRateLimit5hExceeded)
	require.Equal(t, 1, apiKeyRepo.rateCalls)
	require.Equal(t, 0, holdRepo.reserveCalls)
}

func TestBillingCacheServiceCheckBillingEligibility_ReserveUsesPayloadFingerprint(t *testing.T) {
	user := &User{ID: 11, Balance: 10}
	holdRepo := &billingCacheHoldRepoStub{}
	svc := NewBillingCacheService(nil, &billingCacheUserRepoStub{user: user}, nil, &billingCacheAPIKeyRepoStub{holdRepo: holdRepo}, &config.Config{})
	t.Cleanup(svc.Stop)
	ctx := context.WithValue(context.Background(), ctxkey.RequestPayloadHash, "payload-hash-1")

	apiKey := &APIKey{ID: 21, User: user}
	err := svc.CheckBillingEligibility(ctx, user, apiKey, nil, nil)

	require.NoError(t, err)
	require.Equal(t, 1, holdRepo.reserveCalls)
	require.NotNil(t, holdRepo.lastHold)
	require.Equal(t, "payload-hash-1", holdRepo.lastHold.RequestFingerprint)
	require.NotNil(t, apiKey.BillingHold)
	require.Equal(t, BillingHoldStatusHeld, apiKey.BillingHold.Status)
}

func TestBillingCacheServiceCheckBillingEligibility_ExistingHoldSkipsBalanceLookup(t *testing.T) {
	user := &User{ID: 12, Balance: 0}
	userRepo := &billingCacheUserRepoStub{getErr: errors.New("balance lookup should not run")}
	svc := NewBillingCacheService(nil, userRepo, nil, &billingCacheAPIKeyRepoStub{holdRepo: &billingCacheHoldRepoStub{}}, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(context.Background(), user, &APIKey{
		ID:   22,
		User: user,
		BillingHold: &BillingHold{
			RequestID: "req-held",
			APIKeyID:  22,
			UserID:    user.ID,
			Amount:    0.01,
			Status:    BillingHoldStatusHeld,
		},
	}, nil, nil)

	require.NoError(t, err)
	require.Equal(t, 0, userRepo.getCalls)
}

func TestBillingCacheServiceCheckBillingEligibility_ReplayedHoldReturnsConflict(t *testing.T) {
	user := &User{ID: 13, Balance: 10}
	holdRepo := &billingCacheHoldRepoStub{err: ErrBillingRequestReplayed}
	svc := NewBillingCacheService(nil, &billingCacheUserRepoStub{user: user}, nil, &billingCacheAPIKeyRepoStub{holdRepo: holdRepo}, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(context.Background(), user, &APIKey{ID: 23, User: user}, nil, nil)

	require.ErrorIs(t, err, ErrBillingRequestReplayed)
}
