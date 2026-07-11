package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) SubmitImageBatch(c *gin.Context) {
	apiKey, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		h.imageBatchError(c, service.ErrIdempotencyKeyRequired)
		return
	}
	var req service.ImageBatchSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.imageBatchError(c, service.ErrImageBatchInvalidRequest)
		return
	}
	h.executeImageBatchIdempotent(c, "gateway.image_batches.submit", subject.UserID, req, func(ctx context.Context) (any, error) {
		return h.imageBatchService.Submit(ctx, apiKey, req, idempotencyKey)
	})
}

func (h *GatewayHandler) ListImageBatches(c *gin.Context) {
	_, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	before := parseImageBatchBefore(c.Query("before"))
	jobs, err := h.imageBatchService.ListJobs(c.Request.Context(), subject.UserID, parseImageBatchLimit(c.Query("limit"), 50), before)
	if err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": jobs})
}

func (h *GatewayHandler) ListImageBatchModels(c *gin.Context) {
	apiKey, _, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	models, err := h.imageBatchService.ListModels(c.Request.Context(), apiKey)
	if err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": models})
}

func (h *GatewayHandler) GetImageBatchOrModels(c *gin.Context) {
	if strings.EqualFold(strings.TrimSpace(c.Param("id")), "models") {
		h.ListImageBatchModels(c)
		return
	}
	h.GetImageBatch(c)
}

func (h *GatewayHandler) GetImageBatch(c *gin.Context) {
	_, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	job, err := h.imageBatchService.GetJob(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *GatewayHandler) ListImageBatchItems(c *gin.Context) {
	_, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	items, err := h.imageBatchService.ListItems(c.Request.Context(), subject.UserID, c.Param("id"), parseImageBatchLimit(c.Query("limit"), 100))
	if err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": items})
}

func (h *GatewayHandler) GetImageBatchItemContent(c *gin.Context) {
	_, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	content, err := h.imageBatchService.GetItemContent(c.Request.Context(), subject.UserID, c.Param("id"), c.Param("custom_id"))
	if err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(content.FileName))
	c.Data(http.StatusOK, content.ContentType, content.Body)
}

func (h *GatewayHandler) DownloadImageBatch(c *gin.Context) {
	_, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	content, err := h.imageBatchService.DownloadZip(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(content.FileName))
	c.Data(http.StatusOK, content.ContentType, content.Body)
}

func (h *GatewayHandler) CancelImageBatch(c *gin.Context) {
	_, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	job, err := h.imageBatchService.Cancel(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *GatewayHandler) DeleteImageBatch(c *gin.Context) {
	_, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	if err := h.imageBatchService.Delete(c.Request.Context(), subject.UserID, c.Param("id")); err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "deleted": true})
}

func (h *GatewayHandler) DeleteImageBatchOutputs(c *gin.Context) {
	_, subject, ok := h.imageBatchAuth(c)
	if !ok {
		return
	}
	if err := h.imageBatchService.DeleteOutputs(c.Request.Context(), subject.UserID, c.Param("id")); err != nil {
		h.imageBatchError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "outputs_deleted": true})
}

func (h *GatewayHandler) imageBatchAuth(c *gin.Context) (*service.APIKey, middleware2.AuthSubject, bool) {
	var subject middleware2.AuthSubject
	if h == nil || h.imageBatchService == nil {
		h.imageBatchError(c, service.ErrImageBatchUnavailable)
		return nil, subject, false
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return nil, subject, false
	}
	subject, ok = middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return nil, subject, false
	}
	return apiKey, subject, true
}

func (h *GatewayHandler) executeImageBatchIdempotent(c *gin.Context, scope string, userID int64, payload any, execute func(context.Context) (any, error)) {
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		data, err := execute(c.Request.Context())
		if err != nil {
			h.imageBatchError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, data)
		return
	}
	result, err := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{
		Scope:          scope,
		ActorScope:     "user:" + strconv.FormatInt(userID, 10),
		Method:         c.Request.Method,
		Route:          c.FullPath(),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Payload:        payload,
		RequireKey:     true,
		TTL:            service.DefaultWriteIdempotencyTTL(),
	}, execute)
	if err != nil {
		if retryAfter := service.RetryAfterSecondsFromError(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.imageBatchError(c, err)
		return
	}
	if result != nil && result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	if result == nil {
		c.JSON(http.StatusAccepted, gin.H{})
		return
	}
	c.JSON(http.StatusAccepted, result.Data)
}

func (h *GatewayHandler) imageBatchError(c *gin.Context, err error) {
	status, body := infraerrors.ToHTTP(err)
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	message := strings.TrimSpace(body.Message)
	if message == "" {
		message = "Image batch request failed. Please retry or contact support."
	}
	code := strings.TrimSpace(body.Reason)
	if code == "" {
		code = "IMAGE_BATCH_ERROR"
	}
	payload := gin.H{
		"type": "error",
		"error": gin.H{
			"type":    imageBatchErrorType(status),
			"message": message,
			"code":    code,
		},
	}
	if status >= http.StatusInternalServerError || errors.Is(err, service.ErrIdempotencyStoreUnavail) {
		errorID := service.GenerateSafeRequestID()
		payload["error"].(gin.H)["error_id"] = errorID
		payload["metadata"] = gin.H{"error_id": errorID}
	}
	c.JSON(status, payload)
}

func imageBatchErrorType(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusNotFound:
		return "not_found_error"
	case http.StatusConflict:
		return "conflict_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	default:
		if status >= http.StatusInternalServerError {
			return "api_error"
		}
		return "invalid_request_error"
	}
}

func parseImageBatchLimit(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 1000 {
		return 1000
	}
	return value
}

func parseImageBatchBefore(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
