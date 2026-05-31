package service

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/stretchr/testify/require"
)

func TestRecordContentModerationRepeatedPromptSignal_DetectsSameAndSimilarWithinSubject(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	resetContentModerationPromptSignalsForTest()
	t.Cleanup(resetContentModerationPromptSignalsForTest)

	userID := int64(7)
	apiKeyID := int64(11)
	input := &ContentModerationRecordInput{
		UserID:         &userID,
		APIKeyID:       &apiKeyID,
		Model:          "GPT-5.1",
		SourceEndpoint: ContentModerationSourceOpenAIChat,
	}
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	prompt := "Summarize the attached invoice and extract vendor date amount payment terms currency due status reference number"
	nearPrompt := "Summarize the attached invoice and extract vendor date amount payment terms currency due status reference id"

	RecordContentModerationRepeatedPromptSignal(input, prompt, now)
	RecordContentModerationRepeatedPromptSignal(input, prompt, now.Add(time.Minute))
	RecordContentModerationRepeatedPromptSignal(input, nearPrompt, now.Add(2*time.Minute))

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(2), snapshot.AbuseSignalByType[contentModerationRepeatedPromptSignalType])
}

func TestRecordContentModerationRepeatedPromptSignal_IsolatedByUserKeyAndModel(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	resetContentModerationPromptSignalsForTest()
	t.Cleanup(resetContentModerationPromptSignalsForTest)

	userID := int64(7)
	otherUserID := int64(8)
	apiKeyID := int64(11)
	otherAPIKeyID := int64(12)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	prompt := "Draft a compact customer support reply about refund status and order lookup"

	RecordContentModerationRepeatedPromptSignal(&ContentModerationRecordInput{
		UserID:   &userID,
		APIKeyID: &apiKeyID,
		Model:    "gpt-5.1",
	}, prompt, now)
	RecordContentModerationRepeatedPromptSignal(&ContentModerationRecordInput{
		UserID:   &otherUserID,
		APIKeyID: &apiKeyID,
		Model:    "gpt-5.1",
	}, prompt, now.Add(time.Minute))
	RecordContentModerationRepeatedPromptSignal(&ContentModerationRecordInput{
		UserID:   &userID,
		APIKeyID: &otherAPIKeyID,
		Model:    "gpt-5.1",
	}, prompt, now.Add(2*time.Minute))
	RecordContentModerationRepeatedPromptSignal(&ContentModerationRecordInput{
		UserID:   &userID,
		APIKeyID: &apiKeyID,
		Model:    "claude-opus-4-8",
	}, prompt, now.Add(3*time.Minute))

	snapshot := protocolruntime.Snapshot()
	require.Zero(t, snapshot.AbuseSignalByType[contentModerationRepeatedPromptSignalType])
}

func TestRecordContentModerationRepeatedPromptSignal_ExpiresByTTL(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	resetContentModerationPromptSignalsForTest()
	t.Cleanup(resetContentModerationPromptSignalsForTest)

	userID := int64(7)
	apiKeyID := int64(11)
	input := &ContentModerationRecordInput{
		UserID:   &userID,
		APIKeyID: &apiKeyID,
		Model:    "gpt-5.1",
	}
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	prompt := "Explain how to reconcile account usage billing records for the last day"

	RecordContentModerationRepeatedPromptSignal(input, prompt, now)
	RecordContentModerationRepeatedPromptSignal(input, prompt, now.Add(contentModerationPromptSignalTTL+time.Minute))

	snapshot := protocolruntime.Snapshot()
	require.Zero(t, snapshot.AbuseSignalByType[contentModerationRepeatedPromptSignalType])
}
