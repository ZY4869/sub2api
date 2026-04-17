package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type docsHandlerSettingRepoStub struct {
	values      map[string]string
	getValueErr error
}

func (s *docsHandlerSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *docsHandlerSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.getValueErr != nil {
		return "", s.getValueErr
	}
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *docsHandlerSettingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *docsHandlerSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *docsHandlerSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *docsHandlerSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *docsHandlerSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestDocsHandlerGetAPIReference_ReturnsEffectiveContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &docsHandlerSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAPIDocsMarkdown: "# 自定义 API 文档\n",
		},
	}
	docsService := service.NewAPIDocsService(repo)
	handler := NewDocsHandler(docsService)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/docs/api", nil)

	handler.GetAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var body response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)

	data, ok := body.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "# 自定义 API 文档\n", data["content"])
}

func TestDocsHandlerGetAPIReference_FallsBackToDefaultTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &docsHandlerSettingRepoStub{values: map[string]string{}}
	docsService := service.NewAPIDocsService(repo)
	handler := NewDocsHandler(docsService)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/docs/api", nil)

	handler.GetAPIReference(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var body response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)

	data, ok := body.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, docsService.GetDefaultContent(), data["content"])
}

func TestDocsHandlerGetAPIReference_ReturnsInternalErrorWhenLookupFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &docsHandlerSettingRepoStub{
		values:      map[string]string{},
		getValueErr: errors.New("boom"),
	}
	handler := NewDocsHandler(service.NewAPIDocsService(repo))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/docs/api", nil)

	handler.GetAPIReference(ctx)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}
