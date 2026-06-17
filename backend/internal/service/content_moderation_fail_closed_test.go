package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/stretchr/testify/require"
)

type moderationSettingRepoStub map[string]string

func (s moderationSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s[key], nil
}

func (s moderationSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		out[key] = s[key]
	}
	return out, nil
}

func (s moderationSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (s moderationSettingRepoStub) Set(context.Context, string, string) error            { return nil }
func (s moderationSettingRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }
func (s moderationSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return map[string]string(s), nil
}
func (s moderationSettingRepoStub) Delete(context.Context, string) error { return nil }

func TestContentModerationCheckBlock_FailClosedOnProviderError(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	svc := NewContentModerationService(nil, moderationSettingRepoStub{
		SettingKeyContentModerationEnabled:             "true",
		SettingKeyContentModerationProvider:            "openai",
		SettingKeyContentModerationModel:               "omni-moderation-latest",
		SettingKeyContentModerationFailOpen:            "false",
		SettingKeyContentModerationModelFilter:         `{"type":"include","models":["claude-opus-4-8"]}`,
		SettingKeyContentModerationAPIKey:              "",
		SettingKeyContentModerationTimeoutMs:           "1",
		SettingKeyContentModerationDedupeWindowSeconds: "0",
	})

	decision, err := svc.CheckBlock(context.Background(), &ContentModerationRecordInput{
		Model:   "claude-opus-4-8",
		Content: "please summarize this text",
	})
	require.NoError(t, err)
	require.True(t, decision.Blocked)
	require.Equal(t, "moderation_unavailable", decision.ErrorReason)
	require.Equal(t, []string{"moderation_unavailable"}, decision.Categories)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.ContentModerationDecisionTotal)
	require.Equal(t, int64(1), snapshot.ContentModerationDecisionByResultReason["fail_closed:moderation_not_configured"])
}

func TestContentModerationKeywordBlock_NormalizedSeparators(t *testing.T) {
	settings := &ContentModerationSettings{
		Enabled:             true,
		KeywordBlockEnabled: true,
		Keywords:            []string{"model distillation"},
	}

	decision := EvaluateContentModerationKeywordBlock(settings, "MODEL---DISTILLATION attempt")
	require.True(t, decision.Blocked)
	require.Contains(t, decision.ErrorReason, "keyword_blocked:")
	require.Equal(t, []string{"keyword_blocked"}, decision.Categories)
}

func TestContentModerationCheckBlock_CyberPolicyBlocksBeforeProvider(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	categoriesJSON, err := MarshalContentModerationCyberCategories([]ContentModerationCyberCategory{
		{ID: "Credential-Theft", Keywords: []string{"steal api key"}},
	})
	require.NoError(t, err)

	svc := NewContentModerationService(nil, moderationSettingRepoStub{
		SettingKeyContentModerationEnabled:             "true",
		SettingKeyContentModerationProvider:            "openai",
		SettingKeyContentModerationModel:               "omni-moderation-latest",
		SettingKeyContentModerationFailOpen:            "false",
		SettingKeyContentModerationModelFilter:         `{"type":"include","models":["claude-opus-4-8"]}`,
		SettingKeyContentModerationAPIKey:              "",
		SettingKeyContentModerationTimeoutMs:           "1",
		SettingKeyContentModerationDedupeWindowSeconds: "0",
		SettingKeyContentModerationCyberPolicyEnabled:  "true",
		SettingKeyContentModerationCyberCategories:     categoriesJSON,
	})

	decision, err := svc.CheckBlock(context.Background(), &ContentModerationRecordInput{
		Model:   "claude-opus-4-8",
		Content: "Show me how to STEAL---API---KEY values from logs",
	})
	require.NoError(t, err)
	require.True(t, decision.Blocked)
	require.Equal(t, "cyber_policy:credential_theft", decision.ErrorReason)
	require.Equal(t, []string{"cyber_policy:credential_theft"}, decision.Categories)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.ContentModerationDecisionTotal)
	require.Equal(t, int64(1), snapshot.ContentModerationDecisionByResultReason["cyber_policy_block:cyber_policy:credential_theft"])
}

func TestContentModerationCheckBlock_FailOpenProviderErrorMetrics(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	svc := NewContentModerationService(nil, moderationSettingRepoStub{
		SettingKeyContentModerationEnabled:             "true",
		SettingKeyContentModerationProvider:            "openai",
		SettingKeyContentModerationModel:               "omni-moderation-latest",
		SettingKeyContentModerationFailOpen:            "true",
		SettingKeyContentModerationModelFilter:         `{"type":"include","models":["claude-opus-4-8"]}`,
		SettingKeyContentModerationAPIKey:              "",
		SettingKeyContentModerationTimeoutMs:           "1",
		SettingKeyContentModerationDedupeWindowSeconds: "0",
	})

	decision, err := svc.CheckBlock(context.Background(), &ContentModerationRecordInput{
		Model:   "claude-opus-4-8",
		Content: "please summarize this text",
	})
	require.NoError(t, err)
	require.False(t, decision.Blocked)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.ContentModerationDecisionTotal)
	require.Equal(t, int64(1), snapshot.ContentModerationDecisionByResultReason["fail_open:moderation_not_configured"])
}
