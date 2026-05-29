package service

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
)

func evaluateOpsRequestTraceDecision(runtimeCfg opsRequestTraceRuntimeConfig, input *OpsRecordRequestTraceInput) RequestCaptureDecision {
	decision := RequestCaptureDecision{
		Capture:    false,
		Reason:     "",
		Sampled:    false,
		RawEnabled: strings.TrimSpace(runtimeCfg.EncryptionKey) != "",
	}
	if input == nil || !runtimeCfg.Enabled {
		return decision
	}

	normalize := input.Trace.Normalize
	switch {
	case strings.TrimSpace(normalize.ProbeAction) != "":
		decision.Capture = true
		decision.Reason = "probe_action"
	case input.StatusCode >= 400:
		decision.Capture = true
		decision.Reason = "error"
	case input.DurationMs >= runtimeCfg.ForceCaptureSlowMs:
		decision.Capture = true
		decision.Reason = "slow"
	case normalize.Stream:
		decision.Capture = true
		decision.Reason = "stream"
	case normalize.HasTools:
		decision.Capture = true
		decision.Reason = "tools"
	case normalize.HasThinking:
		decision.Capture = true
		decision.Reason = "thinking"
	case isGoogleProtocolTrace(normalize):
		decision.Capture = true
		decision.Reason = "google_gateway"
	case normalize.ProtocolIn != "" && normalize.ProtocolOut != "" && normalize.ProtocolIn != normalize.ProtocolOut:
		decision.Capture = true
		decision.Reason = "protocol_transform"
	case shouldSampleRequestTrace(runtimeCfg.SuccessSampleRate, input):
		decision.Capture = true
		decision.Sampled = true
		decision.Reason = "success_sampled"
	}
	return decision
}

func normalizeOpsRequestTraceStatus(status string, statusCode int) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status != "" {
		return status
	}
	if statusCode >= 400 {
		return "error"
	}
	return "success"
}

func isGoogleProtocolTrace(normalize ProtocolNormalizeResult) bool {
	for _, value := range []string{normalize.Platform, normalize.ProtocolIn, normalize.ProtocolOut, normalize.Channel} {
		value = strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(value, "gemini") || strings.Contains(value, "vertex") || strings.Contains(value, "google") {
			return true
		}
	}
	return false
}

func shouldSampleRequestTrace(rate float64, input *OpsRecordRequestTraceInput) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	key := strings.TrimSpace(firstNonEmptyString(input.RequestID, input.ClientRequestID, input.UpstreamRequestID))
	if key == "" {
		key = strconv.FormatInt(input.CreatedAt.UnixNano(), 10)
	}
	sum := sha256.Sum256([]byte(key))
	value := binary.BigEndian.Uint64(sum[:8])
	threshold := uint64(rate * float64(^uint64(0)))
	return value <= threshold
}
