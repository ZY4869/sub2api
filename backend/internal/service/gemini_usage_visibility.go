package service

import "strings"

type GeminiSuccessUsageDecision struct {
	Persist       bool
	Reason        string
	OperationType string
}

func resolveGeminiOperationPath(inboundEndpoint string, rawInboundPath string) string {
	if trimmed := strings.TrimSpace(rawInboundPath); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(inboundEndpoint)
}

func DecideGeminiSuccessUsagePersistence(inboundEndpoint string, rawInboundPath string, requestBody []byte) GeminiSuccessUsageDecision {
	operationType := detectGeminiOperationType(resolveGeminiOperationPath(inboundEndpoint, rawInboundPath), requestBody)
	switch operationType {
	case "models":
		return GeminiSuccessUsageDecision{Persist: false, Reason: "control_plane_models", OperationType: operationType}
	case "auth_tokens":
		return GeminiSuccessUsageDecision{Persist: false, Reason: "control_plane_auth_tokens", OperationType: operationType}
	case "operation_status":
		return GeminiSuccessUsageDecision{Persist: false, Reason: "control_plane_operation_status", OperationType: operationType}
	case "count_tokens":
		return GeminiSuccessUsageDecision{Persist: false, Reason: "control_plane_count_tokens", OperationType: operationType}
	case "batch_operation":
		return GeminiSuccessUsageDecision{Persist: false, Reason: "control_plane_batch_operation", OperationType: operationType}
	case "file_operation":
		return GeminiSuccessUsageDecision{Persist: false, Reason: "control_plane_file_operation", OperationType: operationType}
	case "cached_content_read":
		return GeminiSuccessUsageDecision{Persist: false, Reason: "control_plane_cached_content_read", OperationType: operationType}
	default:
		return GeminiSuccessUsageDecision{Persist: true, Reason: operationType, OperationType: operationType}
	}
}
