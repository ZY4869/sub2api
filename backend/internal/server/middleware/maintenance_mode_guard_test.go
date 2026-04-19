//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mmSettingRepo struct {
	values map[string]string
}

func (r *mmSettingRepo) Get(_ context.Context, _ string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *mmSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	v, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return v, nil
}

func (r *mmSettingRepo) Set(_ context.Context, _, _ string) error {
	panic("unexpected Set call")
}

func (r *mmSettingRepo) GetMultiple(_ context.Context, _ []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (r *mmSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = make(map[string]string, len(settings))
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *mmSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *mmSettingRepo) Delete(_ context.Context, _ string) error {
	panic("unexpected Delete call")
}

func newMaintenanceModeSettingService(t *testing.T, enabled string) *service.SettingService {
	t.Helper()

	repo := &mmSettingRepo{
		values: map[string]string{
			service.SettingKeyMaintenanceModeEnabled: enabled,
		},
	}
	svc := service.NewSettingService(repo, &config.Config{})
	require.NoError(t, svc.UpdateSettings(context.Background(), &service.SystemSettings{
		MaintenanceModeEnabled: enabled == "true",
	}))

	return svc
}

func TestMaintenanceModeUserGuard(t *testing.T) {
	tests := []struct {
		name       string
		nilService bool
		enabled    string
		role       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "disabled_allows_all",
			enabled:    "false",
			role:       "user",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nil_service_allows_all",
			nilService: true,
			role:       "user",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_admin_allowed",
			enabled:    "true",
			role:       "admin",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_user_blocked",
			enabled:    "true",
			role:       "user",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   service.MaintenanceModeErrorCode,
		},
		{
			name:       "enabled_missing_role_blocked",
			enabled:    "true",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   service.MaintenanceModeMessage,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			r := gin.New()
			if tc.role != "" {
				role := tc.role
				r.Use(func(c *gin.Context) {
					c.Set(string(ContextKeyUserRole), role)
					c.Next()
				})
			}

			var svc *service.SettingService
			if !tc.nilService {
				svc = newMaintenanceModeSettingService(t, tc.enabled)
			}

			r.Use(MaintenanceModeUserGuard(svc))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)
			if tc.wantBody != "" {
				require.Contains(t, w.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestMaintenanceModeAuthGuard(t *testing.T) {
	tests := []struct {
		name       string
		nilService bool
		enabled    string
		path       string
		wantStatus int
	}{
		{
			name:       "disabled_allows_all",
			enabled:    "false",
			path:       "/api/v1/auth/register",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nil_service_allows_all",
			nilService: true,
			path:       "/api/v1/auth/register",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_allows_login",
			enabled:    "true",
			path:       "/api/v1/auth/login",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_allows_login_2fa",
			enabled:    "true",
			path:       "/api/v1/auth/login/2fa",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_allows_logout",
			enabled:    "true",
			path:       "/api/v1/auth/logout",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_allows_refresh",
			enabled:    "true",
			path:       "/api/v1/auth/refresh",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_blocks_register",
			enabled:    "true",
			path:       "/api/v1/auth/register",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "enabled_blocks_forgot_password",
			enabled:    "true",
			path:       "/api/v1/auth/forgot-password",
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			r := gin.New()

			var svc *service.SettingService
			if !tc.nilService {
				svc = newMaintenanceModeSettingService(t, tc.enabled)
			}

			r.Use(MaintenanceModeAuthGuard(svc))
			r.Any("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)
			if tc.wantStatus == http.StatusServiceUnavailable {
				require.Contains(t, w.Body.String(), service.MaintenanceModeErrorCode)
			}
		})
	}
}

func TestMaintenanceModeGatewayGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("google_writer_preserves_google_style", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(ContextKeyUserRole), service.RoleUser)
			c.Next()
		})
		r.Use(MaintenanceModeGatewayGuard(newMaintenanceModeSettingService(t, "true"), "google", GoogleErrorWriter))
		r.POST("/v1beta/models", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.Contains(t, w.Body.String(), service.MaintenanceModeMessage)
		require.Contains(t, w.Body.String(), `"status":"UNAVAILABLE"`)
	})

	t.Run("json_writer_preserves_common_error_code", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(ContextKeyUserRole), service.RoleUser)
			c.Next()
		})
		r.Use(MaintenanceModeGatewayGuard(newMaintenanceModeSettingService(t, "true"), "document_ai", JSONServiceUnavailableWriter))
		r.POST("/document-ai/v1beta/models", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/document-ai/v1beta/models", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.Contains(t, w.Body.String(), service.MaintenanceModeMessage)
		require.Contains(t, w.Body.String(), service.MaintenanceModeErrorCode)
	})
}
