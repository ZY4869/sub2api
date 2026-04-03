package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type BlacklistAdviceDecision string

const (
	BlacklistAdviceAutoBlacklisted    BlacklistAdviceDecision = "auto_blacklisted"
	BlacklistAdviceRecommendBlacklist BlacklistAdviceDecision = "recommend_blacklist"
	BlacklistAdviceNotRecommended     BlacklistAdviceDecision = "not_recommended"
)

type BlacklistAdvice struct {
	Decision            BlacklistAdviceDecision `json:"decision"`
	ReasonCode          string                  `json:"reason_code,omitempty"`
	ReasonMessage       string                  `json:"reason_message,omitempty"`
	AlreadyBlacklisted  bool                    `json:"already_blacklisted,omitempty"`
	FeedbackFingerprint string                  `json:"feedback_fingerprint,omitempty"`
	CollectFeedback     bool                    `json:"collect_feedback,omitempty"`
	Platform            string                  `json:"platform,omitempty"`
	StatusCode          int                     `json:"status_code,omitempty"`
	ErrorCode           string                  `json:"error_code,omitempty"`
	MessageKeywords     []string                `json:"message_keywords,omitempty"`
}

type BlacklistFeedbackInput struct {
	Fingerprint     string   `json:"fingerprint,omitempty"`
	AdviceDecision  string   `json:"advice_decision,omitempty"`
	Action          string   `json:"action,omitempty"`
	Platform        string   `json:"platform,omitempty"`
	StatusCode      int      `json:"status_code,omitempty"`
	ErrorCode       string   `json:"error_code,omitempty"`
	MessageKeywords []string `json:"message_keywords,omitempty"`
}

type BlacklistRuleCandidate struct {
	Fingerprint     string   `json:"fingerprint"`
	Platform        string   `json:"platform"`
	StatusCode      int      `json:"status_code"`
	ErrorCode       string   `json:"error_code,omitempty"`
	MessageKeywords []string `json:"message_keywords,omitempty"`
	AdviceDecision  string   `json:"advice_decision,omitempty"`
	AdminAction     string   `json:"admin_action,omitempty"`
	OccurrenceCount int      `json:"occurrence_count"`
	LastSeenAt      string   `json:"last_seen_at"`
}

type BlacklistRuleCandidateSettings struct {
	Rules []BlacklistRuleCandidate `json:"rules"`
}

var blacklistAdviceStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "api": {}, "account": {}, "accounts": {},
	"auth": {}, "by": {}, "for": {}, "from": {}, "http": {}, "in": {}, "is": {},
	"it": {}, "of": {}, "on": {}, "or": {}, "returned": {}, "status": {}, "the": {},
	"this": {}, "to": {}, "upstream": {}, "was": {}, "with": {},
}

func BuildBlacklistAdvice(account *Account, statusCode int, responseBody []byte) *BlacklistAdvice {
	bodyText := strings.TrimSpace(string(responseBody))
	if account == nil && bodyText == "" {
		return nil
	}

	platform := ""
	if account != nil {
		platform = RoutingPlatformForAccount(account)
	}
	alreadyBlacklisted := account != nil && NormalizeAccountLifecycleInput(account.LifecycleState) == AccountLifecycleBlacklisted

	if match := DetectHardBannedAccount(statusCode, responseBody); match != nil {
		keywords := deriveBlacklistAdviceKeywords(match.ReasonMessage)
		return &BlacklistAdvice{
			Decision:            BlacklistAdviceAutoBlacklisted,
			ReasonCode:          strings.TrimSpace(match.ReasonCode),
			ReasonMessage:       strings.TrimSpace(match.ReasonMessage),
			AlreadyBlacklisted:  true,
			FeedbackFingerprint: buildBlacklistAdviceFingerprint(platform, statusCode, match.ReasonCode, keywords),
			CollectFeedback:     false,
			Platform:            platform,
			StatusCode:          statusCode,
			ErrorCode:           strings.TrimSpace(match.ReasonCode),
			MessageKeywords:     keywords,
		}
	}

	code, message := extractBlacklistAdviceCodeAndMessage(responseBody)
	if message == "" {
		message = bodyText
	}
	keywords := deriveBlacklistAdviceKeywords(message)
	fingerprint := buildBlacklistAdviceFingerprint(platform, statusCode, code, keywords)
	lowerMessage := strings.ToLower(strings.TrimSpace(message))
	lowerCode := strings.ToLower(strings.TrimSpace(code))

	advice := &BlacklistAdvice{
		Decision:            BlacklistAdviceNotRecommended,
		ReasonCode:          "transient_or_unknown_failure",
		ReasonMessage:       firstNonEmptyHardBanString(message, bodyText),
		AlreadyBlacklisted:  alreadyBlacklisted,
		FeedbackFingerprint: fingerprint,
		CollectFeedback:     !alreadyBlacklisted,
		Platform:            platform,
		StatusCode:          statusCode,
		ErrorCode:           lowerCode,
		MessageKeywords:     keywords,
	}

	switch {
	case isRecommendedBlacklistError(lowerCode, lowerMessage, statusCode):
		advice.Decision = BlacklistAdviceRecommendBlacklist
		advice.ReasonCode = firstNonEmptyHardBanString(lowerCode, "credentials_likely_invalid")
	case isDefinitelyTransientError(lowerCode, lowerMessage, statusCode):
		advice.Decision = BlacklistAdviceNotRecommended
		advice.ReasonCode = firstNonEmptyHardBanString(lowerCode, "transient_or_retryable")
	default:
		advice.Decision = BlacklistAdviceNotRecommended
		advice.ReasonCode = firstNonEmptyHardBanString(lowerCode, "manual_review_needed")
	}

	if alreadyBlacklisted {
		advice.Decision = BlacklistAdviceAutoBlacklisted
		advice.ReasonCode = firstNonEmptyHardBanString(account.LifecycleReasonCode, advice.ReasonCode, "already_blacklisted")
		advice.ReasonMessage = firstNonEmptyHardBanString(account.LifecycleReasonMessage, advice.ReasonMessage, bodyText)
		advice.CollectFeedback = false
	}

	return advice
}

func extractBlacklistAdviceCodeAndMessage(responseBody []byte) (string, string) {
	envelope := hardBanErrorEnvelope{}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return "", strings.TrimSpace(string(responseBody))
	}
	code, message := extractHardBanCodeAndMessage(envelope)
	return strings.TrimSpace(strings.ToLower(code)), strings.TrimSpace(message)
}

func isRecommendedBlacklistError(code string, message string, statusCode int) bool {
	if code == "" && message == "" {
		return false
	}

	if statusCode == 401 || statusCode == 403 {
		if strings.Contains(code, "invalid") ||
			strings.Contains(code, "revoked") ||
			strings.Contains(code, "expired") ||
			strings.Contains(code, "unauthorized") ||
			strings.Contains(code, "forbidden") {
			return true
		}
		if strings.Contains(message, "unauthorized") ||
			strings.Contains(message, "forbidden") ||
			strings.Contains(message, "invalid credentials") ||
			strings.Contains(message, "expired credentials") {
			return true
		}
	}

	recommendedPhrases := []string{
		"invalid api key",
		"invalid_api_key",
		"incorrect api key",
		"token has been revoked",
		"refresh token is invalid",
		"access token is invalid",
		"invalid authentication",
		"authentication failed",
		"permission denied for this key",
		"workspace not found",
		"subscription has ended",
	}
	for _, phrase := range recommendedPhrases {
		if strings.Contains(message, phrase) {
			return true
		}
	}
	return false
}

func isDefinitelyTransientError(code string, message string, statusCode int) bool {
	if statusCode == 429 || statusCode == 500 || statusCode == 502 || statusCode == 503 || statusCode == 504 || statusCode == 529 {
		return true
	}
	transientPhrases := []string{
		"rate limit",
		"too many requests",
		"temporarily unavailable",
		"try again later",
		"timeout",
		"timed out",
		"overloaded",
		"billing issue",
		"insufficient_quota",
		"unsupported_country_code",
		"cloudflare challenge",
	}
	for _, phrase := range transientPhrases {
		if strings.Contains(code, phrase) || strings.Contains(message, phrase) {
			return true
		}
	}
	return false
}

func deriveBlacklistAdviceKeywords(message string) []string {
	if strings.TrimSpace(message) == "" {
		return nil
	}
	tokenizer := strings.NewReplacer(
		"{", " ", "}", " ", "[", " ", "]", " ", "(", " ", ")", " ", ",", " ",
		":", " ", ";", " ", ".", " ", "\"", " ", "'", " ", "\n", " ", "\r", " ", "\t", " ", "/", " ",
	)
	parts := strings.Fields(tokenizer.Replace(strings.ToLower(message)))
	seen := make(map[string]struct{}, len(parts))
	keywords := make([]string, 0, 6)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 3 {
			continue
		}
		if _, stop := blacklistAdviceStopWords[part]; stop {
			continue
		}
		if _, err := strconv.Atoi(part); err == nil {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		keywords = append(keywords, part)
		if len(keywords) >= 6 {
			break
		}
	}
	sort.Strings(keywords)
	return keywords
}

func buildBlacklistAdviceFingerprint(platform string, statusCode int, errorCode string, keywords []string) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(strings.ToLower(platform)),
		strconv.Itoa(statusCode),
		strings.TrimSpace(strings.ToLower(errorCode)),
		strings.Join(normalizeStringList(keywords, strings.ToLower), "|"),
	}, "::")))
	return hex.EncodeToString(hash[:8])
}
