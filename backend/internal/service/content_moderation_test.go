//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type moderationAuditRepoStub struct {
	created  []*ContentModerationAudit
	previous *ContentModerationAudit
}

func (s *moderationAuditRepoStub) CreateContentModerationAudit(ctx context.Context, audit *ContentModerationAudit) error {
	cloned := *audit
	s.created = append(s.created, &cloned)
	return nil
}

func (s *moderationAuditRepoStub) FindRecentContentModerationAuditByHash(ctx context.Context, contentHash string, since time.Time) (*ContentModerationAudit, error) {
	if s.previous == nil {
		return nil, nil
	}
	return s.previous, nil
}

func (s *moderationAuditRepoStub) ListContentModerationAudits(ctx context.Context, filter *ContentModerationAuditFilter) (*ContentModerationAuditList, error) {
	return &ContentModerationAuditList{}, nil
}

func (s *moderationAuditRepoStub) GetContentModerationAuditByID(ctx context.Context, id int64) (*ContentModerationAudit, error) {
	return nil, ErrContentModerationAuditNotFound
}

func TestContentModerationService_RecordAudit_UsesDedupeWithoutUpstreamCall(t *testing.T) {
	repo := &moderationAuditRepoStub{
		previous: &ContentModerationAudit{
			Hit:         true,
			ErrorReason: "cached",
		},
	}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:             "true",
			SettingKeyContentModerationProvider:            "openai",
			SettingKeyContentModerationAPIKey:              "sk-test",
			SettingKeyContentModerationModel:               "omni-moderation-latest",
			SettingKeyContentModerationDedupeWindowSeconds: "300",
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIMessages,
		Content:        "hello world",
	})

	require.Len(t, repo.created, 1)
	require.True(t, repo.created[0].Hit)
	require.True(t, repo.created[0].DedupeHit)
	require.Equal(t, "cached", repo.created[0].ErrorReason)
}

func TestContentModerationService_RecordAudit_CallsOpenAIProvider(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() {
		http.DefaultClient = originalClient
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/moderations", r.URL.Path)
		require.Equal(t, "Bearer sk-live", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "omni-moderation-latest", payload["model"])
		require.Equal(t, "hello world", payload["input"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"flagged":true}]}`))
	}))
	defer server.Close()
	http.DefaultClient = server.Client()

	repo := &moderationAuditRepoStub{}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:   "true",
			SettingKeyContentModerationProvider:  "openai",
			SettingKeyContentModerationBaseURL:   server.URL,
			SettingKeyContentModerationAPIKey:    "sk-live",
			SettingKeyContentModerationModel:     "omni-moderation-latest",
			SettingKeyContentModerationTimeoutMs: "1500",
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIResponses,
		Content:        "hello world",
	})

	require.Len(t, repo.created, 1)
	require.True(t, repo.created[0].Hit)
	require.False(t, repo.created[0].DedupeHit)
	require.Equal(t, "", repo.created[0].ErrorReason)
	require.NotContains(t, repo.created[0].ContentSummary, "hello world")
	require.Contains(t, repo.created[0].ContentSummary, "redacted text")
}

func TestContentModerationService_RecordAudit_StoresProviderErrors(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() {
		http.DefaultClient = originalClient
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":{"message":"invalid moderation key"}}`))
	}))
	defer server.Close()
	http.DefaultClient = server.Client()

	repo := &moderationAuditRepoStub{}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:   "true",
			SettingKeyContentModerationProvider:  "openai",
			SettingKeyContentModerationBaseURL:   server.URL,
			SettingKeyContentModerationAPIKey:    "sk-live",
			SettingKeyContentModerationModel:     "omni-moderation-latest",
			SettingKeyContentModerationTimeoutMs: "1500",
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIChat,
		Content:        "hello world",
	})

	require.Len(t, repo.created, 1)
	require.False(t, repo.created[0].Hit)
	require.Equal(t, "invalid moderation key", repo.created[0].ErrorReason)
	require.NotContains(t, repo.created[0].ContentSummary, "hello world")
	require.Contains(t, repo.created[0].ContentSummary, "redacted text")
}

func TestContentModerationService_RecordAudit_DisabledSkipsPersistence(t *testing.T) {
	repo := &moderationAuditRepoStub{}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled: "false",
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIMessages,
		Content:        "sensitive input",
	})

	require.Empty(t, repo.created)
}

func TestContentModerationService_RecordAudit_UsesRedactedSummaryForDedupe(t *testing.T) {
	repo := &moderationAuditRepoStub{
		previous: &ContentModerationAudit{
			Hit:         true,
			ErrorReason: "cached",
		},
	}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:             "true",
			SettingKeyContentModerationProvider:            "openai",
			SettingKeyContentModerationAPIKey:              "sk-test",
			SettingKeyContentModerationModel:               "omni-moderation-latest",
			SettingKeyContentModerationDedupeWindowSeconds: "300",
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIMessages,
		Content:        "hello world",
	})

	require.Len(t, repo.created, 1)
	require.True(t, repo.created[0].DedupeHit)
	require.NotContains(t, repo.created[0].ContentSummary, "hello world")
	require.Contains(t, repo.created[0].ContentSummary, "redacted text")
}
