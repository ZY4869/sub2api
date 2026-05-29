package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

func bindingGroupIDPtr(binding APIKeyGroupBinding) *int64 {
	if binding.GroupID <= 0 {
		return nil
	}
	id := binding.GroupID
	return &id
}

func recordPublicCatalogRouteMiss(ctx context.Context, apiKey *APIKey, groupID *int64, publicModelID string, platform string) {
	protocolruntime.RecordBillingResolverFallback("public_catalog_route_miss")
	fields := publicCatalogLogFields(ctx, nil, groupID, apiKey)
	fields = append(fields,
		zap.String("public_model_id", strings.TrimSpace(publicModelID)),
		zap.String("platform", strings.TrimSpace(platform)),
	)
	logger.FromContext(ctx).Warn("public model catalog route miss", fields...)
}
