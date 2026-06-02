package service

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

const (
	contentModerationRepeatedPromptSignalType = "repeated_similar_prompt"
	contentModerationPromptSignalTTL          = 10 * time.Minute
	contentModerationPromptSignalCooldown     = 30 * time.Minute
	contentModerationPromptSignalMaxEntries   = 8
	contentModerationPromptSignalThreshold    = 3
	contentModerationPromptSignalMinTokens    = 3
	contentModerationPromptSignalHammingLimit = 6
)

var contentModerationPromptSignals sync.Map
var contentModerationPromptSignalCooldowns sync.Map

type contentModerationPromptSignalState struct {
	ExactHash [32]byte
	SimHash   uint64
	TokenSet  []uint64
	SeenAt    time.Time
}

type contentModerationPromptSignalCooldownState struct {
	ExactHash [32]byte
	SimHash   uint64
	TokenSet  []uint64
	AlertedAt time.Time
}

func RecordContentModerationRepeatedPromptSignal(input *ContentModerationRecordInput, content string, now time.Time) {
	if input == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	subjectKey := contentModerationPromptSignalSubjectKey(input)
	if subjectKey == "" {
		return
	}
	tokens := contentModerationPromptSignalTokens(content)
	if len(tokens) < contentModerationPromptSignalMinTokens {
		return
	}
	state := contentModerationPromptSignalState{
		ExactHash: contentModerationPromptSignalExactHash(tokens),
		SimHash:   contentModerationPromptSignalSimHash(tokens),
		TokenSet:  contentModerationPromptSignalTokenSet(tokens),
		SeenAt:    now.UTC(),
	}
	if state.SimHash == 0 {
		return
	}

	current := make([]contentModerationPromptSignalState, 0, contentModerationPromptSignalMaxEntries)
	similarObservationCount := 1
	if raw, ok := contentModerationPromptSignals.Load(subjectKey); ok {
		if previous, ok := raw.([]contentModerationPromptSignalState); ok {
			cutoff := now.Add(-contentModerationPromptSignalTTL)
			for _, item := range previous {
				if item.SeenAt.Before(cutoff) {
					continue
				}
				if contentModerationPromptSignalStatesSimilar(item, state) {
					similarObservationCount++
				}
				current = append(current, item)
			}
		}
	}
	current = append(current, state)
	if len(current) > contentModerationPromptSignalMaxEntries {
		current = current[len(current)-contentModerationPromptSignalMaxEntries:]
	}
	contentModerationPromptSignals.Store(subjectKey, current)

	if similarObservationCount < contentModerationPromptSignalThreshold {
		return
	}
	cooldownSuppressed := contentModerationPromptSignalCooldownSuppressed(subjectKey, state, now)
	if cooldownSuppressed {
		return
	}
	contentModerationPromptSignalStoreCooldown(subjectKey, state, now)
	protocolruntime.RecordAbuseSignal(contentModerationRepeatedPromptSignalType)
	logger.L().Warn(
		"content moderation abuse signal: repeated similar prompt",
		contentModerationPromptSignalLogFields(input, similarObservationCount, cooldownSuppressed)...,
	)
}

func contentModerationPromptSignalSubjectKey(input *ContentModerationRecordInput) string {
	if input == nil || input.UserID == nil || input.APIKeyID == nil || *input.UserID <= 0 || *input.APIKeyID <= 0 {
		return ""
	}
	model := NormalizeModelCatalogModelID(input.Model)
	if model == "" {
		return ""
	}
	return strings.Join([]string{
		strconv.FormatInt(*input.UserID, 10),
		strconv.FormatInt(*input.APIKeyID, 10),
		model,
	}, ":")
}

func contentModerationPromptSignalTokens(content string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(content)) {
		r = normalizeFullWidthASCII(r)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			_, _ = b.WriteRune(r)
			continue
		}
		_ = b.WriteByte(' ')
	}
	fields := strings.Fields(b.String())
	if len(fields) == 0 {
		return nil
	}
	if len(fields) > 256 {
		fields = fields[:256]
	}
	return fields
}

func contentModerationPromptSignalExactHash(tokens []string) [32]byte {
	return sha256.Sum256([]byte(strings.Join(tokens, "\x00")))
}

func contentModerationPromptSignalSimHash(tokens []string) uint64 {
	var weights [64]int
	for _, token := range tokens {
		if token == "" {
			continue
		}
		sum := sha256.Sum256([]byte(token))
		value := binary.BigEndian.Uint64(sum[:8])
		for i := 0; i < 64; i++ {
			if value&(uint64(1)<<i) != 0 {
				weights[i]++
			} else {
				weights[i]--
			}
		}
	}
	var out uint64
	for i, weight := range weights {
		if weight > 0 {
			out |= uint64(1) << i
		}
	}
	return out
}

func contentModerationPromptSignalTokenSet(tokens []string) []uint64 {
	if len(tokens) == 0 {
		return nil
	}
	seen := make(map[uint64]struct{}, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		sum := sha256.Sum256([]byte(token))
		seen[binary.BigEndian.Uint64(sum[:8])] = struct{}{}
	}
	out := make([]uint64, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func contentModerationPromptSignalTokenOverlap(a, b []uint64) bool {
	if len(a) < contentModerationPromptSignalMinTokens || len(b) < contentModerationPromptSignalMinTokens {
		return false
	}
	i, j, matches := 0, 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			matches++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	smaller := len(a)
	if len(b) < smaller {
		smaller = len(b)
	}
	return matches*100 >= smaller*85
}

func contentModerationPromptSignalStatesSimilar(a, b contentModerationPromptSignalState) bool {
	return a.ExactHash == b.ExactHash ||
		bits.OnesCount64(a.SimHash^b.SimHash) <= contentModerationPromptSignalHammingLimit ||
		contentModerationPromptSignalTokenOverlap(a.TokenSet, b.TokenSet)
}

func contentModerationPromptSignalCooldownSimilar(
	item contentModerationPromptSignalCooldownState,
	state contentModerationPromptSignalState,
) bool {
	return item.ExactHash == state.ExactHash ||
		bits.OnesCount64(item.SimHash^state.SimHash) <= contentModerationPromptSignalHammingLimit ||
		contentModerationPromptSignalTokenOverlap(item.TokenSet, state.TokenSet)
}

func contentModerationPromptSignalCooldownSuppressed(
	subjectKey string,
	state contentModerationPromptSignalState,
	now time.Time,
) bool {
	current, suppressed := contentModerationPromptSignalPrunedCooldowns(subjectKey, state, now)
	if len(current) > 0 {
		contentModerationPromptSignalCooldowns.Store(subjectKey, current)
	} else {
		contentModerationPromptSignalCooldowns.Delete(subjectKey)
	}
	return suppressed
}

func contentModerationPromptSignalStoreCooldown(
	subjectKey string,
	state contentModerationPromptSignalState,
	now time.Time,
) {
	current, _ := contentModerationPromptSignalPrunedCooldowns(subjectKey, state, now)
	current = append(current, contentModerationPromptSignalCooldownState{
		ExactHash: state.ExactHash,
		SimHash:   state.SimHash,
		TokenSet:  state.TokenSet,
		AlertedAt: now.UTC(),
	})
	if len(current) > contentModerationPromptSignalMaxEntries {
		current = current[len(current)-contentModerationPromptSignalMaxEntries:]
	}
	contentModerationPromptSignalCooldowns.Store(subjectKey, current)
}

func contentModerationPromptSignalPrunedCooldowns(
	subjectKey string,
	state contentModerationPromptSignalState,
	now time.Time,
) ([]contentModerationPromptSignalCooldownState, bool) {
	current := make([]contentModerationPromptSignalCooldownState, 0, contentModerationPromptSignalMaxEntries)
	suppressed := false
	raw, ok := contentModerationPromptSignalCooldowns.Load(subjectKey)
	if !ok {
		return current, false
	}
	previous, ok := raw.([]contentModerationPromptSignalCooldownState)
	if !ok {
		return current, false
	}
	cutoff := now.Add(-contentModerationPromptSignalCooldown)
	for _, item := range previous {
		if item.AlertedAt.Before(cutoff) {
			continue
		}
		if contentModerationPromptSignalCooldownSimilar(item, state) {
			suppressed = true
		}
		current = append(current, item)
	}
	return current, suppressed
}

func contentModerationPromptSignalLogFields(
	input *ContentModerationRecordInput,
	similarObservationCount int,
	cooldownSuppressed bool,
) []zap.Field {
	fields := []zap.Field{
		zap.String("component", "service.content_moderation"),
		zap.String("signal_scope", "observation_only"),
		zap.String("signal_type", contentModerationRepeatedPromptSignalType),
		zap.Int("similar_observation_count", similarObservationCount),
		zap.Bool("cooldown_suppressed", cooldownSuppressed),
		zap.String("model", NormalizeModelCatalogModelID(input.Model)),
		zap.String("source_endpoint", strings.TrimSpace(input.SourceEndpoint)),
	}
	if input.UserID != nil {
		fields = append(fields, zap.Int64("user_id", *input.UserID))
	}
	if input.APIKeyID != nil {
		fields = append(fields, zap.Int64("api_key_id", *input.APIKeyID))
	}
	if requestID := strings.TrimSpace(input.RequestID); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	return fields
}

func resetContentModerationPromptSignalsForTest() {
	contentModerationPromptSignals = sync.Map{}
	contentModerationPromptSignalCooldowns = sync.Map{}
}
