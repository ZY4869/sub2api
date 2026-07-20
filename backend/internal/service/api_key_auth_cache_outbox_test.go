package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type authOutboxCacheStub struct {
	APIKeyCache
	deleted   []string
	published []string
}

func (s *authOutboxCacheStub) DeleteAuthCache(ctx context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *authOutboxCacheStub) PublishAuthCacheInvalidation(ctx context.Context, cacheKey string) error {
	s.published = append(s.published, cacheKey)
	return nil
}

type authOutboxRepoStub struct {
	APIKeyAuthCacheInvalidationOutboxRepository
	keyEvents   []string
	userEvents  []int64
	groupEvents []int64
}

func (s *authOutboxRepoStub) EnqueueInvalidateByKey(ctx context.Context, cacheKey string) error {
	s.keyEvents = append(s.keyEvents, cacheKey)
	return nil
}

func (s *authOutboxRepoStub) EnqueueInvalidateByUserID(ctx context.Context, userID int64) error {
	s.userEvents = append(s.userEvents, userID)
	return nil
}

func (s *authOutboxRepoStub) EnqueueInvalidateByGroupID(ctx context.Context, groupID int64) error {
	s.groupEvents = append(s.groupEvents, groupID)
	return nil
}

type authOutboxAPIKeyRepoStub struct {
	APIKeyRepository
	keysByUserID  map[int64][]string
	keysByGroupID map[int64][]string
}

func (s *authOutboxAPIKeyRepoStub) ListKeysByUserID(ctx context.Context, userID int64) ([]string, error) {
	return append([]string(nil), s.keysByUserID[userID]...), nil
}

func (s *authOutboxAPIKeyRepoStub) ListKeysByGroupID(ctx context.Context, groupID int64) ([]string, error) {
	return append([]string(nil), s.keysByGroupID[groupID]...), nil
}

func TestAPIKeyServiceInvalidateAuthCacheByKeyEnqueuesOutboxHash(t *testing.T) {
	cache := &authOutboxCacheStub{}
	outbox := &authOutboxRepoStub{}
	svc := &APIKeyService{
		cache:                           cache,
		authCacheInvalidationOutboxRepo: outbox,
	}

	svc.InvalidateAuthCacheByKey(context.Background(), "sk-local")

	expected := svc.authCacheKey("sk-local")
	require.Equal(t, []string{expected}, cache.deleted)
	require.Equal(t, []string{expected}, cache.published)
	require.Equal(t, []string{expected}, outbox.keyEvents)
}

func TestAPIKeyServiceInvalidateAuthCacheByUserAndGroupEnqueuesDurableIntent(t *testing.T) {
	cache := &authOutboxCacheStub{}
	outbox := &authOutboxRepoStub{}
	svc := &APIKeyService{
		apiKeyRepo: &authOutboxAPIKeyRepoStub{
			keysByUserID: map[int64][]string{7: []string{"user-key"}},
			keysByGroupID: map[int64][]string{
				11: []string{"group-key"},
			},
		},
		cache:                           cache,
		authCacheInvalidationOutboxRepo: outbox,
	}

	svc.InvalidateAuthCacheByUserID(context.Background(), 7)
	svc.InvalidateAuthCacheByGroupID(context.Background(), 11)

	require.Equal(t, []int64{7}, outbox.userEvents)
	require.Equal(t, []int64{11}, outbox.groupEvents)
	require.Equal(t, []string{svc.authCacheKey("user-key"), svc.authCacheKey("group-key")}, cache.deleted)
	require.Equal(t, cache.deleted, cache.published)
}

func TestAPIKeyServiceAuthCacheOutboxEventReplaysInvalidations(t *testing.T) {
	cache := &authOutboxCacheStub{}
	svc := &APIKeyService{
		apiKeyRepo: &authOutboxAPIKeyRepoStub{
			keysByUserID: map[int64][]string{
				7: []string{"user-key-a", "user-key-b"},
			},
			keysByGroupID: map[int64][]string{
				11: []string{"group-key"},
			},
		},
		cache: cache,
	}

	keyHash := svc.authCacheKey("direct-key")
	userID := int64(7)
	groupID := int64(11)
	require.NoError(t, svc.handleAuthCacheInvalidationOutboxEvent(context.Background(), APIKeyAuthCacheInvalidationOutboxEvent{
		EventType: APIKeyAuthCacheInvalidationOutboxEventKey,
		CacheKey:  keyHash,
	}))
	require.NoError(t, svc.handleAuthCacheInvalidationOutboxEvent(context.Background(), APIKeyAuthCacheInvalidationOutboxEvent{
		EventType: APIKeyAuthCacheInvalidationOutboxEventUser,
		UserID:    &userID,
	}))
	require.NoError(t, svc.handleAuthCacheInvalidationOutboxEvent(context.Background(), APIKeyAuthCacheInvalidationOutboxEvent{
		EventType: APIKeyAuthCacheInvalidationOutboxEventGroup,
		GroupID:   &groupID,
	}))

	require.Equal(t, []string{
		keyHash,
		svc.authCacheKey("user-key-a"),
		svc.authCacheKey("user-key-b"),
		svc.authCacheKey("group-key"),
	}, cache.deleted)
	require.Equal(t, cache.deleted, cache.published)
}
