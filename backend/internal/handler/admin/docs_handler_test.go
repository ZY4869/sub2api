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

func decodeAdminDocsResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)

	data, ok := body.Data.(map[string]any)
	require.True(t, ok)
	return data
}

func TestAdminDocsHandlerGetAPIReferenceReturnsFullDocumentState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAPIDocsMarkdown: "# Custom Docs\n\n## common\n### Intro\nHello\n",
		},
	}
	handler := NewDocsHandler(service.NewAPIDocsService(repo))

	ctx, recorder := newAdminDocsContext(http.MethodGet, "/api/v1/admin/docs/api", nil)
	handler.GetAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	data := decodeAdminDocsResponse(t, recorder)
	require.Equal(t, "# Custom Docs\n\n## common\n### Intro\nHello\n", data["effective_content"])
	require.NotEmpty(t, data["default_content"])
	require.Equal(t, true, data["has_override"])
}

func TestAdminDocsHandlerGetAPIReferenceSupportsPageID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{values: map[string]string{}}
	serviceInstance := service.NewAPIDocsService(repo)
	require.NoError(t, serviceInstance.SavePageOverride(context.Background(), "document-ai", "# API Docs\n\n## document-ai\n### Runtime\nDocument AI page"))
	handler := NewDocsHandler(serviceInstance)

	ctx, recorder := newAdminDocsContext(http.MethodGet, "/api/v1/admin/docs/api?page_id=document-ai", nil)
	handler.GetAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	data := decodeAdminDocsResponse(t, recorder)
	require.Contains(t, data["effective_content"], "## document-ai")
	require.Contains(t, data["effective_content"], "### Runtime")
	require.Equal(t, true, data["has_override"])
}

func TestAdminDocsHandlerGetAPIReferenceSupportsOpenAINativePageID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{values: map[string]string{}}
	serviceInstance := service.NewAPIDocsService(repo)
	require.NoError(t, serviceInstance.SavePageOverride(context.Background(), "openai-native", "# API Docs\n\n## openai-native\n### Responses\nOpenAI Native page"))
	handler := NewDocsHandler(serviceInstance)

	ctx, recorder := newAdminDocsContext(http.MethodGet, "/api/v1/admin/docs/api?page_id=openai-native", nil)
	handler.GetAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	data := decodeAdminDocsResponse(t, recorder)
	require.Contains(t, data["effective_content"], "## openai-native")
	require.Contains(t, data["effective_content"], "### Responses")
	require.Equal(t, true, data["has_override"])
}

func TestAdminDocsHandlerUpdateAPIReferenceSavesFullOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{values: map[string]string{}}
	handler := NewDocsHandler(service.NewAPIDocsService(repo))

	ctx, recorder := newAdminDocsContext(
		http.MethodPut,
		"/api/v1/admin/docs/api",
		[]byte(`{"content":"# New Docs\n\n## common\n### Intro\nUpdated"}`),
	)

	handler.UpdateAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "# New Docs\n\n## common\n### Intro\nUpdated\n", repo.values[service.SettingKeyAPIDocsMarkdown])

	data := decodeAdminDocsResponse(t, recorder)
	require.Equal(t, "# New Docs\n\n## common\n### Intro\nUpdated\n", data["effective_content"])
	require.Equal(t, true, data["has_override"])
}

func TestAdminDocsHandlerUpdateAPIReferenceSavesPageOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{values: map[string]string{}}
	handler := NewDocsHandler(service.NewAPIDocsService(repo))

	ctx, recorder := newAdminDocsContext(
		http.MethodPut,
		"/api/v1/admin/docs/api?page_id=gemini",
		[]byte(`{"content":"# API Docs\n\n## gemini\n### Runtime\nUpdated gemini"}`),
	)

	handler.UpdateAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, repo.values[service.SettingKeyAPIDocsMarkdown+"_page_gemini"], "## gemini")

	data := decodeAdminDocsResponse(t, recorder)
	require.Contains(t, data["effective_content"], "Updated gemini")
	require.Equal(t, true, data["has_override"])
}

func TestAdminDocsHandlerRejectsBlankContent(t *testing.T) {
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

func TestAdminDocsHandlerRejectsInvalidPageID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewDocsHandler(service.NewAPIDocsService(&adminDocsSettingRepoStub{values: map[string]string{}}))

	getCtx, getRecorder := newAdminDocsContext(http.MethodGet, "/api/v1/admin/docs/api?page_id=invalid", nil)
	handler.GetAPIReference(getCtx)
	require.Equal(t, http.StatusBadRequest, getRecorder.Code)

	putCtx, putRecorder := newAdminDocsContext(
		http.MethodPut,
		"/api/v1/admin/docs/api?page_id=invalid",
		[]byte(`{"content":"# API Docs"}`),
	)
	handler.UpdateAPIReference(putCtx)
	require.Equal(t, http.StatusBadRequest, putRecorder.Code)

	deleteCtx, deleteRecorder := newAdminDocsContext(http.MethodDelete, "/api/v1/admin/docs/api/override?page_id=invalid", nil)
	handler.ClearAPIReferenceOverride(deleteCtx)
	require.Equal(t, http.StatusBadRequest, deleteRecorder.Code)
}

func TestAdminDocsHandlerClearAPIReferenceOverrideRestoresDefaultDocument(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAPIDocsMarkdown: "# Legacy Docs\n",
		},
	}
	serviceInstance := service.NewAPIDocsService(repo)
	handler := NewDocsHandler(serviceInstance)

	ctx, recorder := newAdminDocsContext(http.MethodDelete, "/api/v1/admin/docs/api/override", nil)
	handler.ClearAPIReferenceOverride(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, repo.values)

	data := decodeAdminDocsResponse(t, recorder)
	require.Equal(t, serviceInstance.GetDefaultContent(), data["effective_content"])
	require.Equal(t, serviceInstance.GetDefaultContent(), data["default_content"])
	require.Equal(t, false, data["has_override"])
}

func TestAdminDocsHandlerClearAPIReferenceOverrideRestoresDefaultPageSection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminDocsSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAPIDocsMarkdown: "# Legacy Docs\n\n## gemini\n### Legacy\nLegacy page\n",
		},
	}
	serviceInstance := service.NewAPIDocsService(repo)
	handler := NewDocsHandler(serviceInstance)

	ctx, recorder := newAdminDocsContext(http.MethodDelete, "/api/v1/admin/docs/api/override?page_id=gemini", nil)
	handler.ClearAPIReferenceOverride(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, repo.values[service.SettingKeyAPIDocsMarkdown+"_page_gemini"], "## gemini")

	data := decodeAdminDocsResponse(t, recorder)
	require.Contains(t, data["effective_content"], "## gemini")
	require.Equal(t, false, data["has_override"])
}
