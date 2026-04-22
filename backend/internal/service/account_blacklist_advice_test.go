package service

import "testing"

func TestBuildBlacklistAdviceRecommendsBlacklistForTopLevelUnauthorizedDetail(t *testing.T) {
	t.Parallel()

	advice := BuildBlacklistAdvice(nil, 401, []byte(`{"detail":"Unauthorized"}`))
	if advice == nil {
		t.Fatal("expected blacklist advice, got nil")
	}
	if advice.Decision != BlacklistAdviceRecommendBlacklist {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceRecommendBlacklist)
	}
	if advice.ReasonCode != "credentials_likely_invalid" {
		t.Fatalf("ReasonCode = %q, want %q", advice.ReasonCode, "credentials_likely_invalid")
	}
	if advice.ReasonMessage != "Unauthorized" {
		t.Fatalf("ReasonMessage = %q, want %q", advice.ReasonMessage, "Unauthorized")
	}
}

func TestBuildBlacklistAdviceRecommendsBlacklistForNestedUnauthorizedMessage(t *testing.T) {
	t.Parallel()

	advice := BuildBlacklistAdvice(nil, 401, []byte(`{"error":{"message":"Unauthorized"}}`))
	if advice == nil {
		t.Fatal("expected blacklist advice, got nil")
	}
	if advice.Decision != BlacklistAdviceRecommendBlacklist {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceRecommendBlacklist)
	}
	if advice.ReasonMessage != "Unauthorized" {
		t.Fatalf("ReasonMessage = %q, want %q", advice.ReasonMessage, "Unauthorized")
	}
}

func TestBuildBlacklistAdviceKeepsRateLimitFailuresNotRecommended(t *testing.T) {
	t.Parallel()

	advice := BuildBlacklistAdvice(nil, 429, []byte(`{"error":{"message":"Rate limited. Please try again later."}}`))
	if advice == nil {
		t.Fatal("expected blacklist advice, got nil")
	}
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
	if match == nil {
		t.Fatal("expected hard ban match, got nil")
	}
	if match.ReasonCode != "account_deactivated" {
		t.Fatalf("ReasonCode = %q, want %q", match.ReasonCode, "account_deactivated")
	}
}

func TestBuildBlacklistAdviceDoesNotAutoBlacklist503HelpPageErrors(t *testing.T) {
	t.Parallel()

	body := []byte(`{"error":{"message":"Your OpenAI account has been deactivated. Please contact help.openai.com."}}`)
	match := DetectHardBannedAccount(503, body)
	if match != nil {
		t.Fatalf("expected no hard ban match for 503, got %+v", match)
	}

	advice := BuildBlacklistAdvice(nil, 503, body)
	if advice == nil {
		t.Fatal("expected blacklist advice, got nil")
	}
	if advice.Decision != BlacklistAdviceNotRecommended {
		t.Fatalf("Decision = %q, want %q", advice.Decision, BlacklistAdviceNotRecommended)
	}
	if advice.ReasonCode != "transient_or_retryable" {
		t.Fatalf("ReasonCode = %q, want %q", advice.ReasonCode, "transient_or_retryable")
	}
}
