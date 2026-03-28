package service

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type TLSFingerprintProfileRepository interface {
	List(ctx context.Context) ([]*model.TLSFingerprintProfile, error)
	GetByID(ctx context.Context, id int64) (*model.TLSFingerprintProfile, error)
	Create(ctx context.Context, profile *model.TLSFingerprintProfile) (*model.TLSFingerprintProfile, error)
	Update(ctx context.Context, profile *model.TLSFingerprintProfile) (*model.TLSFingerprintProfile, error)
	Delete(ctx context.Context, id int64) error
}

type TLSFingerprintProfileCache interface {
	Get(ctx context.Context) ([]*model.TLSFingerprintProfile, bool)
	Set(ctx context.Context, profiles []*model.TLSFingerprintProfile) error
	Invalidate(ctx context.Context) error
	NotifyUpdate(ctx context.Context) error
	SubscribeUpdates(ctx context.Context, handler func())
}

type TLSFingerprintProfileService struct {
	repo  TLSFingerprintProfileRepository
	cache TLSFingerprintProfileCache

	localCache map[int64]*model.TLSFingerprintProfile
	localMu    sync.RWMutex
}

func NewTLSFingerprintProfileService(repo TLSFingerprintProfileRepository, cache TLSFingerprintProfileCache) *TLSFingerprintProfileService {
	svc := &TLSFingerprintProfileService{
		repo:       repo,
		cache:      cache,
		localCache: make(map[int64]*model.TLSFingerprintProfile),
	}

	ctx := context.Background()
	if err := svc.reloadFromDB(ctx); err != nil {
		logger.LegacyPrintf("service.tls_fingerprint_profile", "[TLSFingerprintProfileService] startup load failed: %v", err)
		if fallbackErr := svc.refreshLocalCache(ctx); fallbackErr != nil {
			logger.LegacyPrintf("service.tls_fingerprint_profile", "[TLSFingerprintProfileService] cache fallback failed: %v", fallbackErr)
		}
	}

	if cache != nil {
		cache.SubscribeUpdates(ctx, func() {
			if err := svc.refreshLocalCache(context.Background()); err != nil {
				logger.LegacyPrintf("service.tls_fingerprint_profile", "[TLSFingerprintProfileService] refresh on pubsub failed: %v", err)
			}
		})
	}

	return svc
}

func (s *TLSFingerprintProfileService) List(ctx context.Context) ([]*model.TLSFingerprintProfile, error) {
	return s.repo.List(ctx)
}

func (s *TLSFingerprintProfileService) GetByID(ctx context.Context, id int64) (*model.TLSFingerprintProfile, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TLSFingerprintProfileService) Create(ctx context.Context, profile *model.TLSFingerprintProfile) (*model.TLSFingerprintProfile, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	created, err := s.repo.Create(ctx, profile)
	if err != nil {
		return nil, err
	}
	refreshCtx, cancel := s.newRefreshContext()
	defer cancel()
	s.invalidateAndNotify(refreshCtx)
	return created, nil
}

func (s *TLSFingerprintProfileService) Update(ctx context.Context, profile *model.TLSFingerprintProfile) (*model.TLSFingerprintProfile, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, profile)
	if err != nil {
		return nil, err
	}
	refreshCtx, cancel := s.newRefreshContext()
	defer cancel()
	s.invalidateAndNotify(refreshCtx)
	return updated, nil
}

func (s *TLSFingerprintProfileService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	refreshCtx, cancel := s.newRefreshContext()
	defer cancel()
	s.invalidateAndNotify(refreshCtx)
	return nil
}

func (s *TLSFingerprintProfileService) GetProfileByID(id int64) *tlsfingerprint.Profile {
	s.localMu.RLock()
	profile := s.localCache[id]
	s.localMu.RUnlock()
	if profile == nil {
		return nil
	}
	return profile.ToTLSProfile()
}

func (s *TLSFingerprintProfileService) ResolveTLSProfile(account *Account) *tlsfingerprint.Profile {
	if account == nil || !account.IsTLSFingerprintEnabled() {
		return nil
	}
	profileID := account.GetTLSFingerprintProfileID()
	if profileID > 0 {
		if profile := s.GetProfileByID(profileID); profile != nil {
			return profile
		}
	}
	if profileID == -1 {
		if profile := s.getRandomProfile(); profile != nil {
			return profile
		}
	}
	return defaultTLSFingerprintProfile()
}

func (s *TLSFingerprintProfileService) getRandomProfile() *tlsfingerprint.Profile {
	s.localMu.RLock()
	defer s.localMu.RUnlock()

	if len(s.localCache) == 0 {
		return nil
	}
	profiles := make([]*model.TLSFingerprintProfile, 0, len(s.localCache))
	for _, profile := range s.localCache {
		if profile != nil {
			profiles = append(profiles, profile)
		}
	}
	if len(profiles) == 0 {
		return nil
	}
	return profiles[rand.IntN(len(profiles))].ToTLSProfile()
}

func (s *TLSFingerprintProfileService) refreshLocalCache(ctx context.Context) error {
	if s.cache != nil {
		if profiles, ok := s.cache.Get(ctx); ok {
			s.setLocalCache(profiles)
			return nil
		}
	}
	return s.reloadFromDB(ctx)
}

func (s *TLSFingerprintProfileService) reloadFromDB(ctx context.Context) error {
	profiles, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	if s.cache != nil {
		if err := s.cache.Set(ctx, profiles); err != nil {
			logger.LegacyPrintf("service.tls_fingerprint_profile", "[TLSFingerprintProfileService] cache set failed: %v", err)
		}
	}
	s.setLocalCache(profiles)
	return nil
}

func (s *TLSFingerprintProfileService) setLocalCache(profiles []*model.TLSFingerprintProfile) {
	cache := make(map[int64]*model.TLSFingerprintProfile, len(profiles))
	for _, profile := range profiles {
		if profile != nil {
			cache[profile.ID] = profile
		}
	}
	s.localMu.Lock()
	s.localCache = cache
	s.localMu.Unlock()
}

func (s *TLSFingerprintProfileService) newRefreshContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Second)
}

func (s *TLSFingerprintProfileService) invalidateAndNotify(ctx context.Context) {
	if s.cache != nil {
		if err := s.cache.Invalidate(ctx); err != nil {
			logger.LegacyPrintf("service.tls_fingerprint_profile", "[TLSFingerprintProfileService] cache invalidate failed: %v", err)
		}
	}
	if err := s.reloadFromDB(ctx); err != nil {
		logger.LegacyPrintf("service.tls_fingerprint_profile", "[TLSFingerprintProfileService] local cache reload failed: %v", err)
		s.localMu.Lock()
		s.localCache = make(map[int64]*model.TLSFingerprintProfile)
		s.localMu.Unlock()
	}
	if s.cache != nil {
		if err := s.cache.NotifyUpdate(ctx); err != nil {
			logger.LegacyPrintf("service.tls_fingerprint_profile", "[TLSFingerprintProfileService] cache notify failed: %v", err)
		}
	}
}
