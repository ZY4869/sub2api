package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBlacklistAdviceRecommendsBlacklistForTopLevelUnauthorizedDetail(t *testing.T) {
	t.Parallel()

	advice := BuildBlacklistAdvice(nil, 401, []byte(`{"detail":"Unauthorized"}`))
	require.NotNil(t, advice)
	if advice.Decision != BlacklistAdviceRecommendBlacklist {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceRecommendBlacklist)
	}
	if advice.ReasonCode != "credentials_need_reauth" {
		t.Fatalf("ReasonCode = %q, want %q", advice.ReasonCode, "credentials_need_reauth")
	}
	if advice.ReasonMessage != "Unauthorized" {
		t.Fatalf("ReasonMessage = %q, want %q", advice.ReasonMessage, "Unauthorized")
	}
}

func TestBuildBlacklistAdviceRecommendsBlacklistForNestedUnauthorizedMessage(t *testing.T) {
	t.Parallel()

	advice := BuildBlacklistAdvice(nil, 401, []byte(`{"error":{"message":"Unauthorized"}}`))
	require.NotNil(t, advice)
	if advice.Decision != BlacklistAdviceRecommendBlacklist {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceRecommendBlacklist)
	}
	if advice.ReasonMessage != "Unauthorized" {
		t.Fatalf("ReasonMessage = %q, want %q", advice.ReasonMessage, "Unauthorized")
	}
}

func TestBuildBlacklistAdviceRecommendsBlacklistForPlainUnauthorizedText(t *testing.T) {
	t.Parallel()

	advice := BuildBlacklistAdvice(nil, 401, []byte(`unauthorized`))
	require.NotNil(t, advice)
	if advice.Decision != BlacklistAdviceRecommendBlacklist {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceRecommendBlacklist)
	}
	if advice.ReasonMessage != "unauthorized" {
		t.Fatalf("ReasonMessage = %q, want %q", advice.ReasonMessage, "unauthorized")
	}
}

func TestBuildBlacklistAdviceRecommendsBlacklistForFailoverWrappedUnauthorizedText(t *testing.T) {
	t.Parallel()

	advice := BuildBlacklistAdvice(nil, 401, []byte(`upstream error: 401 (failover) unauthorized`))
	require.NotNil(t, advice)
	if advice.Decision != BlacklistAdviceRecommendBlacklist {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceRecommendBlacklist)
	}
	if advice.ReasonCode != "credentials_need_reauth" {
		t.Fatalf("ReasonCode = %q, want %q", advice.ReasonCode, "credentials_need_reauth")
	}
}

func TestBuildBlacklistAdviceKeepsRateLimitFailuresNotRecommended(t *testing.T) {
	t.Parallel()

	advice := BuildBlacklistAdvice(nil, 429, []byte(`{"error":{"message":"Rate limited. Please try again later."}}`))
	require.NotNil(t, advice)
	if advice.Decision != BlacklistAdviceNotRecommended {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceNotRecommended)
	}
	if advice.ReasonCode != "transient_or_retryable" {
		t.Fatalf("ReasonCode = %q, want %q", advice.ReasonCode, "transient_or_retryable")
	}
}

func TestDetectHardBannedAccountMatchesDeactivatedDetailMessage(t *testing.T) {
	t.Parallel()

	match := DetectHardBannedAccount(401, []byte(`{"detail":"Your OpenAI account has been deactivated. Please contact help.openai.com."}`))
	require.NotNil(t, match)
	if match.ReasonCode != "account_deactivated" {
		t.Fatalf("ReasonCode = %q, want %q", match.ReasonCode, "account_deactivated")
	}
}

func TestBuildBlacklistAdviceDoesNotAutoBlacklist503HelpPageErrors(t *testing.T) {
	t.Parallel()

	body := []byte(`{"error":{"message":"Your OpenAI account has been deactivated. Please contact help.openai.com."}}`)
	match := DetectHardBannedAccount(503, body)
	require.Nil(t, match)

	advice := BuildBlacklistAdvice(nil, 503, body)
	require.NotNil(t, advice)
	if advice.Decision != BlacklistAdviceNotRecommended {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceNotRecommended)
	}
	if advice.ReasonCode != "transient_or_retryable" {
		t.Fatalf("ReasonCode = %q, want %q", advice.ReasonCode, "transient_or_retryable")
	}
}
