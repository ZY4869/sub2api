package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const defaultPublicModelCatalogRevalidationInterval = 24 * time.Hour

type PublicModelCatalogRevalidationRunner struct {
	modelCatalogService *ModelCatalogService
	interval            time.Duration
	stopCh              chan struct{}
	startOnce           sync.Once
	stopOnce            sync.Once
	wg                  sync.WaitGroup
}

func NewPublicModelCatalogRevalidationRunner(
	modelCatalogService *ModelCatalogService,
	interval time.Duration,
) *PublicModelCatalogRevalidationRunner {
	if interval <= 0 {
		interval = defaultPublicModelCatalogRevalidationInterval
	}
	return &PublicModelCatalogRevalidationRunner{
		modelCatalogService: modelCatalogService,
		interval:            interval,
		stopCh:              make(chan struct{}),
	}
}

func ProvidePublicModelCatalogRevalidationRunner(
	modelCatalogService *ModelCatalogService,
	gatewayService *GatewayService,
) *PublicModelCatalogRevalidationRunner {
	_ = gatewayService
	runner := NewPublicModelCatalogRevalidationRunner(modelCatalogService, defaultPublicModelCatalogRevalidationInterval)
	runner.Start()
	return runner
}

func (r *PublicModelCatalogRevalidationRunner) Start() {
	if r == nil || r.modelCatalogService == nil || r.interval <= 0 {
		return
	}
	r.startOnce.Do(func() {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			ticker := time.NewTicker(r.interval)
			defer ticker.Stop()

			r.runOnce(context.Background())
			for {
				select {
				case <-ticker.C:
					r.runOnce(context.Background())
				case <-r.stopCh:
					return
				}
			}
		}()
	})
}

func (r *PublicModelCatalogRevalidationRunner) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
	r.wg.Wait()
}

func (r *PublicModelCatalogRevalidationRunner) runOnce(ctx context.Context) {
	if r == nil || r.modelCatalogService == nil {
		return
	}
	requestID := "system:public-model-catalog-revalidation:" + GenerateSafeRequestID()
	ctx = context.WithValue(ctx, ctxkey.RequestID, requestID)
	if !r.modelCatalogService.IsPublicModelCatalogAutoRevalidationEnabled(ctx) {
		logger.FromContext(ctx).Info(
			"public model catalog auto revalidation skipped",
			zap.String("component", "service.model_catalog"),
			zap.String("request_id", requestID),
			zap.String("actor", "system"),
			zap.String("reason", "disabled"),
		)
		return
	}
	result, err := r.modelCatalogService.RevalidatePublishedPublicModelCatalog(ctx, ModelCatalogActor{Email: "system"})
	if err != nil {
		fields := []zap.Field{
			zap.String("component", "service.model_catalog"),
			zap.String("request_id", requestID),
			zap.String("actor", "system"),
			zap.Error(err),
		}
		if appErr := errors.Unwrap(err); appErr != nil {
			fields = append(fields, zap.String("cause", appErr.Error()))
		}
		logger.FromContext(ctx).Warn("public model catalog auto revalidation failed", fields...)
		return
	}
	fields := []zap.Field{
		zap.String("component", "service.model_catalog"),
		zap.String("request_id", requestID),
		zap.String("actor", "system"),
		zap.Int("model_count", result.ModelCount),
		zap.Int("stale_count", result.StaleCount),
		zap.Any("reason_counts", result.Reasons),
	}
	logger.FromContext(ctx).Info("public model catalog auto revalidation completed", fields...)
}
