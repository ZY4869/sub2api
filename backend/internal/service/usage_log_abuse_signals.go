package service

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

const (
	usageLogLongContextSignalTokens       = 200_000
	usageLogHighRiskModelSwitchSignalTTL  = 10 * time.Minute
	usageLogHighRiskModelSwitchSignalType = "high_risk_model_switch"
)

var usageLogLastHighRiskModelBySubject sync.Map

type usageLogModelSwitchState struct {
	Model  string
	SeenAt time.Time
}

func RecordUsageLogAbuseSignals(log *UsageLog) {
	if log == nil {
		return
	}
	recordUsageLogLongContextSignal(log)
	recordUsageLogHighRiskModelSwitchSignal(log, time.Now())
}

func recordUsageLogLongContextSignal(log *UsageLog) {
	if log.RequestContextLengthTokens == nil || *log.RequestContextLengthTokens < usageLogLongContextSignalTokens {
		return
	}
	protocolruntime.RecordAbuseSignal("long_context_request")
	fields := usageLogAbuseSignalFields(log)
	fields = append(fields, zap.Int("request_context_length_tokens", *log.RequestContextLengthTokens))
	logger.L().Info("usage abuse signal: long context request", fields...)
}

func recordUsageLogHighRiskModelSwitchSignal(log *UsageLog, now time.Time) {
	model := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(log.RequestedModel, log.Model))
	if model == "" || !usageLogHighRiskModel(model) || log.APIKeyID <= 0 {
		return
	}
	key := usageLogHighRiskModelSwitchSubjectKey(log)
	if key == "" {
		return
	}
	if previousRaw, ok := usageLogLastHighRiskModelBySubject.Load(key); ok {
		if previous, ok := previousRaw.(usageLogModelSwitchState); ok &&
			previous.Model != "" &&
			previous.Model != model &&
			now.Sub(previous.SeenAt) <= usageLogHighRiskModelSwitchSignalTTL {
			protocolruntime.RecordAbuseSignal(usageLogHighRiskModelSwitchSignalType)
			fields := usageLogAbuseSignalFields(log)
			fields = append(fields,
				zap.String("previous_model", previous.Model),
				zap.Duration("switch_window", now.Sub(previous.SeenAt)),
			)
			logger.L().Warn("usage abuse signal: high risk model switch", fields...)
		}
	}
	usageLogLastHighRiskModelBySubject.Store(key, usageLogModelSwitchState{Model: model, SeenAt: now})
}

func usageLogHighRiskModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "opus") ||
		strings.Contains(normalized, "reason") ||
		strings.Contains(normalized, "gpt-5") ||
		strings.Contains(normalized, "deepseek-v4")
}

func usageLogHighRiskModelSwitchSubjectKey(log *UsageLog) string {
	if log == nil || log.APIKeyID <= 0 {
		return ""
	}
	return strings.Join([]string{
		strconv.FormatInt(log.UserID, 10),
		strconv.FormatInt(log.APIKeyID, 10),
	}, ":")
}

func usageLogAbuseSignalFields(log *UsageLog) []zap.Field {
	fields := []zap.Field{
		zap.String("component", "service.usage_log"),
		zap.String("signal_scope", "observation_only"),
		zap.String("model", NormalizeModelCatalogModelID(firstNonEmptyTrimmed(log.RequestedModel, log.Model))),
		zap.Int64("user_id", log.UserID),
		zap.Int64("api_key_id", log.APIKeyID),
	}
	if requestID := strings.TrimSpace(log.RequestID); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	return fields
}
