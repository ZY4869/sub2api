package handler

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

type imageCountQuotaService interface {
	TryReserveImageCount(ctx context.Context, id int64, count int) (bool, error)
	RollbackImageCount(ctx context.Context, id int64, count int) error
}

func settleAPIKeyImageCountUnits(
	ctx context.Context,
	reqLog *zap.Logger,
	quotaService imageCountQuotaService,
	apiKey *service.APIKey,
	reservedUnits int,
	actualCount int,
	actualTier string,
) bool {
	if reservedUnits <= 0 {
		return true
	}
	if reqLog == nil {
		reqLog = zap.NewNop()
	}
	if quotaService == nil || apiKey == nil {
		reqLog.Error("api_key_image_count_settle_unavailable",
			zap.Int("reserved_units", reservedUnits),
			zap.Int("actual_count", actualCount),
			zap.String("image_size_tier", strings.TrimSpace(actualTier)),
		)
		return true
	}

	actualUnits := apiKey.ImageCountUnitsForTier(actualCount, actualTier)
	fields := []zap.Field{
		zap.String("image_size_tier", actualTier),
		zap.Int("final_count", actualCount),
		zap.Int("actual_units", actualUnits),
		zap.Int("reserved_units", reservedUnits),
	}
	if actualUnits < reservedUnits {
		diff := reservedUnits - actualUnits
		if err := quotaService.RollbackImageCount(ctx, apiKey.ID, diff); err != nil {
			reqLog.Error("api_key_image_count_settle_rollback_failed", append(fields, zap.Error(err), zap.Int("rollback_units", diff))...)
			return true
		}
		reqLog.Info("api_key_image_count_settled", append(fields, zap.Int("rollback_units", diff))...)
		return true
	}
	if actualUnits > reservedUnits {
		diff := actualUnits - reservedUnits
		ok, err := quotaService.TryReserveImageCount(ctx, apiKey.ID, diff)
		if err != nil {
			reqLog.Error("api_key_image_count_settle_reserve_failed", append(fields, zap.Error(err), zap.Int("extra_units", diff))...)
			return true
		}
		if !ok {
			reqLog.Warn("api_key_image_count_settle_reserve_exhausted", append(fields, zap.Int("extra_units", diff))...)
			return true
		}
		reqLog.Info("api_key_image_count_settled", append(fields, zap.Int("extra_units", diff))...)
		return true
	}

	reqLog.Info("api_key_image_count_settled", fields...)
	return true
}
