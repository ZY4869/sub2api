package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type DocumentAIHandler struct {
	documentAIService *service.DocumentAIService
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
		upload, err := readDocumentAIFile(c, "file")
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
		upload, err := readDocumentAIFile(c, "file")
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
	if input.FileBase64 != "" {
		payload, err := base64.StdEncoding.DecodeString(input.FileBase64)
		if err == nil {
			input.FileBytes = payload
			input.FileSize = int64(len(payload))
		}
	}
	return input, nil
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
	if apiKey == nil {
		return 0
	}
	if apiKey.Group != nil && apiKey.Group.ID > 0 && apiKey.Group.Platform == service.PlatformBaiduDocumentAI {
		return apiKey.Group.ID
	}
	for _, binding := range apiKey.GroupBindings {
		if binding.Group != nil && binding.Group.ID > 0 && binding.Group.Platform == service.PlatformBaiduDocumentAI {
			return binding.Group.ID
		}
	}
	return 0
}

func readDocumentAIFile(c *gin.Context, field string) (*documentAIFileUpload, error) {
	fileHeader, err := c.FormFile(field)
	if err != nil {
		return nil, serviceErrBadRequest("document_ai_invalid_request", "file is required")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, serviceErrBadRequest("document_ai_invalid_request", "failed to open uploaded file")
	}
	defer func() { _ = file.Close() }()
	payload, err := io.ReadAll(file)
	if err != nil {
		return nil, serviceErrBadRequest("document_ai_invalid_request", "failed to read uploaded file")
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
