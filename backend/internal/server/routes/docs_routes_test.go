package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type routesDocsSettingRepoStub struct {
	values map[string]string
}

func (s *routesDocsSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *routesDocsSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *routesDocsSettingRepoStub) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *routesDocsSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *routesDocsSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *routesDocsSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *routesDocsSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func newDocsRoutesTestServices() (*service.SettingService, *service.APIDocsService) {
	repo := &routesDocsSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAPIDocsMarkdown: "# 路由测试文档\n",
		},
	}
	return service.NewSettingService(repo, nil), service.NewAPIDocsService(repo)
}

func TestUserDocsRoute_RequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingService, docsService := newDocsRoutesTestServices()
	router := gin.New()
	v1 := router.Group("/api/v1")

	jwtAuth := servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1, Concurrency: 1})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleUser)
		c.Next()
	})

	RegisterUserRoutes(v1, &handler.Handlers{
		Docs: handler.NewDocsHandler(docsService),
	}, jwtAuth, settingService)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs/api", nil)
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/docs/api", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAdminDocsRoute_RequiresAdminAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, docsService := newDocsRoutesTestServices()
	router := gin.New()
	v1 := router.Group("/api/v1")

	adminAuth := servermiddleware.AdminAuthMiddleware(func(c *gin.Context) {
		switch c.GetHeader("Authorization") {
		case "":
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		case "Bearer admin-token":
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1, Concurrency: 1})
			c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleAdmin)
			c.Next()
		default:
			c.AbortWithStatus(http.StatusForbidden)
		}
	})

	RegisterAdminRoutes(v1, &handler.Handlers{
		Admin: &handler.AdminHandlers{
			Docs: adminhandler.NewDocsHandler(docsService),
		},
	}, adminAuth)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/docs/api", nil)
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/docs/api", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusForbidden, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/docs/api", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
}
