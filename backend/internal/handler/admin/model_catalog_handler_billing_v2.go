package admin

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *ModelCatalogHandler) ListBillingPricingProviders(c *gin.Context) {
	items, err := h.modelCatalogService.ListBillingPricingProviders(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *ModelCatalogHandler) ListBillingPricingModels(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := service.BillingPricingListFilter{
		Search:    c.Query("search"),
		Provider:  c.Query("provider"),
		Mode:      c.Query("mode"),
		GroupID:   parseOptionalInt64(c.Query("group_id")),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Page:      page,
		PageSize:  pageSize,
	}
	items, total, err := h.modelCatalogService.ListBillingPricingModels(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func parseOptionalInt64(value string) *int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func (h *ModelCatalogHandler) GetBillingPricingDetails(c *gin.Context) {
	var req service.BillingPricingDetailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	items, err := h.modelCatalogService.GetBillingPricingDetails(c.Request.Context(), req.Models, req.GroupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *ModelCatalogHandler) SaveBillingPricingLayer(c *gin.Context) {
	var req service.UpsertBillingPricingLayerInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Model = strings.TrimSpace(c.Param("model"))
	req.Layer = strings.TrimSpace(c.Param("layer"))
	detail, err := h.modelCatalogService.SaveBillingPricingLayer(c.Request.Context(), h.resolveActor(c), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

func (h *ModelCatalogHandler) GetPublicModelCatalogDraft(c *gin.Context) {
	payload, err := h.modelCatalogService.GetPublicModelCatalogDraftPayload(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

func (h *ModelCatalogHandler) SavePublicModelCatalogDraft(c *gin.Context) {
	var draft service.PublicModelCatalogDraft
	if err := c.ShouldBindJSON(&draft); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.modelCatalogService.SavePublicModelCatalogDraft(c.Request.Context(), draft)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ModelCatalogHandler) PublishPublicModelCatalog(c *gin.Context) {
	var draft *service.PublicModelCatalogDraft
	if c.Request != nil && c.Request.Body != nil {
		var payload service.PublicModelCatalogDraft
		if err := c.ShouldBindJSON(&payload); err != nil {
			if !errors.Is(err, io.EOF) {
				response.BadRequest(c, "Invalid request: "+err.Error())
				return
			}
		} else {
			draft = &payload
		}
	}
	result, err := h.modelCatalogService.PublishPublicModelCatalog(c.Request.Context(), h.resolveActor(c), draft)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ModelCatalogHandler) GetPublishedPublicModelCatalogSummary(c *gin.Context) {
	result, err := h.modelCatalogService.GetPublishedPublicModelCatalogSummary(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ModelCatalogHandler) RefreshBillingPricingCatalog(c *gin.Context) {
	result, err := h.modelCatalogService.RefreshBillingPricingCatalog(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ModelCatalogHandler) GetBillingPricingAudit(c *gin.Context) {
	audit, err := h.modelCatalogService.GetBillingPricingAudit(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, audit)
}

func (h *ModelCatalogHandler) CopyBillingPricingOfficialToSale(c *gin.Context) {
	var req service.BillingCopyOfficialToSaleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	items, err := h.modelCatalogService.CopyBillingPricingOfficialToSale(c.Request.Context(), h.resolveActor(c), req.Models)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	protocolruntime.RecordBillingBulkApply("copy_from_official")
	response.Success(c, items)
}

func (h *ModelCatalogHandler) ApplyBillingPricingSaleDiscount(c *gin.Context) {
	var req service.BillingBulkApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	items, err := h.modelCatalogService.ApplyBillingPricingSaleDiscount(c.Request.Context(), h.resolveActor(c), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	protocolruntime.RecordBillingBulkApply("discount")
	response.Success(c, items)
}

func (h *ModelCatalogHandler) ListBillingRules(c *gin.Context) {
	response.Success(c, h.modelCatalogService.ListBillingRules(c.Request.Context()))
}

func (h *ModelCatalogHandler) DeprecatedBillingCenter(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.BillingCenter)
}

func (h *ModelCatalogHandler) DeprecatedUpsertBillingSheet(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.UpsertBillingSheet)
}

func (h *ModelCatalogHandler) DeprecatedDeleteBillingSheet(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.DeleteBillingSheet)
}

func (h *ModelCatalogHandler) DeprecatedUpsertBillingRule(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.UpsertBillingRule)
}

func (h *ModelCatalogHandler) DeprecatedDeleteBillingRule(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.DeleteBillingRule)
}

func (h *ModelCatalogHandler) DeprecatedSimulateBilling(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.SimulateBilling)
}

func (h *ModelCatalogHandler) DeprecatedCopyBillingSheetOfficialToSale(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.CopyBillingSheetOfficialToSale)
}

func (h *ModelCatalogHandler) DeprecatedUpsertOfficialPricingOverride(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.UpsertOfficialPricingOverride)
}

func (h *ModelCatalogHandler) DeprecatedDeleteOfficialPricingOverride(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.DeleteOfficialPricingOverride)
}

func (h *ModelCatalogHandler) DeprecatedUpsertPricingOverride(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.UpsertPricingOverride)
}

func (h *ModelCatalogHandler) DeprecatedDeletePricingOverride(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.DeletePricingOverride)
}

func (h *ModelCatalogHandler) DeprecatedCopyOfficialPricingToSale(c *gin.Context) {
	h.serveDeprecatedBillingAPI(c, h.CopyOfficialPricingToSale)
}

func (h *ModelCatalogHandler) serveDeprecatedBillingAPI(c *gin.Context, next func(*gin.Context)) {
	h.logDeprecatedBillingAPI(c)
	next(c)
}

func (h *ModelCatalogHandler) logDeprecatedBillingAPI(c *gin.Context) {
	path := ""
	if c != nil && c.FullPath() != "" {
		path = c.FullPath()
	} else if c != nil && c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	protocolruntime.RecordBillingDeprecatedAPI(path)
	log := logger.FromContext(c.Request.Context()).With(
		zap.String("component", "handler.admin.model_catalog"),
		zap.String("deprecated_api_path", path),
		zap.Bool("deprecated_api", true),
	)
	log.Warn("deprecated billing api hit")
}
