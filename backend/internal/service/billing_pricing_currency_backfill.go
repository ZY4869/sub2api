package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const billingPricingRuntimeFXBackfillActor = "runtime_fx_backfill"

func billingContextRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
		return "client:" + strings.TrimSpace(clientRequestID)
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		return "local:" + strings.TrimSpace(requestID)
	}
	return ""
}

func billingPricingFXState(meta ModelPricingCurrencyMetadata) string {
	switch normalizeBillingCurrency(meta.Currency) {
	case ModelPricingCurrencyCNY:
		if meta.USDToCNYRate != nil && *meta.USDToCNYRate > 0 && meta.FXLockedAt != nil && !meta.FXLockedAt.IsZero() {
			return "locked"
		}
		return "pending"
	default:
		return "usd"
	}
}

func modelPricingCurrencyMetadataFromExchangeRate(rate *ModelCatalogExchangeRate) ModelPricingCurrencyMetadata {
	meta := ModelPricingCurrencyMetadata{Currency: ModelPricingCurrencyCNY}
	if rate == nil || rate.Rate <= 0 {
		return meta
	}
	lockedAt := rate.UpdatedAt
	if lockedAt.IsZero() {
		lockedAt = time.Now().UTC()
	}
	meta.USDToCNYRate = modelCatalogFloat64Ptr(rate.Rate)
	meta.FXRateDate = strings.TrimSpace(rate.Date)
	meta.FXLockedAt = cloneBillingTime(&lockedAt)
	return meta
}

func (s *BillingCenterService) backfillModelPricingCurrencyFX(
	ctx context.Context,
	model string,
	meta ModelPricingCurrencyMetadata,
) error {
	if s == nil || s.modelCatalogService == nil {
		return nil
	}
	if normalizeBillingCurrency(meta.Currency) != ModelPricingCurrencyCNY || meta.USDToCNYRate == nil || *meta.USDToCNYRate <= 0 {
		return nil
	}
	if err := s.modelCatalogService.saveModelPricingCurrency(
		ctx,
		ModelCatalogActor{Email: billingPricingRuntimeFXBackfillActor},
		model,
		ModelPricingCurrencyCNY,
		meta,
	); err != nil {
		return err
	}
	recordBillingPricingRuntimeFXBackfillSuccess()
	s.syncBillingServiceOverrides(ctx)
	logger.FromContext(ctx).Info(
		"billing pricing runtime fx backfill completed",
		zap.String("component", "service.billing_center"),
		zap.String("request_id", billingContextRequestID(ctx)),
		zap.String("model", NormalizeModelCatalogModelID(model)),
		zap.Float64("fx_rate", *meta.USDToCNYRate),
		zap.String("fx_date", strings.TrimSpace(meta.FXRateDate)),
		zap.Time("fx_locked_at", meta.FXLockedAt.UTC()),
	)
	return nil
}
