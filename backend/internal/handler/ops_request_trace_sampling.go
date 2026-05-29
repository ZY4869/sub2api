package handler

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func shouldQueueOpsRequestTrace(input *service.OpsRecordRequestTraceInput) bool {
	if input == nil {
		return false
	}
	normalize := input.Trace.Normalize
	switch {
	case input.StatusCode >= 400:
		return true
	case input.DurationMs >= opsRequestTraceSlowMs:
		return true
	case normalize.Stream:
		return true
	case normalize.HasTools:
		return true
	case normalize.HasThinking:
		return true
	case isGoogleTraceForQueue(normalize):
		return true
	case normalize.ProtocolIn != "" && normalize.ProtocolOut != "" && normalize.ProtocolIn != normalize.ProtocolOut:
		return true
	default:
		return shouldSampleOpsTrace(opsRequestTraceSampleRate, input)
	}
}

func isGoogleTraceForQueue(normalize service.ProtocolNormalizeResult) bool {
	for _, value := range []string{normalize.Platform, normalize.Channel} {
		value = strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(value, "gemini") || strings.Contains(value, "vertex") || strings.Contains(value, "google") {
			return true
		}
	}
	for _, value := range []string{normalize.ProtocolIn, normalize.ProtocolOut} {
		if opsTraceProtocolFamily(value) == "gemini" {
			return true
		}
		value = strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(value, "vertex") || strings.Contains(value, "google") {
			return true
		}
	}
	return false
}

func shouldSampleOpsTrace(rate float64, input *service.OpsRecordRequestTraceInput) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	key := strings.TrimSpace(firstNonEmptyString(input.RequestID, input.ClientRequestID, input.UpstreamRequestID))
	if key == "" {
		key = input.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	sum := serviceHashString(key)
	return float64(sum%10000)/10000.0 < rate
}
