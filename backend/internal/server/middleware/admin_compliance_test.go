package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type complianceSettingRepo struct {
	values map[string]string
}

func (r *complianceSettingRepo) Get(_ context.Context, key string) (*service.Setting, error) {
	value, err := r.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (r *complianceSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (r *complianceSettingRepo) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *complianceSettingRepo) GetMultiple(_ context.Context, _ []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (r *complianceSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *complianceSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *complianceSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestAdminComplianceGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newRouter := func(svc *service.SettingService, path string) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(ContextKeyUser), AuthSubject{UserID: 101})
			c.Set(string(ContextKeyUserRole), service.RoleAdmin)
			c.Next()
		})
		r.Use(AdminComplianceGuard(svc))
		r.GET(path, func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})
		return r
	}

	t.Run("disabled allows admin route", func(t *testing.T) {
		svc := service.NewSettingService(&complianceSettingRepo{values: map[string]string{}}, nil)
		rec := httptest.NewRecorder()
		newRouter(svc, "/api/v1/admin/users").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("enabled blocks unacknowledged admin route", func(t *testing.T) {
		svc := service.NewSettingService(&complianceSettingRepo{values: map[string]string{
			service.SettingKeyAdminComplianceEnabled: "true",
		}}, nil)
		rec := httptest.NewRecorder()
		newRouter(svc, "/api/v1/admin/users").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil))
		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Contains(t, rec.Body.String(), "ADMIN_COMPLIANCE_REQUIRED")
	})

	t.Run("enabled allows compliance status route", func(t *testing.T) {
		svc := service.NewSettingService(&complianceSettingRepo{values: map[string]string{
			service.SettingKeyAdminComplianceEnabled: "true",
		}}, nil)
		rec := httptest.NewRecorder()
		newRouter(svc, "/api/v1/admin/compliance/status").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/compliance/status", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("enabled does not bypass similar admin paths", func(t *testing.T) {
		svc := service.NewSettingService(&complianceSettingRepo{values: map[string]string{
			service.SettingKeyAdminComplianceEnabled: "true",
		}}, nil)
		rec := httptest.NewRecorder()
		newRouter(svc, "/api/v1/admin/compliance-report").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/compliance-report", nil))
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("enabled allows acknowledged admin route", func(t *testing.T) {
		repo := &complianceSettingRepo{values: map[string]string{
			service.SettingKeyAdminComplianceEnabled: "true",
		}}
		svc := service.NewSettingService(repo, nil)
		_, err := svc.AcknowledgeAdminCompliance(context.Background(), 101)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		newRouter(svc, "/api/v1/admin/users").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	})
}
