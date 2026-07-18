package securityaudit

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const logComponent = "securityaudit.prompt"

func promptAuditLog(ctx context.Context) *zap.Logger {
	return logger.FromContext(ctx).With(zap.String("component", logComponent))
}

func requestLogFields(req Request, promptHash string, result *NormalizedResult, errorCode string) []zap.Field {
	fields := []zap.Field{
		zap.String("request_id", strings.TrimSpace(req.RequestID)),
		zap.String("client_request_id", strings.TrimSpace(req.ClientRequestID)),
		zap.String("prompt_hash", strings.TrimSpace(promptHash)),
		zap.String("protocol", strings.TrimSpace(req.Protocol)),
		zap.String("model", strings.TrimSpace(req.Model)),
		zap.String("error_code", strings.TrimSpace(errorCode)),
	}
	if result != nil {
		fields = append(fields,
			zap.String("guard_endpoint_id", strings.TrimSpace(result.GuardEndpointID)),
			zap.Int("latency_ms", result.LatencyMS),
			zap.String("decision", string(result.Decision)),
		)
		return fields
	}
	fields = append(fields,
		zap.String("guard_endpoint_id", ""),
		zap.Int("latency_ms", 0),
		zap.String("decision", ""),
	)
	return fields
}

func jobLogFields(job *Job, result *NormalizedResult, errorCode string) []zap.Field {
	req := Request{}
	promptHash := ""
	if job != nil {
		req = job.Request
		promptHash = job.PromptHash
	}
	return requestLogFields(req, promptHash, result, errorCode)
}

func errorReason(err error) string {
	if err == nil {
		return ""
	}
	if reason := strings.TrimSpace(infraerrors.Reason(err)); reason != "" {
		return reason
	}
	return guardErrorCode(err)
}
