package admin

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ModelCatalogHandler struct {
	modelCatalogService *service.ModelCatalogService
	userService         *service.UserService
}

func NewModelCatalogHandler(modelCatalogService *service.ModelCatalogService, userService *service.UserService) *ModelCatalogHandler {
	return &ModelCatalogHandler{modelCatalogService: modelCatalogService, userService: userService}
}

func (h *ModelCatalogHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := service.ModelCatalogListFilter{
		Search:        c.Query("search"),
		Provider:      c.Query("provider"),
		Mode:          c.Query("mode"),
		Availability:  c.Query("availability"),
		PricingSource: c.Query("pricing_source"),
		Page:          page,
		PageSize:      pageSize,
	}
	log := logger.FromContext(c.Request.Context()).With(zap.String("component", "handler.admin.model_catalog"))
	log.Info("list model catalog start", zap.Any("filter", filter))
	items, total, err := h.modelCatalogService.ListModels(c.Request.Context(), filter)
	if err != nil {
		log.Warn("list model catalog failed", zap.Error(err))
		response.ErrorFrom(c, err)
		return
	}
	log.Info("list model catalog success", zap.Int("count", len(items)), zap.Int64("total", total))
	response.Paginated(c, items, total, page, pageSize)
}

func (h *ModelCatalogHandler) Detail(c *gin.Context) {
	model := strings.TrimSpace(c.Query("model"))
	log := logger.FromContext(c.Request.Context()).With(
		zap.String("component", "handler.admin.model_catalog"),
		zap.String("model", model),
	)
	log.Info("get model detail start")
	detail, err := h.modelCatalogService.GetModelDetail(c.Request.Context(), model)
	if err != nil {
		log.Warn("get model detail failed", zap.Error(err))
		response.ErrorFrom(c, err)
		return
	}
	log.Info("get model detail success")
	response.Success(c, detail)
}

func (h *ModelCatalogHandler) UpsertPricingOverride(c *gin.Context) {
	var req service.UpsertModelPricingOverrideInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actor := h.resolveActor(c)
	changedFields := pricingOverrideFieldNames(req.ModelCatalogPricing)
	thresholdSummary, aboveThresholdSummary := pricingOverrideTierSummaries(req.ModelCatalogPricing)
	log := logger.FromContext(c.Request.Context()).With(
		zap.String("component", "handler.admin.model_catalog"),
		zap.Int64("admin_user_id", actor.UserID),
		zap.String("admin_email", actor.Email),
		zap.String("model", req.Model),
		zap.Strings("changed_fields", changedFields),
		zap.Strings("tier_thresholds", thresholdSummary),
		zap.Strings("tier_above_threshold_prices", aboveThresholdSummary),
	)
	log.Info("upsert model pricing override start")
	detail, err := h.modelCatalogService.UpsertPricingOverride(c.Request.Context(), actor, req)
	if err != nil {
		log.Warn("upsert model pricing override failed", zap.Error(err))
		response.ErrorFrom(c, err)
		return
	}
	log.Info("upsert model pricing override success")
	response.Success(c, detail)
}

func (h *ModelCatalogHandler) DeletePricingOverride(c *gin.Context) {
	model := strings.TrimSpace(c.Query("model"))
	actor := h.resolveActor(c)
	thresholdSummary, aboveThresholdSummary := h.overrideTierSummaries(c, model)
	log := logger.FromContext(c.Request.Context()).With(
		zap.String("component", "handler.admin.model_catalog"),
		zap.Int64("admin_user_id", actor.UserID),
		zap.String("admin_email", actor.Email),
		zap.String("model", model),
		zap.Strings("tier_thresholds", thresholdSummary),
		zap.Strings("tier_above_threshold_prices", aboveThresholdSummary),
	)
	log.Info("delete model pricing override start")
	if err := h.modelCatalogService.DeletePricingOverride(c.Request.Context(), actor, model); err != nil {
		log.Warn("delete model pricing override failed", zap.Error(err))
		response.ErrorFrom(c, err)
		return
	}
	log.Info("delete model pricing override success")
	response.Success(c, gin.H{"model": service.CanonicalizeModelNameForPricing(model)})
}

func (h *ModelCatalogHandler) resolveActor(c *gin.Context) service.ModelCatalogActor {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		return service.ModelCatalogActor{}
	}
	actor := service.ModelCatalogActor{UserID: subject.UserID}
	user, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err == nil && user != nil {
		actor.Email = user.Email
	}
	return actor
}

func pricingOverrideFieldNames(pricing service.ModelCatalogPricing) []string {
	fields := make([]string, 0, 15)
	appendIfPresent := func(name string, value *float64) {
		if value != nil {
			fields = append(fields, name)
		}
	}
	appendIntIfPresent := func(name string, value *int) {
		if value != nil {
			fields = append(fields, name)
		}
	}
	appendIfPresent("input_cost_per_token", pricing.InputCostPerToken)
	appendIfPresent("input_cost_per_token_priority", pricing.InputCostPerTokenPriority)
	appendIntIfPresent("input_token_threshold", pricing.InputTokenThreshold)
	appendIfPresent("input_cost_per_token_above_threshold", pricing.InputCostPerTokenAboveThreshold)
	appendIfPresent("input_cost_per_token_priority_above_threshold", pricing.InputCostPerTokenPriorityAboveThreshold)
	appendIfPresent("output_cost_per_token", pricing.OutputCostPerToken)
	appendIfPresent("output_cost_per_token_priority", pricing.OutputCostPerTokenPriority)
	appendIntIfPresent("output_token_threshold", pricing.OutputTokenThreshold)
	appendIfPresent("output_cost_per_token_above_threshold", pricing.OutputCostPerTokenAboveThreshold)
	appendIfPresent("output_cost_per_token_priority_above_threshold", pricing.OutputCostPerTokenPriorityAboveThreshold)
	appendIfPresent("cache_creation_input_token_cost", pricing.CacheCreationInputTokenCost)
	appendIfPresent("cache_creation_input_token_cost_above_1hr", pricing.CacheCreationInputTokenCostAbove1hr)
	appendIfPresent("cache_read_input_token_cost", pricing.CacheReadInputTokenCost)
	appendIfPresent("cache_read_input_token_cost_priority", pricing.CacheReadInputTokenCostPriority)
	appendIfPresent("output_cost_per_image", pricing.OutputCostPerImage)
	return fields
}

func pricingOverrideTierSummaries(pricing service.ModelCatalogPricing) ([]string, []string) {
	thresholds := make([]string, 0, 2)
	abovePrices := make([]string, 0, 4)
	appendIntIfPresent := func(name string, value *int) {
		if value != nil {
			thresholds = append(thresholds, fmt.Sprintf("%s=%d", name, *value))
		}
	}
	appendFloatIfPresent := func(name string, value *float64) {
		if value != nil {
			abovePrices = append(abovePrices, fmt.Sprintf("%s=%g", name, *value))
		}
	}
	appendIntIfPresent("input_token_threshold", pricing.InputTokenThreshold)
	appendIntIfPresent("output_token_threshold", pricing.OutputTokenThreshold)
	appendFloatIfPresent("input_cost_per_token_above_threshold", pricing.InputCostPerTokenAboveThreshold)
	appendFloatIfPresent("input_cost_per_token_priority_above_threshold", pricing.InputCostPerTokenPriorityAboveThreshold)
	appendFloatIfPresent("output_cost_per_token_above_threshold", pricing.OutputCostPerTokenAboveThreshold)
	appendFloatIfPresent("output_cost_per_token_priority_above_threshold", pricing.OutputCostPerTokenPriorityAboveThreshold)
	return thresholds, abovePrices
}

func (h *ModelCatalogHandler) overrideTierSummaries(c *gin.Context, model string) ([]string, []string) {
	detail, err := h.modelCatalogService.GetModelDetail(c.Request.Context(), model)
	if err != nil || detail == nil || detail.OverridePricing == nil {
		return nil, nil
	}
	return pricingOverrideTierSummaries(detail.OverridePricing.ModelCatalogPricing)
}
