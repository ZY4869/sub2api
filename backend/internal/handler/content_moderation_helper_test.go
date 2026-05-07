//go:build unit

package handler

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type moderationHelperRepoStub struct {
	created []*service.ContentModerationAudit
}

func (s *moderationHelperRepoStub) CreateContentModerationAudit(ctx context.Context, audit *service.ContentModerationAudit) error {
	clone := *audit
	s.created = append(s.created, &clone)
	return nil
}

func (s *moderationHelperRepoStub) FindRecentContentModerationAuditByHash(ctx context.Context, contentHash string, since time.Time) (*service.ContentModerationAudit, error) {
	return nil, nil
}

func (s *moderationHelperRepoStub) ListContentModerationAudits(ctx context.Context, filter *service.ContentModerationAuditFilter) (*service.ContentModerationAuditList, error) {
	return &service.ContentModerationAuditList{}, nil
}

func (s *moderationHelperRepoStub) GetContentModerationAuditByID(ctx context.Context, id int64) (*service.ContentModerationAudit, error) {
	return nil, service.ErrContentModerationAuditNotFound
}

func TestBuildContentModerationRecordInput_MapsSubjectAndAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42, Concurrency: 3})
	c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{ID: 99})

	record := buildContentModerationRecordInput(
		c,
		service.ContentModerationSourceOpenAIMessages,
		service.PlatformOpenAI,
		"gpt-5.1",
		[]byte(`{"input":[{"type":"input_text","text":"hello moderation"}]}`),
	)

	require.NotNil(t, record)
	require.Equal(t, service.ContentModerationSourceOpenAIMessages, record.SourceEndpoint)
	require.Equal(t, service.PlatformOpenAI, record.Provider)
	require.Equal(t, "gpt-5.1", record.Model)
	require.NotNil(t, record.UserID)
	require.Equal(t, int64(42), *record.UserID)
	require.NotNil(t, record.APIKeyID)
	require.Equal(t, int64(99), *record.APIKeyID)
}

func TestSubmitContentModerationAudit_SkipsEmptyExtractedContent(t *testing.T) {
	settingsRepo := &socialOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyContentModerationEnabled:             "true",
		service.SettingKeyContentModerationProvider:            "openai",
		service.SettingKeyContentModerationAPIKey:              "sk-test",
		service.SettingKeyContentModerationModel:               "omni-moderation-latest",
		service.SettingKeyContentModerationDedupeWindowSeconds: "0",
	}}
	moderationRepo := &moderationHelperRepoStub{}
	moderationService := service.NewContentModerationService(
		moderationRepo,
		settingsRepo,
	)

	submitContentModerationAudit(
		context.Background(),
		moderationService,
		&service.ContentModerationRecordInput{
			SourceEndpoint: service.ContentModerationSourceOpenAIResponses,
			Provider:       service.PlatformOpenAI,
			Model:          "gpt-5.1",
			Content:        `{"metadata":{"note":"no prompt fields here"}}`,
		},
	)

	time.Sleep(50 * time.Millisecond)
	require.Empty(t, moderationRepo.created)
}
