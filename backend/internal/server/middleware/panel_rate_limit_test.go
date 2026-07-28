package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type panelRateLimitSettingRepo struct {
	values map[string]string
	err    error
}

func (r *panelRateLimitSettingRepo) Get(_ context.Context, _ string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *panelRateLimitSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (r *panelRateLimitSettingRepo) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *panelRateLimitSettingRepo) GetMultiple(_ context.Context, _ []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (r *panelRateLimitSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *panelRateLimitSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *panelRateLimitSettingRepo) Delete(_ context.Context, _ string) error {
	panic("unexpected Delete call")
}

type fakePanelRateLimitChecker struct {
	allowed bool
	err     error
	calls   int
	keys    []string
}

func (f *fakePanelRateLimitChecker) Allow(ctx context.Context, key string, limit int, window time.Duration, opts basemiddleware.RateLimitOptions) (bool, error) {
	f.calls++
	f.keys = append(f.keys, key)
	if f.err != nil {
		return f.allowed, f.err
	}
	return f.allowed, nil
}

func TestPanelRateLimiterDefaultDisabledDoesNotHitRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &panelRateLimitSettingRepo{values: map[string]string{}}
	svc := service.NewSettingService(repo, &config.Config{})
	checker := &fakePanelRateLimitChecker{allowed: true}
	limiter := NewPanelRateLimiter(svc, checker)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 42})
		c.Set(string(ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	router.Use(limiter.User())
	router.GET("/panel", func(c *gin.Context) { c.Status(http.StatusOK) })

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/panel", nil))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 0, checker.calls)
}

func TestPanelRateLimiterUserLimitReturns429(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &panelRateLimitSettingRepo{values: map[string]string{
		service.SettingKeyPanelRateLimitSettings: `{"enabled":true,"user_rpm":1,"heavy_rpm":1,"public_ip_rpm":1,"exempt_admin":false}`,
	}}
	svc := service.NewSettingService(repo, &config.Config{})
	checker := &fakePanelRateLimitChecker{allowed: false}
	limiter := NewPanelRateLimiter(svc, checker)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 42})
		c.Set(string(ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	router.Use(limiter.User())
	router.GET("/panel", func(c *gin.Context) { c.Status(http.StatusOK) })

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/panel", nil))
	require.Equal(t, http.StatusTooManyRequests, resp.Code)
	require.Equal(t, []string{"panel:user:user:42"}, checker.keys)
}

func TestPanelRateLimiterAdminExemptAndPublicPrivateIPSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &panelRateLimitSettingRepo{values: map[string]string{
		service.SettingKeyPanelRateLimitSettings: `{"enabled":true,"user_rpm":1,"heavy_rpm":1,"public_ip_rpm":1,"exempt_admin":true}`,
	}}
	svc := service.NewSettingService(repo, &config.Config{})
	checker := &fakePanelRateLimitChecker{allowed: true}
	limiter := NewPanelRateLimiter(svc, checker)

	adminRouter := gin.New()
	adminRouter.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})
		c.Set(string(ContextKeyUserRole), service.RoleAdmin)
		c.Next()
	})
	adminRouter.Use(limiter.Heavy())
	adminRouter.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })

	resp := httptest.NewRecorder()
	adminRouter.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/admin", nil))
	require.Equal(t, http.StatusOK, resp.Code)

	publicRouter := gin.New()
	publicRouter.Use(limiter.PublicIP())
	publicRouter.GET("/public", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	resp = httptest.NewRecorder()
	publicRouter.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 0, checker.calls)
}

func TestPanelRateLimiterFailOpenWhenCheckerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &panelRateLimitSettingRepo{values: map[string]string{
		service.SettingKeyPanelRateLimitSettings: `{"enabled":true,"user_rpm":1,"heavy_rpm":1,"public_ip_rpm":1,"exempt_admin":false}`,
	}}
	svc := service.NewSettingService(repo, &config.Config{})
	checker := &fakePanelRateLimitChecker{allowed: true, err: errors.New("redis unavailable")}
	limiter := NewPanelRateLimiter(svc, checker)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 42})
		c.Set(string(ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	router.Use(limiter.User())
	router.GET("/panel", func(c *gin.Context) { c.Status(http.StatusOK) })

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/panel", nil))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 1, checker.calls)
}

func TestPanelRateLimiterCachesSettingsBriefly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &panelRateLimitSettingRepo{values: map[string]string{
		service.SettingKeyPanelRateLimitSettings: `{"enabled":true,"user_rpm":10,"heavy_rpm":10,"public_ip_rpm":10,"exempt_admin":false}`,
	}}
	svc := service.NewSettingService(repo, &config.Config{})
	checker := &fakePanelRateLimitChecker{allowed: true}
	limiter := NewPanelRateLimiter(svc, checker)
	now := time.Unix(1000, 0)
	limiter.now = func() time.Time { return now }

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 42})
		c.Set(string(ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	router.Use(limiter.User())
	router.GET("/panel", func(c *gin.Context) { c.Status(http.StatusOK) })

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/panel", nil))
	require.Equal(t, http.StatusOK, resp.Code)

	repo.values[service.SettingKeyPanelRateLimitSettings] = `{"enabled":false,"user_rpm":10,"heavy_rpm":10,"public_ip_rpm":10,"exempt_admin":false}`
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/panel", nil))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 2, checker.calls)
}
