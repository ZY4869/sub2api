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
			channel_id,
			model_mapping_chain,
			billing_tier,
			billing_mode,
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
			billing_currency,
			total_cost_usd_equivalent,
			actual_cost_usd_equivalent,
			usd_to_cny_rate,
			fx_rate_date,
			fx_locked_at,
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
			operation_type,
			charge_source,
			image_count,
			image_size,
			image_output_tokens,
			image_output_cost,
			service_tier,
			reasoning_effort,
			thinking_enabled,
			inbound_endpoint,
			upstream_endpoint,
			upstream_url,
			upstream_service,
			cache_ttl_overridden,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18,
			$19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56, $57, $58, $59, $60, $61, $62
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
	channelID := nullInt64(log.ChannelID)
	modelMappingChain := nullString(log.ModelMappingChain)
	billingTier := nullString(log.BillingTier)
	billingMode := nullString(log.BillingMode)
	groupID := nullInt64(log.GroupID)
	subscriptionID := nullInt64(log.SubscriptionID)
	duration := nullInt(log.DurationMs)
	firstToken := nullInt(log.FirstTokenMs)
	userAgent := nullString(log.UserAgent)
	ipAddress := nullString(log.IPAddress)
	imageSize := nullString(log.ImageSize)
	serviceTier := nullString(log.ServiceTier)
	reasoningEffort := nullString(log.ReasoningEffort)
	thinkingEnabled := nullUsageLogBool(log.ThinkingEnabled)
	inboundEndpoint := nullString(log.InboundEndpoint)
	upstreamEndpoint := nullString(log.UpstreamEndpoint)
	upstreamURL := nullString(log.UpstreamURL)
	upstreamService := nullString(log.UpstreamService)
	billingExemptReason := nullString(log.BillingExemptReason)
	status := service.NormalizeUsageLogStatus(log.Status)
	httpStatus := nullInt(log.HTTPStatus)
	errorCode := nullString(log.ErrorCode)
	errorMessage := nullString(log.ErrorMessage)
	simulatedClient := nullString(service.NormalizeUsageLogSimulatedClient(nullStringValue(log.SimulatedClient)))
	operationType := nullString(log.OperationType)
	chargeSource := nullString(log.ChargeSource)
	imageOutputTokens := nullInt(log.ImageOutputTokens)
	imageOutputCost := nullFloat(log.ImageOutputCost)
	billingCurrency := service.NormalizeUsageBillingCurrency(log.BillingCurrency)
	totalCostUSDEquivalent := log.TotalCostUSDEquivalent
	if totalCostUSDEquivalent == 0 && log.TotalCost != 0 {
		totalCostUSDEquivalent = service.CostUSDEquivalentForPersistence(log.TotalCost, billingCurrency, log.USDToCNYRate)
	}
	actualCostUSDEquivalent := log.ActualCostUSDEquivalent
	if actualCostUSDEquivalent == 0 && log.ActualCost != 0 {
		actualCostUSDEquivalent = service.CostUSDEquivalentForPersistence(log.ActualCost, billingCurrency, log.USDToCNYRate)
	}
	fxRateDate := nullString(log.FXRateDate)
	fxLockedAt := nullUsageLogTime(log.FXLockedAt)
	var requestIDArg any
	if requestID != "" {
		requestIDArg = requestID
	}
	args := []any{log.UserID, log.APIKeyID, log.AccountID, requestIDArg, log.Model, nullString(requestedModelPtr), upstreamModel, channelID, modelMappingChain, billingTier, billingMode, groupID, subscriptionID, log.InputTokens, log.OutputTokens, log.CacheCreationTokens, log.CacheReadTokens, log.CacheCreation5mTokens, log.CacheCreation1hTokens, log.InputCost, log.OutputCost, log.CacheCreationCost, log.CacheReadCost, log.TotalCost, log.ActualCost, billingCurrency, totalCostUSDEquivalent, actualCostUSDEquivalent, log.USDToCNYRate, fxRateDate, fxLockedAt, billingExemptReason, rateMultiplier, log.AccountRateMultiplier, log.BillingType, requestType, status, log.Stream, log.OpenAIWSMode, duration, firstToken, userAgent, ipAddress, httpStatus, errorCode, errorMessage, simulatedClient, operationType, chargeSource, log.ImageCount, imageSize, imageOutputTokens, imageOutputCost, serviceTier, reasoningEffort, thinkingEnabled, inboundEndpoint, upstreamEndpoint, upstreamURL, upstreamService, log.CacheTTLOverridden, createdAt}
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
