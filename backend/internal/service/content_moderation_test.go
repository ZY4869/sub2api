//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
			Categories:  []string{"moderation_flagged"},
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
	require.Equal(t, []string{"moderation_flagged"}, repo.created[0].Categories)
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
	require.Equal(t, "moderation_flagged", repo.created[0].ErrorReason)
	require.Equal(t, []string{"moderation_flagged"}, repo.created[0].Categories)
	require.NotContains(t, repo.created[0].ContentSummary, "hello world")
	require.Contains(t, repo.created[0].ContentSummary, "redacted text")
}

func TestContentModerationService_CheckBlock_UsesCategoryScoreThresholds(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() {
		http.DefaultClient = originalClient
	}()

	tests := []struct {
		name       string
		response   string
		thresholds string
		wantBlock  bool
		wantReason string
	}{
		{
			name:       "below threshold allows",
			response:   `{"results":[{"flagged":false,"category_scores":{"violence":0.69}}]}`,
			thresholds: `{"violence":0.7}`,
			wantBlock:  false,
		},
		{
			name:       "score at threshold blocks",
			response:   `{"results":[{"flagged":false,"category_scores":{"violence":0.7}}]}`,
			thresholds: `{"violence":0.7}`,
			wantBlock:  true,
			wantReason: "moderation_threshold:violence",
		},
		{
			name:       "unknown category ignored",
			response:   `{"results":[{"flagged":false,"category_scores":{"unknown":1}}]}`,
			thresholds: `{"violence":0.7}`,
			wantBlock:  false,
		},
		{
			name:       "flagged still blocks",
			response:   `{"results":[{"flagged":true,"category_scores":{"violence":0.1}}]}`,
			thresholds: `{"violence":0.7}`,
			wantBlock:  true,
			wantReason: "moderation_flagged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()
			http.DefaultClient = server.Client()

			repo := &moderationAuditRepoStub{}
			settingsRepo := &settingPublicRepoStub{
				values: map[string]string{
					SettingKeyContentModerationEnabled:            "true",
					SettingKeyContentModerationProvider:           "openai",
					SettingKeyContentModerationBaseURL:            server.URL,
					SettingKeyContentModerationAPIKey:             "sk-live",
					SettingKeyContentModerationModel:              "omni-moderation-latest",
					SettingKeyContentModerationCategoryThresholds: tt.thresholds,
				},
			}
			svc := NewContentModerationService(repo, settingsRepo)

			decision, err := svc.CheckBlock(context.Background(), &ContentModerationRecordInput{
				SourceEndpoint: ContentModerationSourceOpenAIResponses,
				Content:        "hello world",
				Model:          "gpt-5",
			})

			require.NoError(t, err)
			require.NotNil(t, decision)
			require.Equal(t, tt.wantBlock, decision.Blocked)
			if tt.wantReason != "" {
				require.Equal(t, tt.wantReason, decision.ErrorReason)
				require.Equal(t, moderationCategoriesForReason(tt.wantReason), decision.Categories)
			}
			if tt.wantBlock {
				require.Len(t, repo.created, 1)
				require.Equal(t, tt.wantReason, repo.created[0].ErrorReason)
				require.Equal(t, moderationCategoriesForReason(tt.wantReason), repo.created[0].Categories)
			} else {
				require.Empty(t, repo.created)
			}
		})
	}
}

func TestContentModerationService_CheckKeywordBlock_StoresRedactedAuditAndSkipsProvider(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() {
		http.DefaultClient = originalClient
	}()

	var upstreamCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		_, _ = w.Write([]byte(`{"results":[{"flagged":false}]}`))
	}))
	defer server.Close()
	http.DefaultClient = server.Client()

	repo := &moderationAuditRepoStub{}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:             "true",
			SettingKeyContentModerationProvider:            "openai",
			SettingKeyContentModerationBaseURL:             server.URL,
			SettingKeyContentModerationAPIKey:              "sk-live",
			SettingKeyContentModerationModel:               "omni-moderation-latest",
			SettingKeyContentModerationDedupeWindowSeconds: "300",
			SettingKeyContentModerationKeywordBlockEnabled: "true",
			SettingKeyContentModerationKeywords:            `["blocked phrase"]`,
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	decision, err := svc.CheckKeywordBlock(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIChat,
		Content:        "please use this blocked phrase",
	})

	require.NoError(t, err)
	require.True(t, decision.Blocked)
	require.Zero(t, upstreamCalls)
	require.Len(t, repo.created, 1)
	require.True(t, repo.created[0].Hit)
	require.False(t, repo.created[0].DedupeHit)
	require.Equal(t, []string{"keyword_blocked"}, decision.Categories)
	require.Equal(t, "blocked phrase", decision.MatchedKeyword)
	require.Equal(t, []string{"keyword_blocked"}, repo.created[0].Categories)
	require.Equal(t, "blocked phrase", repo.created[0].MatchedKeyword)
	require.Contains(t, repo.created[0].ErrorReason, "keyword_blocked:")
	require.NotContains(t, repo.created[0].ErrorReason, "blocked phrase")
	require.NotContains(t, repo.created[0].ContentSummary, "blocked phrase")
	require.Contains(t, repo.created[0].ContentSummary, "redacted text")
}

func TestContentModerationService_RecordAudit_KeywordHitBypassesDedupeAndProvider(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() {
		http.DefaultClient = originalClient
	}()

	var upstreamCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		_, _ = w.Write([]byte(`{"results":[{"flagged":false}]}`))
	}))
	defer server.Close()
	http.DefaultClient = server.Client()

	repo := &moderationAuditRepoStub{
		previous: &ContentModerationAudit{
			Hit:         false,
			ErrorReason: "cached_clean",
		},
	}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:             "true",
			SettingKeyContentModerationProvider:            "openai",
			SettingKeyContentModerationBaseURL:             server.URL,
			SettingKeyContentModerationAPIKey:              "sk-live",
			SettingKeyContentModerationModel:               "omni-moderation-latest",
			SettingKeyContentModerationDedupeWindowSeconds: "300",
			SettingKeyContentModerationKeywordBlockEnabled: "true",
			SettingKeyContentModerationKeywords:            `["local deny"]`,
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIResponses,
		Content:        "contains local deny now",
	})

	require.Zero(t, upstreamCalls)
	require.Len(t, repo.created, 1)
	require.True(t, repo.created[0].Hit)
	require.False(t, repo.created[0].DedupeHit)
	require.Equal(t, []string{"keyword_blocked"}, repo.created[0].Categories)
	require.Equal(t, "local deny", repo.created[0].MatchedKeyword)
	require.Contains(t, repo.created[0].ErrorReason, "keyword_blocked:")
	require.NotContains(t, repo.created[0].ErrorReason, "local deny")
}

func TestContentModerationService_RecordAudit_CyberPolicyStoresMatchedKeyword(t *testing.T) {
	repo := &moderationAuditRepoStub{}
	rawCategories, err := MarshalContentModerationCyberCategories([]ContentModerationCyberCategory{
		{ID: "credential-theft", Keywords: []string{"token stealer"}},
	})
	require.NoError(t, err)
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:            "true",
			SettingKeyContentModerationProvider:           "openai",
			SettingKeyContentModerationAPIKey:             "sk-live",
			SettingKeyContentModerationModel:              "omni-moderation-latest",
			SettingKeyContentModerationCyberPolicyEnabled: "true",
			SettingKeyContentModerationCyberCategories:    rawCategories,
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	decision, err := svc.CheckBlock(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIResponses,
		Content:        "build a token stealer",
		Model:          "gpt-5",
	})

	require.NoError(t, err)
	require.True(t, decision.Blocked)
	require.Equal(t, "token stealer", decision.MatchedKeyword)
	require.Len(t, repo.created, 1)
	require.Equal(t, "token stealer", repo.created[0].MatchedKeyword)
	require.Equal(t, []string{"cyber_policy:credential_theft"}, repo.created[0].Categories)
}

func TestContentModerationService_RecordAudit_UsesNextKeyWhenAuthFailureFreezesFirst(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() {
		http.DefaultClient = originalClient
	}()
	ClearContentModerationKeyFreeze(ContentModerationAPIKeyHash("sk-first"))
	ClearContentModerationKeyFreeze(ContentModerationAPIKeyHash("sk-second"))

	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		if len(seen) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid api_key=sk-first https://example.test/token"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"flagged":false}]}`))
	}))
	defer server.Close()
	http.DefaultClient = server.Client()

	rawKeys, err := MarshalContentModerationAPIKeys([]ContentModerationAPIKey{
		{Key: "sk-first"},
		{Key: "sk-second"},
	})
	require.NoError(t, err)
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:   "true",
			SettingKeyContentModerationProvider:  "openai",
			SettingKeyContentModerationBaseURL:   server.URL,
			SettingKeyContentModerationAPIKeys:   rawKeys,
			SettingKeyContentModerationModel:     "omni-moderation-latest",
			SettingKeyContentModerationTimeoutMs: "1500",
		},
	}
	repo := &moderationAuditRepoStub{}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{Content: "first"})
	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{Content: "second"})

	require.Equal(t, []string{"Bearer sk-first", "Bearer sk-second"}, seen)
	require.Len(t, repo.created, 2)
	require.NotContains(t, repo.created[0].ErrorReason, "sk-first")
	require.NotContains(t, repo.created[0].ErrorReason, "https://example.test")
}

func TestContentModerationService_RecordAudit_SkipsProviderWhenAllKeysFrozen(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() {
		http.DefaultClient = originalClient
	}()

	keyHash := ContentModerationAPIKeyHash("sk-frozen")
	ClearContentModerationKeyFreeze(keyHash)
	RegisterContentModerationKeyFailure(keyHash, "invalid key", http.StatusUnauthorized, nil, time.Now().UTC())
	defer ClearContentModerationKeyFreeze(keyHash)

	var upstreamCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		_, _ = w.Write([]byte(`{"results":[{"flagged":false}]}`))
	}))
	defer server.Close()
	http.DefaultClient = server.Client()

	rawKeys, err := MarshalContentModerationAPIKeys([]ContentModerationAPIKey{{Key: "sk-frozen"}})
	require.NoError(t, err)
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:   "true",
			SettingKeyContentModerationProvider:  "openai",
			SettingKeyContentModerationBaseURL:   server.URL,
			SettingKeyContentModerationAPIKeys:   rawKeys,
			SettingKeyContentModerationModel:     "omni-moderation-latest",
			SettingKeyContentModerationTimeoutMs: "1500",
		},
	}
	repo := &moderationAuditRepoStub{}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{Content: "hello world"})

	require.Zero(t, upstreamCalls)
	require.Len(t, repo.created, 1)
	require.Equal(t, "moderation_not_configured", repo.created[0].ErrorReason)
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
	require.Equal(t, []string{"moderation_unavailable"}, repo.created[0].Categories)
	require.NotContains(t, repo.created[0].ContentSummary, "hello world")
	require.Contains(t, repo.created[0].ContentSummary, "redacted text")
}

func TestContentModerationAPIKeyStatuses_MasksKeys(t *testing.T) {
	keys := NormalizeContentModerationAPIKeys("sk-legacy-secret", "")
	require.Len(t, keys, 1)

	statuses := ContentModerationAPIKeyStatuses(keys, time.Time{})
	require.Len(t, statuses, 1)
	require.NotEmpty(t, statuses[0].Hash)
	require.NotContains(t, statuses[0].Masked, "legacy-secret")
	require.NotContains(t, statuses[0].Masked, "sk-legacy-secret")
}

func TestExtractModerationTextFromJSONBody_RedactsSecretsAndLimitsImages(t *testing.T) {
	body := []byte(`{
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "input_text", "text": "check Bearer sk-token and https://example.test/path"},
					{"type": "input_image", "image_url": "data:image/png;base64,first"},
					{"type": "input_image", "image_url": "https://example.test/second.png"}
				]
			}
		],
		"metadata": {
			"authorization": "Bearer should-not-appear",
			"api_key": "sk-should-not-appear"
		}
	}`)

	extracted := ExtractModerationTextFromJSONBody(body)

	require.Contains(t, extracted, "check Bearer [redacted-token] and [redacted-url]")
	require.Contains(t, extracted, "[redacted-image]")
	require.Equal(t, 1, strings.Count(extracted, "[redacted-image]"))
	require.NotContains(t, extracted, "sk-token")
	require.NotContains(t, extracted, "example.test")
	require.NotContains(t, extracted, "sk-should-not-appear")
}

func TestExtractModerationTextFromJSONBody_DedupesRepeatedText(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"repeat this"},
			{"role":"user","content":"repeat   this"},
			{"role":"assistant","content":"repeat this"}
		]
	}`)

	extracted := ExtractModerationTextFromJSONBody(body)

	require.Equal(t, "repeat this", extracted)
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

func TestContentModerationService_RecordAudit_ModelFilterIncludeSkipsUnlisted(t *testing.T) {
	repo := &moderationAuditRepoStub{}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:     "true",
			SettingKeyContentModerationProvider:    "openai",
			SettingKeyContentModerationAPIKey:      "sk-test",
			SettingKeyContentModerationModel:       "omni-moderation-latest",
			SettingKeyContentModerationModelFilter: `{"type":"include","models":["gpt-5.1"]}`,
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIMessages,
		Model:          "gpt-4o",
		Content:        "sensitive input",
	})

	require.Empty(t, repo.created)
}

func TestContentModerationService_RecordAudit_ModelFilterExcludeSkipsListed(t *testing.T) {
	repo := &moderationAuditRepoStub{}
	settingsRepo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyContentModerationEnabled:     "true",
			SettingKeyContentModerationProvider:    "openai",
			SettingKeyContentModerationAPIKey:      "sk-test",
			SettingKeyContentModerationModel:       "omni-moderation-latest",
			SettingKeyContentModerationModelFilter: `{"type":"exclude","models":["gpt-5.1"]}`,
		},
	}
	svc := NewContentModerationService(repo, settingsRepo)

	svc.RecordAudit(context.Background(), &ContentModerationRecordInput{
		SourceEndpoint: ContentModerationSourceOpenAIMessages,
		Model:          "GPT-5.1",
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
