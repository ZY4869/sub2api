package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminSettingRepoStub struct {
	values map[string]string
}

func (s *adminSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *adminSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *adminSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *adminSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *adminSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *adminSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *adminSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func newAdminSettingTestHandler(repo *adminSettingRepoStub) *SettingHandler {
	settingSvc := service.NewSettingService(repo, &config.Config{})
	return NewSettingHandler(settingSvc, nil, nil, nil, nil)
}

func performAdminSettingsUpdate(t *testing.T, handler *SettingHandler, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1, Concurrency: 1})
	c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)

	handler.UpdateSettings(c)
	return recorder
}

func decodeUpdatedSystemSettings(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var payload struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
	return payload.Data
}

func TestSettingHandlerUpdateSettings_ContentModerationAPIKeysAppendReplaceAndDelete(t *testing.T) {
	rawKeys, err := service.MarshalContentModerationAPIKeys([]service.ContentModerationAPIKey{
		{Key: "sk-first"},
		{Key: "sk-second"},
	})
	require.NoError(t, err)

	repo := &adminSettingRepoStub{
		values: map[string]string{
			service.SettingKeyContentModerationAPIKey:  "sk-first",
			service.SettingKeyContentModerationAPIKeys: rawKeys,
		},
	}
	handler := newAdminSettingTestHandler(repo)

	appendResp := performAdminSettingsUpdate(t, handler, `{
		"content_moderation_api_keys":["sk-third"],
		"content_moderation_api_keys_mode":"append"
	}`)
	require.Equal(t, http.StatusOK, appendResp.Code)

	settings := decodeUpdatedSystemSettings(t, appendResp)
	statuses, ok := settings["content_moderation_api_key_statuses"].([]any)
	require.True(t, ok)
	require.Len(t, statuses, 3)

	currentKeys := service.NormalizeContentModerationAPIKeys(
		repo.values[service.SettingKeyContentModerationAPIKey],
		repo.values[service.SettingKeyContentModerationAPIKeys],
	)
	require.Len(t, currentKeys, 3)
	require.Equal(t, "sk-first", currentKeys[0].Key)
	require.Equal(t, "sk-second", currentKeys[1].Key)
	require.Equal(t, "sk-third", currentKeys[2].Key)

	deleteHash := currentKeys[1].Hash
	deleteResp := performAdminSettingsUpdate(t, handler, `{
		"delete_content_moderation_api_key_hashes":["`+deleteHash+`"]
	}`)
	require.Equal(t, http.StatusOK, deleteResp.Code)

	currentKeys = service.NormalizeContentModerationAPIKeys(
		repo.values[service.SettingKeyContentModerationAPIKey],
		repo.values[service.SettingKeyContentModerationAPIKeys],
	)
	require.Len(t, currentKeys, 2)
	require.Equal(t, "sk-first", currentKeys[0].Key)
	require.Equal(t, "sk-third", currentKeys[1].Key)

	replaceResp := performAdminSettingsUpdate(t, handler, `{
		"content_moderation_api_keys":["sk-replace"],
		"content_moderation_api_keys_mode":"replace"
	}`)
	require.Equal(t, http.StatusOK, replaceResp.Code)

	currentKeys = service.NormalizeContentModerationAPIKeys(
		repo.values[service.SettingKeyContentModerationAPIKey],
		repo.values[service.SettingKeyContentModerationAPIKeys],
	)
	require.Len(t, currentKeys, 1)
	require.Equal(t, "sk-replace", currentKeys[0].Key)
	require.Equal(t, "sk-replace", repo.values[service.SettingKeyContentModerationAPIKey])
}

func TestSettingHandlerUpdateSettings_ContentModerationLegacyKeyAndNoMutationPreserveExistingList(t *testing.T) {
	rawKeys, err := service.MarshalContentModerationAPIKeys([]service.ContentModerationAPIKey{
		{Key: "sk-first"},
		{Key: "sk-second"},
	})
	require.NoError(t, err)

	repo := &adminSettingRepoStub{
		values: map[string]string{
			service.SettingKeySiteName:                  "Before",
			service.SettingKeyContentModerationAPIKey:   "sk-first",
			service.SettingKeyContentModerationAPIKeys:  rawKeys,
			service.SettingKeyContentModerationProvider: "openai",
		},
	}
	handler := newAdminSettingTestHandler(repo)

	noMutationResp := performAdminSettingsUpdate(t, handler, `{"site_name":"After"}`)
	require.Equal(t, http.StatusOK, noMutationResp.Code)

	currentKeys := service.NormalizeContentModerationAPIKeys(
		repo.values[service.SettingKeyContentModerationAPIKey],
		repo.values[service.SettingKeyContentModerationAPIKeys],
	)
	require.Len(t, currentKeys, 2)
	require.Equal(t, "sk-first", currentKeys[0].Key)
	require.Equal(t, "sk-second", currentKeys[1].Key)

	legacyResp := performAdminSettingsUpdate(t, handler, `{
		"content_moderation_api_key":"sk-legacy-new"
	}`)
	require.Equal(t, http.StatusOK, legacyResp.Code)

	currentKeys = service.NormalizeContentModerationAPIKeys(
		repo.values[service.SettingKeyContentModerationAPIKey],
		repo.values[service.SettingKeyContentModerationAPIKeys],
	)
	require.Len(t, currentKeys, 3)
	require.Equal(t, "sk-first", currentKeys[0].Key)
	require.Equal(t, "sk-second", currentKeys[1].Key)
	require.Equal(t, "sk-legacy-new", currentKeys[2].Key)
}

func TestSettingHandlerUpdateSettings_LoginAgreementRequiresPublishedMarkdownPages(t *testing.T) {
	repo := &adminSettingRepoStub{
		values: map[string]string{
			service.SettingKeyCustomMenuItems: `[{"id":"terms","label":"Terms","icon_svg":"","url":"","visibility":"user","sort_order":0,"page_mode":"markdown","page_slug":"terms","page_content":"# Terms","page_published":true}]`,
		},
	}
	handler := newAdminSettingTestHandler(repo)

	noDocsResp := performAdminSettingsUpdate(t, handler, `{
		"login_agreement_enabled":true,
		"login_agreement_documents":[]
	}`)
	require.Equal(t, http.StatusBadRequest, noDocsResp.Code)
	require.Contains(t, noDocsResp.Body.String(), "Login agreement requires at least one published markdown page")

	unpublishedResp := performAdminSettingsUpdate(t, handler, `{
		"login_agreement_enabled":true,
		"login_agreement_documents":[{"id":"privacy","title":"Privacy","page_slug":"privacy"}]
	}`)
	require.Equal(t, http.StatusBadRequest, unpublishedResp.Code)
	require.Contains(t, unpublishedResp.Body.String(), "Login agreement document must reference a published markdown page")

	successResp := performAdminSettingsUpdate(t, handler, `{
		"login_agreement_enabled":true,
		"login_agreement_mode":"checkbox",
		"login_agreement_updated_at":"2026-05-08",
		"login_agreement_documents":[{"id":"terms","title":"Terms","page_slug":"terms"}]
	}`)
	require.Equal(t, http.StatusOK, successResp.Code)

	settings := decodeUpdatedSystemSettings(t, successResp)
	require.Equal(t, true, settings["login_agreement_enabled"])
	require.Equal(t, "checkbox", settings["login_agreement_mode"])
	require.Equal(t, "2026-05-08", settings["login_agreement_updated_at"])
}

func TestSettingHandlerUpdateSettings_AccountAiryWhiteSurfaceEnabled(t *testing.T) {
	repo := &adminSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAccountAiryWhiteSurfaceEnabled: "false",
		},
	}
	handler := newAdminSettingTestHandler(repo)

	resp := performAdminSettingsUpdate(t, handler, `{
		"account_airy_white_surface_enabled": true
	}`)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyAccountAiryWhiteSurfaceEnabled])

	settings := decodeUpdatedSystemSettings(t, resp)
	require.Equal(t, true, settings["account_airy_white_surface_enabled"])
}
