package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type DocumentAIHandler struct {
	documentAIService   *service.DocumentAIService
	billingCacheService *service.BillingCacheService
	apiKeyService       *service.APIKeyService
	subscriptionService *service.SubscriptionService
}

type documentAIFileUpload struct {
	Name        string
	ContentType string
	Size        int64
	Bytes       []byte
}

func NewDocumentAIHandler(documentAIService *service.DocumentAIService) *DocumentAIHandler {
	return &DocumentAIHandler{documentAIService: documentAIService}
}

func ProvideDocumentAIHandler(
	documentAIService *service.DocumentAIService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
) *DocumentAIHandler {
	handler := NewDocumentAIHandler(documentAIService)
	handler.SetBillingServices(billingCacheService, apiKeyService, subscriptionService)
	return handler
}

func (h *DocumentAIHandler) SetBillingServices(
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
) {
	if h == nil {
		return
	}
	h.billingCacheService = billingCacheService
	h.apiKeyService = apiKeyService
	h.subscriptionService = subscriptionService
}

func (h *DocumentAIHandler) ListModels(c *gin.Context) {
	apiKey, groupID, ok := h.requireDocumentAIAccess(c)
	if !ok {
		return
	}
	models, err := h.documentAIService.ListModels(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"provider": service.DocumentAIProviderBaidu,
		"user_id":  apiKey.User.ID,
		"group_id": groupID,
		"models":   models,
	})
}

func (h *DocumentAIHandler) CreateJob(c *gin.Context) {
	apiKey, groupID, ok := h.requireDocumentAIAccess(c)
	if !ok {
		return
	}
	input, err := h.buildSubmitInput(c, apiKey, groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	setDocumentAISubmitRequestFingerprint(c, input)
	if !h.ensureDocumentAIWriteBilling(c, apiKey) {
		return
	}
	defer releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, apiKey)
	job, err := h.documentAIService.SubmitJob(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, buildDocumentAIJobSummaryResponse(job))
}

func (h *DocumentAIHandler) GetJob(c *gin.Context) {
	apiKey, _, ok := h.requireDocumentAIAccess(c)
	if !ok {
		return
	}
	job, err := h.documentAIService.GetJob(c.Request.Context(), c.Param("job_id"), apiKey.User.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "Document AI job not found")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, buildDocumentAIJobSummaryResponse(job))
}

func (h *DocumentAIHandler) GetJobResult(c *gin.Context) {
	apiKey, _, ok := h.requireDocumentAIAccess(c)
	if !ok {
		return
	}
	job, err := h.documentAIService.GetJob(c.Request.Context(), c.Param("job_id"), apiKey.User.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "Document AI job not found")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"provider":          service.DocumentAIProviderBaidu,
		"job_id":            job.JobID,
		"provider_job_id":   job.ProviderJobID,
		"mode":              job.Mode,
		"model":             job.Model,
		"status":            job.Status,
		"provider_result":   decodeJSONString(job.ProviderResultJSON),
		"normalized_result": decodeJSONString(job.NormalizedResultJSON),
		"error_code":        stringPtrValue(job.ErrorCode),
		"error_message":     stringPtrValue(job.ErrorMessage),
		"completed_at":      job.CompletedAt,
	})
}

func (h *DocumentAIHandler) Parse(c *gin.Context) {
	apiKey, groupID, ok := h.requireDocumentAIAccess(c)
	if !ok {
		return
	}
	modelID, action, err := parseDocumentAIModelAction(c.Param("modelAction"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if action != "parse" {
		response.NotFound(c, "Document AI action not found")
		return
	}
	input, err := h.buildDirectInput(c, apiKey, groupID, modelID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	setDocumentAIDirectRequestFingerprint(c, input)
	if !h.ensureDocumentAIWriteBilling(c, apiKey) {
		return
	}
	defer releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, apiKey)
	job, err := h.documentAIService.ParseDirect(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"provider":          service.DocumentAIProviderBaidu,
		"job_id":            job.JobID,
		"provider_job_id":   job.ProviderJobID,
		"mode":              job.Mode,
		"model":             job.Model,
		"status":            job.Status,
		"provider_result":   decodeJSONString(job.ProviderResultJSON),
		"normalized_result": decodeJSONString(job.NormalizedResultJSON),
		"completed_at":      job.CompletedAt,
	})
}

func (h *DocumentAIHandler) buildSubmitInput(c *gin.Context, apiKey *service.APIKey, groupID int64) (service.DocumentAISubmitJobInput, error) {
	input := service.DocumentAISubmitJobInput{
		APIKey:  apiKey,
		GroupID: groupID,
	}
	if strings.HasPrefix(strings.ToLower(c.ContentType()), "multipart/form-data") {
		upload, err := readDocumentAIFile(c, "file", h.documentAIService.UploadMaxBytes())
		if err != nil {
			return input, err
		}
		options, err := parseDocumentAIOptionsField(c.PostForm("options"))
		if err != nil {
			return input, err
		}
		input.Model = c.PostForm("model")
		input.SourceType = service.DocumentAISourceTypeFile
		input.FileName = upload.Name
		input.ContentType = upload.ContentType
		input.FileSize = upload.Size
		input.FileBytes = upload.Bytes
		input.Options = options
		return input, nil
	}
	var req struct {
		Model   string         `json:"model"`
		FileURL string         `json:"file_url"`
		Options map[string]any `json:"options"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return input, serviceErrBadRequest("document_ai_invalid_request", "invalid document ai submit request")
	}
	input.Model = req.Model
	input.SourceType = service.DocumentAISourceTypeFileURL
	input.FileURL = req.FileURL
	input.Options = req.Options
	return input, nil
}

func (h *DocumentAIHandler) buildDirectInput(c *gin.Context, apiKey *service.APIKey, groupID int64, modelID string) (service.DocumentAIParseDirectInput, error) {
	input := service.DocumentAIParseDirectInput{
		APIKey:  apiKey,
		GroupID: groupID,
		Model:   modelID,
	}
	if strings.HasPrefix(strings.ToLower(c.ContentType()), "multipart/form-data") {
		upload, err := readDocumentAIFile(c, "file", h.documentAIService.UploadMaxBytes())
		if err != nil {
			return input, err
		}
		options, err := parseDocumentAIOptionsField(c.PostForm("options"))
		if err != nil {
			return input, err
		}
		input.SourceType = service.DocumentAISourceTypeFile
		input.FileType = c.PostForm("file_type")
		input.FileName = upload.Name
		input.ContentType = upload.ContentType
		input.FileSize = upload.Size
		input.FileBytes = upload.Bytes
		input.Options = options
		return input, nil
	}
	var req struct {
		FileBase64 string         `json:"file_base64"`
		FileType   string         `json:"file_type"`
		Options    map[string]any `json:"options"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return input, serviceErrBadRequest("document_ai_invalid_request", "invalid document ai direct request")
	}
	input.SourceType = service.DocumentAISourceTypeFileBase64
	input.FileBase64 = strings.TrimSpace(req.FileBase64)
	input.FileType = req.FileType
	input.Options = req.Options
	return input, nil
}

type documentAIRequestFingerprintPayload struct {
	Mode           string         `json:"mode"`
	GroupID        int64          `json:"group_id"`
	Model          string         `json:"model"`
	SourceType     string         `json:"source_type"`
	FileURL        string         `json:"file_url,omitempty"`
	FileType       string         `json:"file_type,omitempty"`
	FileName       string         `json:"file_name,omitempty"`
	ContentType    string         `json:"content_type,omitempty"`
	FileSize       int64          `json:"file_size,omitempty"`
	FileHash       string         `json:"file_hash,omitempty"`
	FileBase64Hash string         `json:"file_base64_hash,omitempty"`
	Options        map[string]any `json:"options,omitempty"`
}

func setDocumentAISubmitRequestFingerprint(c *gin.Context, input service.DocumentAISubmitJobInput) {
	setDocumentAIRequestFingerprint(c, documentAIRequestFingerprintPayload{
		Mode:        service.DocumentAIJobModeAsync,
		GroupID:     input.GroupID,
		Model:       strings.TrimSpace(input.Model),
		SourceType:  strings.TrimSpace(input.SourceType),
		FileURL:     strings.TrimSpace(input.FileURL),
		FileName:    strings.TrimSpace(input.FileName),
		ContentType: strings.TrimSpace(input.ContentType),
		FileSize:    input.FileSize,
		FileHash:    documentAIFileBytesHash(input.FileHash, input.FileBytes),
		Options:     input.Options,
	})
}

func setDocumentAIDirectRequestFingerprint(c *gin.Context, input service.DocumentAIParseDirectInput) {
	setDocumentAIRequestFingerprint(c, documentAIRequestFingerprintPayload{
		Mode:           service.DocumentAIJobModeDirect,
		GroupID:        input.GroupID,
		Model:          strings.TrimSpace(input.Model),
		SourceType:     strings.TrimSpace(input.SourceType),
		FileType:       strings.TrimSpace(input.FileType),
		FileName:       strings.TrimSpace(input.FileName),
		ContentType:    strings.TrimSpace(input.ContentType),
		FileSize:       input.FileSize,
		FileHash:       documentAIFileBytesHash(input.FileHash, input.FileBytes),
		FileBase64Hash: service.HashUsageRequestPayload([]byte(strings.TrimSpace(input.FileBase64))),
		Options:        input.Options,
	})
}

func documentAIFileBytesHash(existing string, payload []byte) string {
	if strings.TrimSpace(existing) != "" {
		return strings.TrimSpace(existing)
	}
	return service.HashUsageRequestPayload(payload)
}

func setDocumentAIRequestFingerprint(c *gin.Context, payload documentAIRequestFingerprintPayload) {
	if c == nil || c.Request == nil {
		return
	}
	fingerprint := hashDocumentAIRequestPayload(payload)
	if fingerprint == "" {
		return
	}
	ctx := context.WithValue(c.Request.Context(), ctxkey.RequestPayloadHash, fingerprint)
	c.Request = c.Request.WithContext(ctx)
}

func hashDocumentAIRequestPayload(payload any) string {
	body, err := json.Marshal(payload)
	if err != nil || len(body) == 0 {
		return ""
	}
	return service.HashUsageRequestPayload(body)
}

func (h *DocumentAIHandler) ensureDocumentAIWriteBilling(c *gin.Context, apiKey *service.APIKey) bool {
	if h == nil || h.billingCacheService == nil {
		return true
	}
	group := resolveDocumentAIGroup(apiKey)
	subscription, err := h.resolveDocumentAISubscription(c, apiKey, group)
	if err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, group, subscription); err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	return true
}

func (h *DocumentAIHandler) resolveDocumentAISubscription(c *gin.Context, apiKey *service.APIKey, group *service.Group) (*service.UserSubscription, error) {
	if group == nil || !group.IsSubscriptionType() || apiKey == nil || apiKey.User == nil {
		return nil, nil
	}
	if subscription, ok := servermiddleware.GetSubscriptionFromContext(c); ok && subscription != nil {
		return subscription, nil
	}
	if h == nil || h.subscriptionService == nil {
		return nil, service.ErrSubscriptionInvalid
	}
	subscription, err := h.subscriptionService.GetActiveSubscription(c.Request.Context(), apiKey.User.ID, group.ID)
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (h *DocumentAIHandler) requireDocumentAIAccess(c *gin.Context) (*service.APIKey, int64, bool) {
	apiKey, ok := servermiddleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.User == nil {
		response.Forbidden(c, "Document AI access requires a valid API key")
		return nil, 0, false
	}
	groupID := resolveDocumentAIGroupID(apiKey)
	if groupID <= 0 {
		response.Forbidden(c, "This API key is not bound to any Baidu Document AI group")
		return nil, 0, false
	}
	return apiKey, groupID, true
}

func resolveDocumentAIGroupID(apiKey *service.APIKey) int64 {
	if group := resolveDocumentAIGroup(apiKey); group != nil {
		return group.ID
	}
	return 0
}

func resolveDocumentAIGroup(apiKey *service.APIKey) *service.Group {
	if apiKey == nil {
		return nil
	}
	if apiKey.Group != nil && apiKey.Group.ID > 0 && apiKey.Group.Platform == service.PlatformBaiduDocumentAI {
		return apiKey.Group
	}
	for _, binding := range apiKey.GroupBindings {
		if binding.Group != nil && binding.Group.ID > 0 && binding.Group.Platform == service.PlatformBaiduDocumentAI {
			return binding.Group
		}
	}
	return nil
}

func readDocumentAIFile(c *gin.Context, field string, maxBytes int64) (*documentAIFileUpload, error) {
	fileHeader, err := c.FormFile(field)
	if err != nil {
		return nil, serviceErrBadRequest("document_ai_invalid_request", "file is required")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, serviceErrBadRequest("document_ai_invalid_request", "failed to open uploaded file")
	}
	defer func() { _ = file.Close() }()
	if maxBytes <= 0 {
		maxBytes = 50 * 1024 * 1024
	}
	payload, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, serviceErrBadRequest("document_ai_invalid_request", "failed to read uploaded file")
	}
	if int64(len(payload)) > maxBytes {
		return nil, serviceErrBadRequest("document_ai_invalid_request", "file exceeds document ai size limit")
	}
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType == "" && len(payload) > 0 {
		contentType = http.DetectContentType(payload)
	}
	return &documentAIFileUpload{
		Name:        fileHeader.Filename,
		ContentType: contentType,
		Size:        int64(len(payload)),
		Bytes:       payload,
	}, nil
}

func parseDocumentAIOptionsField(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var options map[string]any
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		return nil, serviceErrBadRequest("document_ai_invalid_request", "options must be a JSON object")
	}
	return options, nil
}

func parseDocumentAIModelAction(raw string) (string, string, error) {
	trimmed := strings.Trim(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return "", "", errors.New("invalid document ai action")
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid document ai action")
	}
	return parts[0], parts[1], nil
}

func buildDocumentAIJobSummaryResponse(job *service.DocumentAIJob) gin.H {
	if job == nil {
		return gin.H{}
	}
	return gin.H{
		"provider":          service.DocumentAIProviderBaidu,
		"job_id":            job.JobID,
		"provider_job_id":   job.ProviderJobID,
		"provider_batch_id": job.ProviderBatchID,
		"group_id":          job.GroupID,
		"mode":              job.Mode,
		"model":             job.Model,
		"source_type":       job.SourceType,
		"status":            job.Status,
		"error_code":        stringPtrValue(job.ErrorCode),
		"error_message":     stringPtrValue(job.ErrorMessage),
		"created_at":        job.CreatedAt,
		"updated_at":        job.UpdatedAt,
		"completed_at":      job.CompletedAt,
	}
}

func decodeJSONString(raw *string) any {
	value := strings.TrimSpace(stringPtrValue(raw))
	if value == "" {
		return nil
	}
	var out any
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return value
	}
	return out
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func serviceErrBadRequest(reason, message string) error {
	return infraerrors.BadRequest(reason, message)
}
