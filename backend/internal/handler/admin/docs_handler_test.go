package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminDocsSettingRepoStub struct {
	values map[string]string
}

func (s *adminDocsSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *adminDocsSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *adminDocsSettingRepoStub) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *adminDocsSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *adminDocsSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *adminDocsSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *adminDocsSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func newAdminDocsContext(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 9, Concurrency: 1})
	ctx.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
	return ctx, recorder
}

func TestAdminDocsHandlerGetAPIReference_ReturnsDocumentState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAPIDocsMarkdown: "# 管理员覆盖文档\n",
		},
	}
	docsService := service.NewAPIDocsService(repo)
	handler := NewDocsHandler(docsService)

	ctx, recorder := newAdminDocsContext(http.MethodGet, "/api/v1/admin/docs/api", nil)
	handler.GetAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var body response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)

	data, ok := body.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "# 管理员覆盖文档\n", data["effective_content"])
	require.Equal(t, docsService.GetDefaultContent(), data["default_content"])
	require.Equal(t, true, data["has_override"])
}

func TestAdminDocsHandlerUpdateAPIReference_SavesOverrideAndReturnsDocument(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{values: map[string]string{}}
	docsService := service.NewAPIDocsService(repo)
	handler := NewDocsHandler(docsService)

	ctx, recorder := newAdminDocsContext(
		http.MethodPut,
		"/api/v1/admin/docs/api",
		[]byte(`{"content":"# 新文档\n\n说明"}`),
	)

	handler.UpdateAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "# 新文档\n\n说明\n", repo.values[service.SettingKeyAPIDocsMarkdown])

	var body response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)

	data, ok := body.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "# 新文档\n\n说明\n", data["effective_content"])
	require.Equal(t, true, data["has_override"])
}

func TestAdminDocsHandlerUpdateAPIReference_RejectsBlankContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewDocsHandler(service.NewAPIDocsService(&adminDocsSettingRepoStub{values: map[string]string{}}))
	ctx, recorder := newAdminDocsContext(
		http.MethodPut,
		"/api/v1/admin/docs/api",
		[]byte(`{"content":"   "}`),
	)

	handler.UpdateAPIReference(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var body response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, http.StatusBadRequest, body.Code)
	require.Equal(t, "API_DOCS_EMPTY", body.Reason)
}

func TestAdminDocsHandlerUpdateAPIReference_RejectsInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewDocsHandler(service.NewAPIDocsService(&adminDocsSettingRepoStub{values: map[string]string{}}))
	ctx, recorder := newAdminDocsContext(
		http.MethodPut,
		"/api/v1/admin/docs/api",
		[]byte(`{"content":`),
	)

	handler.UpdateAPIReference(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAdminDocsHandlerClearAPIReferenceOverride_RestoresDefaultDocument(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAPIDocsMarkdown: "# 覆盖文档\n",
		},
	}
	docsService := service.NewAPIDocsService(repo)
	handler := NewDocsHandler(docsService)

	ctx, recorder := newAdminDocsContext(http.MethodDelete, "/api/v1/admin/docs/api/override", nil)
	handler.ClearAPIReferenceOverride(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	_, exists := repo.values[service.SettingKeyAPIDocsMarkdown]
	require.False(t, exists)

	var body response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)

	data, ok := body.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, docsService.GetDefaultContent(), data["effective_content"])
	require.Equal(t, docsService.GetDefaultContent(), data["default_content"])
	require.Equal(t, false, data["has_override"])
}
