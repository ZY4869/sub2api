package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func (s *PaymentService) paymentSettings(ctx context.Context) PaymentSettings {
	if s != nil && s.paymentSettingsOverride != nil {
		return s.paymentSettingsOverride(ctx)
	}
	if s == nil || s.settings == nil {
		return DefaultPaymentSettings()
	}
	return s.settings.GetPaymentSettings(ctx)
}

func resolvePaymentMode(settings PaymentSettings) string {
	if settings.MobileForceQRCodeEnabled {
		return PaymentModeQRCode
	}
	return PaymentModeDefault
}

func (s *PaymentService) logInfo(ctx context.Context, msg string, fields ...zap.Field) {
	logger.FromContext(ctx).Info(msg, append(paymentLogFields(ctx), fields...)...)
}

func (s *PaymentService) logWarn(ctx context.Context, msg string, fields ...zap.Field) {
	logger.FromContext(ctx).Warn(msg, append(paymentLogFields(ctx), fields...)...)
}

func paymentLogFields(ctx context.Context) []zap.Field {
	fields := []zap.Field{zap.String("component", "payment")}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		fields = append(fields, zap.String("request_id", strings.TrimSpace(requestID)))
	}
	return fields
}

func randomPaymentHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func hashPaymentToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func firstPaymentString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func redactPaymentPayload(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitivePaymentPayloadKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = redactPaymentPayload(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, redactPaymentPayload(item))
		}
		return out
	default:
		return value
	}
}

func isSensitivePaymentPayloadKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"secret", "token", "api_key", "apikey", "authorization", "client_secret", "email", "phone", "name", "address"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}
