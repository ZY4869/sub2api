package repository

import (
	"context"
	"database/sql"
	"errors"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"strings"
	"time"
)

type usageLogSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (r *usageLogRepository) Create(ctx context.Context, log *service.UsageLog) (bool, error) {
	if log == nil {
		return false, nil
	}
	var sqlq usageLogSQLExecutor = r.sql
	if tx := dbent.TxFromContext(ctx); tx != nil {
		sqlq = tx.Client()
	}
	createdAt := log.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	requestID := strings.TrimSpace(log.RequestID)
	log.RequestID = requestID
	rateMultiplier := log.RateMultiplier
	log.SyncRequestTypeAndLegacyFields()
	requestType := int16(log.RequestType)
	query := `
		INSERT INTO usage_logs (
			user_id,
			api_key_id,
			account_id,
			request_id,
			model,
			requested_model,
			upstream_model,
			group_id,
			subscription_id,
			input_tokens,
			output_tokens,
			cache_creation_tokens,
			cache_read_tokens,
			cache_creation_5m_tokens,
			cache_creation_1h_tokens,
			input_cost,
			output_cost,
			cache_creation_cost,
			cache_read_cost,
			total_cost,
			actual_cost,
			billing_exempt_reason,
			rate_multiplier,
			account_rate_multiplier,
			billing_type,
			request_type,
			status,
			stream,
			openai_ws_mode,
			duration_ms,
			first_token_ms,
			user_agent,
			ip_address,
			http_status,
			error_code,
			error_message,
			simulated_client,
			image_count,
			image_size,
			media_type,
			service_tier,
			reasoning_effort,
			thinking_enabled,
			inbound_endpoint,
			upstream_endpoint,
			cache_ttl_overridden,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15,
			$16, $17, $18, $19, $20, $21,
			$22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47
		)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id, created_at
	`
	requestedModel := strings.TrimSpace(log.RequestedModel)
	if requestedModel == "" {
		requestedModel = strings.TrimSpace(log.Model)
	}
	log.RequestedModel = requestedModel
	upstreamModel := nullString(log.UpstreamModel)
	requestedModelPtr := &requestedModel
	groupID := nullInt64(log.GroupID)
	subscriptionID := nullInt64(log.SubscriptionID)
	duration := nullInt(log.DurationMs)
	firstToken := nullInt(log.FirstTokenMs)
	userAgent := nullString(log.UserAgent)
	ipAddress := nullString(log.IPAddress)
	imageSize := nullString(log.ImageSize)
	mediaType := nullString(log.MediaType)
	serviceTier := nullString(log.ServiceTier)
	reasoningEffort := nullString(log.ReasoningEffort)
	thinkingEnabled := nullUsageLogBool(log.ThinkingEnabled)
	inboundEndpoint := nullString(log.InboundEndpoint)
	upstreamEndpoint := nullString(log.UpstreamEndpoint)
	billingExemptReason := nullString(log.BillingExemptReason)
	status := service.NormalizeUsageLogStatus(log.Status)
	httpStatus := nullInt(log.HTTPStatus)
	errorCode := nullString(log.ErrorCode)
	errorMessage := nullString(log.ErrorMessage)
	simulatedClient := nullString(service.NormalizeUsageLogSimulatedClient(nullStringValue(log.SimulatedClient)))
	var requestIDArg any
	if requestID != "" {
		requestIDArg = requestID
	}
	args := []any{log.UserID, log.APIKeyID, log.AccountID, requestIDArg, log.Model, nullString(requestedModelPtr), upstreamModel, groupID, subscriptionID, log.InputTokens, log.OutputTokens, log.CacheCreationTokens, log.CacheReadTokens, log.CacheCreation5mTokens, log.CacheCreation1hTokens, log.InputCost, log.OutputCost, log.CacheCreationCost, log.CacheReadCost, log.TotalCost, log.ActualCost, billingExemptReason, rateMultiplier, log.AccountRateMultiplier, log.BillingType, requestType, status, log.Stream, log.OpenAIWSMode, duration, firstToken, userAgent, ipAddress, httpStatus, errorCode, errorMessage, simulatedClient, log.ImageCount, imageSize, mediaType, serviceTier, reasoningEffort, thinkingEnabled, inboundEndpoint, upstreamEndpoint, log.CacheTTLOverridden, createdAt}
	if err := scanSingleRow(ctx, sqlq, query, args, &log.ID, &log.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) && requestID != "" {
			selectQuery := "SELECT id, created_at FROM usage_logs WHERE request_id = $1 AND api_key_id = $2"
			if err := scanSingleRow(ctx, sqlq, selectQuery, []any{requestID, log.APIKeyID}, &log.ID, &log.CreatedAt); err != nil {
				return false, err
			}
			log.RateMultiplier = rateMultiplier
			return false, nil
		} else {
			return false, err
		}
	}
	log.RateMultiplier = rateMultiplier
	return true, nil
}
