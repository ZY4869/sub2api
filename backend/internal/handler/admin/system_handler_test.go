package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type systemUpdateServiceStub struct {
	rollbackCalls        int
	rollbackToVersion    string
	rollbackToVersionHit bool
	performCtxErr        error
}

func (s *systemUpdateServiceStub) CheckUpdate(context.Context, bool) (*service.UpdateInfo, error) {
	return &service.UpdateInfo{CurrentVersion: "0.1.380", LatestVersion: "0.1.380", BuildType: "release"}, nil
}

func (s *systemUpdateServiceStub) PerformUpdate(ctx context.Context) error {
	s.performCtxErr = ctx.Err()
	return nil
}

func (s *systemUpdateServiceStub) Rollback() error {
	s.rollbackCalls++
	return nil
}

func (s *systemUpdateServiceStub) RollbackToVersion(_ context.Context, targetVersion string) error {
	s.rollbackToVersion = targetVersion
	s.rollbackToVersionHit = true
	return nil
}

func TestSystemHandlerRollbackTargetVersionContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service.SetDefaultIdempotencyCoordinator(nil)

	updateSvc := &systemUpdateServiceStub{}
	lockRepo := newSystemHandlerIdempotencyRepo()
	lockSvc := service.NewSystemOperationLockService(lockRepo, service.DefaultIdempotencyConfig())
	handler := NewSystemHandler(updateSvc, lockSvc)
	router := gin.New()
	router.POST("/admin/system/rollback", handler.Rollback)

	req := httptest.NewRequest(http.MethodPost, "/admin/system/rollback", strings.NewReader(`{"target_version":" 0.1.378 "}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, updateSvc.rollbackToVersionHit)
	require.Equal(t, "0.1.378", updateSvc.rollbackToVersion)
	require.Zero(t, updateSvc.rollbackCalls)
}

func TestSystemHandlerRollbackKeepsLegacyBackupBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service.SetDefaultIdempotencyCoordinator(nil)

	updateSvc := &systemUpdateServiceStub{}
	lockSvc := service.NewSystemOperationLockService(newSystemHandlerIdempotencyRepo(), service.DefaultIdempotencyConfig())
	handler := NewSystemHandler(updateSvc, lockSvc)
	router := gin.New()
	router.POST("/admin/system/rollback", handler.Rollback)

	req := httptest.NewRequest(http.MethodPost, "/admin/system/rollback", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, updateSvc.rollbackCalls)
	require.False(t, updateSvc.rollbackToVersionHit)
}

func TestSystemHandlerPerformUpdateUsesBackgroundOperationContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service.SetDefaultIdempotencyCoordinator(nil)

	updateSvc := &systemUpdateServiceStub{}
	lockSvc := service.NewSystemOperationLockService(newSystemHandlerIdempotencyRepo(), service.DefaultIdempotencyConfig())
	handler := NewSystemHandler(updateSvc, lockSvc)
	router := gin.New()
	router.POST("/admin/system/update", handler.PerformUpdate)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/admin/system/update", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, updateSvc.performCtxErr)
}

type systemHandlerIdempotencyRepo struct {
	mu     sync.Mutex
	nextID int64
	data   map[string]*service.IdempotencyRecord
}

func newSystemHandlerIdempotencyRepo() *systemHandlerIdempotencyRepo {
	return &systemHandlerIdempotencyRepo{
		nextID: 1,
		data:   map[string]*service.IdempotencyRecord{},
	}
}

func (r *systemHandlerIdempotencyRepo) CreateProcessing(_ context.Context, record *service.IdempotencyRecord) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := r.key(record.Scope, record.IdempotencyKeyHash)
	if _, ok := r.data[key]; ok {
		return false, nil
	}
	stored := cloneSystemHandlerIdempotencyRecord(record)
	stored.ID = r.nextID
	r.nextID++
	r.data[key] = stored
	record.ID = stored.ID
	return true, nil
}

func (r *systemHandlerIdempotencyRepo) GetByScopeAndKeyHash(_ context.Context, scope, keyHash string) (*service.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneSystemHandlerIdempotencyRecord(r.data[r.key(scope, keyHash)]), nil
}

func (r *systemHandlerIdempotencyRepo) TryReclaim(_ context.Context, id int64, fromStatus string, now, newLockedUntil, newExpiresAt time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.data {
		if rec.ID == id && rec.Status == fromStatus && (rec.LockedUntil == nil || !rec.LockedUntil.After(now)) {
			rec.Status = service.IdempotencyStatusProcessing
			rec.LockedUntil = &newLockedUntil
			rec.ExpiresAt = newExpiresAt
			return true, nil
		}
	}
	return false, nil
}

func (r *systemHandlerIdempotencyRepo) ExtendProcessingLock(_ context.Context, id int64, requestFingerprint string, newLockedUntil, newExpiresAt time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.data {
		if rec.ID == id && rec.RequestFingerprint == requestFingerprint && rec.Status == service.IdempotencyStatusProcessing {
			rec.LockedUntil = &newLockedUntil
			rec.ExpiresAt = newExpiresAt
			return true, nil
		}
	}
	return false, nil
}

func (r *systemHandlerIdempotencyRepo) MarkSucceeded(_ context.Context, id int64, responseStatus int, responseBody string, expiresAt time.Time) error {
	return r.finish(id, service.IdempotencyStatusSucceeded, responseStatus, responseBody, "", nil, expiresAt)
}

func (r *systemHandlerIdempotencyRepo) MarkFailedRetryable(_ context.Context, id int64, errorReason string, lockedUntil, expiresAt time.Time) error {
	return r.finish(id, service.IdempotencyStatusFailedRetryable, 0, "", errorReason, &lockedUntil, expiresAt)
}

func (r *systemHandlerIdempotencyRepo) DeleteExpired(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func (r *systemHandlerIdempotencyRepo) key(scope, hash string) string {
	return scope + "|" + hash
}

func (r *systemHandlerIdempotencyRepo) finish(id int64, status string, responseStatus int, responseBody, reason string, lockedUntil *time.Time, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.data {
		if rec.ID != id {
			continue
		}
		rec.Status = status
		rec.LockedUntil = lockedUntil
		rec.ExpiresAt = expiresAt
		if responseStatus > 0 {
			rec.ResponseStatus = &responseStatus
			rec.ResponseBody = &responseBody
		}
		if reason != "" {
			rec.ErrorReason = &reason
		}
		return nil
	}
	return errors.New("idempotency record not found")
}

func cloneSystemHandlerIdempotencyRecord(in *service.IdempotencyRecord) *service.IdempotencyRecord {
	if in == nil {
		return nil
	}
	out := *in
	if in.LockedUntil != nil {
		v := *in.LockedUntil
		out.LockedUntil = &v
	}
	return &out
}
