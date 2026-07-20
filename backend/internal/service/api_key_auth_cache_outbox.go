package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const apiKeyAuthCacheInvalidationOutboxJobName = "api_key_auth_cache_invalidation_outbox"

func (s *APIKeyService) SetAuthCacheInvalidationOutbox(repo APIKeyAuthCacheInvalidationOutboxRepository, db *sql.DB) {
	s.authCacheInvalidationOutboxRepo = repo
	s.authCacheInvalidationOutboxDB = db
}

func (s *APIKeyService) StartAuthCacheInvalidationOutboxPoller(ctx context.Context) {
	if s == nil || s.authCacheInvalidationOutboxRepo == nil || s.authCacheInvalidationOutboxDB == nil {
		return
	}
	interval := s.authCacheInvalidationOutboxPollInterval()
	if interval <= 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.authOutboxOnce.Do(func() {
		go s.runAuthCacheInvalidationOutboxPoller(ctx, interval)
	})
}

func (s *APIKeyService) authCacheInvalidationOutboxPollInterval() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	seconds := s.cfg.Gateway.Scheduling.OutboxPollIntervalSeconds
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func (s *APIKeyService) runAuthCacheInvalidationOutboxPoller(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.pollAuthCacheInvalidationOutbox(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollAuthCacheInvalidationOutbox(ctx)
		}
	}
}

func (s *APIKeyService) pollAuthCacheInvalidationOutbox(ctx context.Context) {
	if s == nil || s.authCacheInvalidationOutboxRepo == nil || s.authCacheInvalidationOutboxDB == nil {
		return
	}
	lockID := hashAdvisoryLockID(apiKeyAuthCacheInvalidationOutboxJobName)
	release, ok := tryAcquireDBAdvisoryLock(ctx, s.authCacheInvalidationOutboxDB, lockID)
	if !ok {
		return
	}
	defer release()

	var watermark int64
	for {
		batchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		events, err := s.authCacheInvalidationOutboxRepo.ListAfter(batchCtx, watermark, 200)
		cancel()
		if err != nil {
			logger.LegacyPrintf("service.api_key_auth_cache", "[APIKeyAuthCache] outbox poll failed: %v", err)
			return
		}
		if len(events) == 0 {
			return
		}
		for _, event := range events {
			eventCtx, cancelEvent := context.WithTimeout(ctx, 10*time.Second)
			err := s.handleAuthCacheInvalidationOutboxEvent(eventCtx, event)
			cancelEvent()
			if err != nil {
				logger.LegacyPrintf("service.api_key_auth_cache", "[APIKeyAuthCache] outbox handle failed: id=%d type=%s err=%v", event.ID, event.EventType, err)
				return
			}
			watermark = event.ID
		}
		deleteCtx, cancelDelete := context.WithTimeout(ctx, 10*time.Second)
		err = s.authCacheInvalidationOutboxRepo.DeleteThrough(deleteCtx, watermark)
		cancelDelete()
		if err != nil {
			logger.LegacyPrintf("service.api_key_auth_cache", "[APIKeyAuthCache] outbox cleanup failed: watermark=%d err=%v", watermark, err)
			return
		}
	}
}

func (s *APIKeyService) handleAuthCacheInvalidationOutboxEvent(ctx context.Context, event APIKeyAuthCacheInvalidationOutboxEvent) error {
	switch strings.TrimSpace(event.EventType) {
	case APIKeyAuthCacheInvalidationOutboxEventKey:
		if strings.TrimSpace(event.CacheKey) == "" {
			return nil
		}
		s.deleteAuthCache(ctx, strings.TrimSpace(event.CacheKey))
		return nil
	case APIKeyAuthCacheInvalidationOutboxEventUser:
		if event.UserID == nil || *event.UserID <= 0 {
			return nil
		}
		if s.apiKeyRepo == nil {
			return errors.New("api key repository unavailable")
		}
		keys, err := s.apiKeyRepo.ListKeysByUserID(ctx, *event.UserID)
		if err != nil {
			return err
		}
		s.deleteAuthCacheByKeys(ctx, keys)
		return nil
	case APIKeyAuthCacheInvalidationOutboxEventGroup:
		if event.GroupID == nil || *event.GroupID <= 0 {
			return nil
		}
		if s.apiKeyRepo == nil {
			return errors.New("api key repository unavailable")
		}
		keys, err := s.apiKeyRepo.ListKeysByGroupID(ctx, *event.GroupID)
		if err != nil {
			return err
		}
		s.deleteAuthCacheByKeys(ctx, keys)
		return nil
	default:
		return nil
	}
}

func (s *APIKeyService) enqueueAuthCacheInvalidationOutboxKey(ctx context.Context, cacheKey string) {
	if s == nil || s.authCacheInvalidationOutboxRepo == nil || strings.TrimSpace(cacheKey) == "" {
		return
	}
	if err := s.authCacheInvalidationOutboxRepo.EnqueueInvalidateByKey(ctx, strings.TrimSpace(cacheKey)); err != nil {
		logger.LegacyPrintf("service.api_key_auth_cache", "[APIKeyAuthCache] outbox enqueue key failed: err=%v", err)
	}
}

func (s *APIKeyService) enqueueAuthCacheInvalidationOutboxUser(ctx context.Context, userID int64) {
	if s == nil || s.authCacheInvalidationOutboxRepo == nil || userID <= 0 {
		return
	}
	if err := s.authCacheInvalidationOutboxRepo.EnqueueInvalidateByUserID(ctx, userID); err != nil {
		logger.LegacyPrintf("service.api_key_auth_cache", "[APIKeyAuthCache] outbox enqueue user failed: user_id=%d err=%v", userID, err)
	}
}

func (s *APIKeyService) enqueueAuthCacheInvalidationOutboxGroup(ctx context.Context, groupID int64) {
	if s == nil || s.authCacheInvalidationOutboxRepo == nil || groupID <= 0 {
		return
	}
	if err := s.authCacheInvalidationOutboxRepo.EnqueueInvalidateByGroupID(ctx, groupID); err != nil {
		logger.LegacyPrintf("service.api_key_auth_cache", "[APIKeyAuthCache] outbox enqueue group failed: group_id=%d err=%v", groupID, err)
	}
}
