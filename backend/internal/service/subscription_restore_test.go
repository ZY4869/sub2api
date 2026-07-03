package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type restoreSubscriptionRepoStub struct {
	userSubRepoNoop

	sub           *UserSubscription
	active        *UserSubscription
	restoreStatus string
	restoreCalls  int
}

func (s *restoreSubscriptionRepoStub) GetByIDIncludingDeleted(_ context.Context, id int64) (*UserSubscription, error) {
	if s.sub == nil || s.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *s.sub
	return &cp, nil
}

func (s *restoreSubscriptionRepoStub) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	if s.active == nil {
		return nil, ErrSubscriptionNotFound
	}
	if s.active.UserID == userID && s.active.GroupID == groupID {
		cp := *s.active
		return &cp, nil
	}
	return nil, ErrSubscriptionNotFound
}

func (s *restoreSubscriptionRepoStub) Restore(_ context.Context, id int64, status string) error {
	if s.sub == nil || s.sub.ID != id {
		return ErrSubscriptionNotFound
	}
	s.restoreCalls++
	s.restoreStatus = status
	s.sub.Status = status
	s.sub.DeletedAt = nil
	return nil
}

func (s *restoreSubscriptionRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if s.sub == nil || s.sub.ID != id || s.sub.DeletedAt != nil {
		return nil, ErrSubscriptionNotFound
	}
	cp := *s.sub
	return &cp, nil
}

func TestSubscriptionServiceRestoreSubscription(t *testing.T) {
	now := time.Now()

	t.Run("restores revoked subscription as active", func(t *testing.T) {
		deletedAt := now.Add(-time.Hour)
		repo := &restoreSubscriptionRepoStub{sub: &UserSubscription{
			ID:        10,
			UserID:    100,
			GroupID:   200,
			Status:    SubscriptionStatusRevoked,
			DeletedAt: &deletedAt,
			ExpiresAt: now.Add(time.Hour),
		}}
		svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

		sub, err := svc.RestoreSubscription(context.Background(), 10)
		require.NoError(t, err)
		require.Equal(t, SubscriptionStatusActive, sub.Status)
		require.Nil(t, sub.DeletedAt)
		require.Equal(t, 1, repo.restoreCalls)
		require.Equal(t, SubscriptionStatusActive, repo.restoreStatus)
	})

	t.Run("restores expired revoked subscription as expired", func(t *testing.T) {
		deletedAt := now.Add(-time.Hour)
		repo := &restoreSubscriptionRepoStub{sub: &UserSubscription{
			ID:        11,
			UserID:    101,
			GroupID:   201,
			Status:    SubscriptionStatusRevoked,
			DeletedAt: &deletedAt,
			ExpiresAt: now.Add(-time.Minute),
		}}
		svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

		sub, err := svc.RestoreSubscription(context.Background(), 11)
		require.NoError(t, err)
		require.Equal(t, SubscriptionStatusExpired, sub.Status)
		require.Equal(t, SubscriptionStatusExpired, repo.restoreStatus)
	})

	t.Run("rejects subscription that is not revoked", func(t *testing.T) {
		repo := &restoreSubscriptionRepoStub{sub: &UserSubscription{
			ID:        12,
			UserID:    102,
			GroupID:   202,
			Status:    SubscriptionStatusActive,
			ExpiresAt: now.Add(time.Hour),
		}}
		svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

		_, err := svc.RestoreSubscription(context.Background(), 12)
		require.ErrorIs(t, err, ErrSubscriptionNotRevoked)
		require.Zero(t, repo.restoreCalls)
	})

	t.Run("rejects when same user group already has active subscription", func(t *testing.T) {
		deletedAt := now.Add(-time.Hour)
		repo := &restoreSubscriptionRepoStub{
			sub: &UserSubscription{
				ID:        13,
				UserID:    103,
				GroupID:   203,
				Status:    SubscriptionStatusRevoked,
				DeletedAt: &deletedAt,
				ExpiresAt: now.Add(time.Hour),
			},
			active: &UserSubscription{
				ID:        14,
				UserID:    103,
				GroupID:   203,
				Status:    SubscriptionStatusActive,
				ExpiresAt: now.Add(time.Hour),
			},
		}
		svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

		_, err := svc.RestoreSubscription(context.Background(), 13)
		require.True(t, errors.Is(err, ErrSubscriptionAlreadyExists))
		require.Zero(t, repo.restoreCalls)
	})
}
