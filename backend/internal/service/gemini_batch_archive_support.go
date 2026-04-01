package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func (s *GeminiMessagesCompatService) getGoogleBatchArchiveSettings(ctx context.Context) *GoogleBatchArchiveSettings {
	if s == nil || s.settingService == nil {
		return DefaultGoogleBatchArchiveSettings()
	}
	settings, err := s.settingService.GetGoogleBatchArchiveSettings(ctx)
	if err != nil || settings == nil {
		return DefaultGoogleBatchArchiveSettings()
	}
	return NormalizeGoogleBatchArchiveSettings(settings)
}

func (s *GeminiMessagesCompatService) upsertGoogleBatchArchiveJob(ctx context.Context, job *GoogleBatchArchiveJob) error {
	if s == nil || s.googleBatchArchiveJobRepo == nil || job == nil {
		return nil
	}
	if job.MetadataJSON == nil {
		job.MetadataJSON = map[string]any{}
	}
	return s.googleBatchArchiveJobRepo.Upsert(ctx, job)
}

func (s *GeminiMessagesCompatService) upsertGoogleBatchArchiveObject(ctx context.Context, object *GoogleBatchArchiveObject) error {
	if s == nil || s.googleBatchArchiveObjectRepo == nil || object == nil {
		return nil
	}
	if object.MetadataJSON == nil {
		object.MetadataJSON = map[string]any{}
	}
	if object.StorageBackend == "" {
		object.StorageBackend = GoogleBatchArchiveStorageBackendLocalFS
	}
	return s.googleBatchArchiveObjectRepo.Upsert(ctx, object)
}

func (s *GeminiMessagesCompatService) getGoogleBatchArchiveJobByPublicBatchName(ctx context.Context, publicBatchName string) (*GoogleBatchArchiveJob, error) {
	if s == nil || s.googleBatchArchiveJobRepo == nil {
		return nil, nil
	}
	job, err := s.googleBatchArchiveJobRepo.GetByPublicBatchName(ctx, strings.TrimSpace(publicBatchName))
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *GeminiMessagesCompatService) getGoogleBatchArchiveObject(ctx context.Context, kind string, name string) (*GoogleBatchArchiveObject, error) {
	if s == nil || s.googleBatchArchiveObjectRepo == nil {
		return nil, nil
	}
	object, err := s.googleBatchArchiveObjectRepo.GetByPublicResource(ctx, strings.TrimSpace(kind), strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	return object, nil
}

func (s *GeminiMessagesCompatService) storeGoogleBatchArchiveObjectBytes(ctx context.Context, settings *GoogleBatchArchiveSettings, job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject, filename string, contentType string, payload []byte) error {
	if s == nil || s.googleBatchArchiveStorage == nil || job == nil || object == nil {
		return nil
	}
	relativePath, size, sha256, err := s.googleBatchArchiveStorage.StoreBytes(ctx, settings, job, filename, payload)
	if err != nil {
		return err
	}
	object.RelativePath = relativePath
	object.SizeBytes = size
	object.SHA256 = sha256
	object.ContentType = strings.TrimSpace(contentType)
	if object.ContentType == "" {
		object.ContentType = "application/octet-stream"
	}
	if err := s.upsertGoogleBatchArchiveObject(ctx, object); err != nil {
		return err
	}
	if object.IsResultPayload && job.ArchiveState != GoogleBatchArchiveLifecycleArchived {
		job.ArchiveState = GoogleBatchArchiveLifecycleArchived
		if err := s.upsertGoogleBatchArchiveJob(ctx, job); err != nil {
			return err
		}
	}
	return s.persistGoogleBatchArchiveManifest(ctx, settings, job)
}

func (s *GeminiMessagesCompatService) storeGoogleBatchArchiveObjectReader(ctx context.Context, settings *GoogleBatchArchiveSettings, job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject, filename string, contentType string, reader io.Reader) error {
	if s == nil || s.googleBatchArchiveStorage == nil || job == nil || object == nil {
		return nil
	}
	relativePath, size, sha256, err := s.googleBatchArchiveStorage.StoreReader(ctx, settings, job, filename, reader)
	if err != nil {
		return err
	}
	object.RelativePath = relativePath
	object.SizeBytes = size
	object.SHA256 = sha256
	object.ContentType = strings.TrimSpace(contentType)
	if object.ContentType == "" {
		object.ContentType = "application/octet-stream"
	}
	if err := s.upsertGoogleBatchArchiveObject(ctx, object); err != nil {
		return err
	}
	if object.IsResultPayload && job.ArchiveState != GoogleBatchArchiveLifecycleArchived {
		job.ArchiveState = GoogleBatchArchiveLifecycleArchived
		if err := s.upsertGoogleBatchArchiveJob(ctx, job); err != nil {
			return err
		}
	}
	return s.persistGoogleBatchArchiveManifest(ctx, settings, job)
}

func (s *GeminiMessagesCompatService) storeGoogleBatchSnapshot(ctx context.Context, settings *GoogleBatchArchiveSettings, job *GoogleBatchArchiveJob, payload []byte) error {
	if job == nil || len(payload) == 0 {
		return nil
	}
	object := &GoogleBatchArchiveObject{
		JobID:              job.ID,
		PublicResourceKind: GoogleBatchArchiveResourceKindBatch,
		PublicResourceName: strings.TrimSpace(job.PublicBatchName),
		ContentType:        "application/json",
		MetadataJSON:       map[string]any{"snapshot_type": "batch_snapshot"},
	}
	if err := s.storeGoogleBatchArchiveObjectBytes(ctx, settings, job, object, googleBatchArchiveSnapshotFilename, "application/json", payload); err != nil {
		return err
	}
	if job.MetadataJSON == nil {
		job.MetadataJSON = map[string]any{}
	}
	job.MetadataJSON["batch_snapshot_relative_path"] = object.RelativePath
	if err := s.upsertGoogleBatchArchiveJob(ctx, job); err != nil {
		return err
	}
	return s.persistGoogleBatchArchiveManifest(ctx, settings, job)
}

func (s *GeminiMessagesCompatService) buildGoogleBatchJSONResult(status int, body []byte) *UpstreamHTTPResult {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	return &UpstreamHTTPResult{StatusCode: status, Headers: headers, Body: body}
}

func (s *GeminiMessagesCompatService) buildGoogleBatchBinaryResult(contentType string, filename string, body []byte) *UpstreamHTTPResult {
	headers := make(http.Header)
	if strings.TrimSpace(contentType) != "" {
		headers.Set("Content-Type", contentType)
	} else {
		headers.Set("Content-Type", "application/octet-stream")
	}
	if strings.TrimSpace(filename) != "" {
		headers.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	}
	return &UpstreamHTTPResult{StatusCode: http.StatusOK, Headers: headers, Body: body}
}

func (s *GeminiMessagesCompatService) buildGoogleBatchBinaryStreamResult(contentType string, filename string, body io.ReadCloser, contentLength int64) *UpstreamHTTPStreamResult {
	headers := make(http.Header)
	if strings.TrimSpace(contentType) != "" {
		headers.Set("Content-Type", contentType)
	} else {
		headers.Set("Content-Type", "application/octet-stream")
	}
	if strings.TrimSpace(filename) != "" {
		headers.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	}
	return &UpstreamHTTPStreamResult{
		StatusCode:    http.StatusOK,
		Headers:       headers,
		Body:          body,
		ContentLength: contentLength,
	}
}

func (s *GeminiMessagesCompatService) openGoogleBatchArchiveObjectStreamResult(settings *GoogleBatchArchiveSettings, object *GoogleBatchArchiveObject, filename string) (*UpstreamHTTPStreamResult, error) {
	if s == nil || s.googleBatchArchiveStorage == nil || object == nil || strings.TrimSpace(object.RelativePath) == "" {
		return nil, nil
	}
	file, info, err := s.googleBatchArchiveStorage.OpenReader(settings, object.RelativePath)
	if err != nil {
		return nil, err
	}
	contentLength := int64(-1)
	if info != nil {
		contentLength = info.Size()
	}
	return s.buildGoogleBatchBinaryStreamResult(object.ContentType, filename, file, contentLength), nil
}

func (s *GeminiMessagesCompatService) recordGoogleBatchUsageEvent(ctx context.Context, input GoogleBatchForwardInput, account *Account, requestedModel string, operationType string, chargeSource string, tokens UsageTokens, cost *CostBreakdown, requestID string) error {
	if s == nil || s.usageLogRepo == nil || account == nil {
		return nil
	}
	if strings.TrimSpace(requestID) == "" {
		requestID = "google-batch:" + operationType + ":" + generateRequestID()
	}
	totalCost := 0.0
	actualCost := 0.0
	inputCost := 0.0
	outputCost := 0.0
	cacheCreationCost := 0.0
	cacheReadCost := 0.0
	if cost != nil {
		totalCost = cost.TotalCost
		actualCost = cost.ActualCost
		inputCost = cost.InputCost
		outputCost = cost.OutputCost
		cacheCreationCost = cost.CacheCreationCost
		cacheReadCost = cost.CacheReadCost
	}
	usageLog := &UsageLog{
		UserID:                input.UserID,
		APIKeyID:              input.APIKeyID,
		AccountID:             account.ID,
		RequestID:             requestID,
		Model:                 strings.TrimSpace(requestedModel),
		RequestedModel:        strings.TrimSpace(requestedModel),
		GroupID:               input.GroupID,
		SubscriptionID:        input.SubscriptionID,
		InputTokens:           tokens.InputTokens,
		OutputTokens:          tokens.OutputTokens,
		CacheCreationTokens:   tokens.CacheCreationTokens,
		CacheReadTokens:       tokens.CacheReadTokens,
		CacheCreation5mTokens: tokens.CacheCreation5mTokens,
		CacheCreation1hTokens: tokens.CacheCreation1hTokens,
		InputCost:             inputCost,
		OutputCost:            outputCost,
		CacheCreationCost:     cacheCreationCost,
		CacheReadCost:         cacheReadCost,
		TotalCost:             totalCost,
		ActualCost:            actualCost,
		RateMultiplier:        account.BillingRateMultiplier(),
		AccountRateMultiplier: usageLogFloat64Ptr(account.BillingRateMultiplier()),
		BillingType:           input.BillingType,
		RequestType:           RequestTypeSync,
		Status:                UsageLogStatusSucceeded,
		OperationType:         optionalTrimmedStringPtr(operationType),
		ChargeSource:          optionalTrimmedStringPtr(chargeSource),
		InboundEndpoint:       optionalTrimmedStringPtr(strings.TrimSpace(input.Path)),
		UpstreamEndpoint:      optionalTrimmedStringPtr(strings.TrimSpace(input.Path)),
		UpstreamURL:           optionalTrimmedStringPtr(ResolveUsageLogUpstreamURL(account, "")),
		UpstreamService:       optionalTrimmedStringPtr(ResolveUsageLogUpstreamService(account, "")),
		CreatedAt:             time.Now(),
	}
	if _, err := s.usageLogRepo.Create(ctx, usageLog); err != nil {
		return err
	}
	if s.usageBillingRepo == nil || input.APIKey == nil || actualCost <= 0 {
		return nil
	}
	cmd := &UsageBillingCommand{
		RequestID:           requestID,
		APIKeyID:            input.APIKeyID,
		RequestFingerprint:  operationType + "|" + strings.TrimSpace(requestedModel) + "|" + strings.TrimSpace(chargeSource),
		RequestPayloadHash:  HashUsageRequestPayload(input.Body),
		UserID:              input.UserID,
		AccountID:           account.ID,
		GroupID:             input.GroupID,
		SubscriptionID:      input.SubscriptionID,
		AccountType:         account.Type,
		Model:               strings.TrimSpace(requestedModel),
		BillingType:         input.BillingType,
		InputTokens:         tokens.InputTokens,
		OutputTokens:        tokens.OutputTokens,
		CacheCreationTokens: tokens.CacheCreationTokens,
		CacheReadTokens:     tokens.CacheReadTokens,
		AccountQuotaCost:    totalCost * account.BillingRateMultiplier(),
	}
	switch input.BillingType {
	case BillingTypeSubscription:
		if input.SubscriptionID != nil {
			cmd.SubscriptionCost = actualCost
		}
	default:
		cmd.BalanceCost = actualCost
	}
	if input.APIKey.Quota > 0 {
		cmd.APIKeyQuotaCost = actualCost
	}
	if input.APIKey.GroupID != nil {
		cmd.APIKeyGroupQuotaCost = actualCost
		cmd.GroupID = input.APIKey.GroupID
	}
	if input.APIKey.HasRateLimits() {
		cmd.APIKeyRateLimitCost = actualCost
	}
	cmd.Normalize()
	_, err := s.usageBillingRepo.Apply(ctx, cmd)
	return err
}

func usageLogFloat64Ptr(value float64) *float64 {
	return &value
}

func googleBatchUsageFromJSONLine(line []byte) UsageTokens {
	if len(line) == 0 {
		return UsageTokens{}
	}
	usage := extractGeminiUsage(line)
	if usage == nil {
		usage = extractGeminiUsage([]byte(gjson.GetBytes(line, "response").Raw))
	}
	if usage == nil {
		return UsageTokens{}
	}
	return UsageTokens{
		InputTokens:     usage.InputTokens,
		OutputTokens:    usage.OutputTokens,
		CacheReadTokens: usage.CacheReadInputTokens,
	}
}

func googleBatchAggregateUsageFromJSONL(payload []byte) UsageTokens {
	tokens, _ := googleBatchAggregateUsageFromReader(strings.NewReader(string(payload)))
	return tokens
}

func googleBatchAggregateUsageFromReader(reader io.Reader) (UsageTokens, error) {
	var tokens UsageTokens
	err := walkJSONLLines(reader, func(_ int, line []byte) error {
		current := googleBatchUsageFromJSONLine(line)
		tokens.InputTokens += current.InputTokens
		tokens.OutputTokens += current.OutputTokens
		tokens.CacheCreationTokens += current.CacheCreationTokens
		tokens.CacheReadTokens += current.CacheReadTokens
		tokens.CacheCreation5mTokens += current.CacheCreation5mTokens
		tokens.CacheCreation1hTokens += current.CacheCreation1hTokens
		return nil
	})
	return tokens, err
}

func googleBatchCostWithDiscount(base *CostBreakdown, factor float64) *CostBreakdown {
	if base == nil {
		return &CostBreakdown{}
	}
	if factor <= 0 {
		factor = 1
	}
	return &CostBreakdown{
		InputCost:         base.InputCost * factor,
		OutputCost:        base.OutputCost * factor,
		CacheCreationCost: base.CacheCreationCost * factor,
		CacheReadCost:     base.CacheReadCost * factor,
		TotalCost:         base.TotalCost * factor,
		ActualCost:        base.ActualCost * factor,
	}
}

func (s *GeminiMessagesCompatService) calculateGoogleBatchSettlementCost(requestedModel string, account *Account, tokens UsageTokens) *CostBreakdown {
	if s == nil || s.billingService == nil || account == nil {
		return nil
	}
	base, err := s.billingService.CalculateCost(strings.TrimSpace(requestedModel), tokens, account.BillingRateMultiplier())
	if err != nil {
		return nil
	}
	return googleBatchCostWithDiscount(base, 0.5)
}

func archivedBatchSnapshotMetadata(job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject) map[string]any {
	if object != nil && len(object.MetadataJSON) > 0 {
		return object.MetadataJSON
	}
	if job != nil && len(job.MetadataJSON) > 0 {
		return job.MetadataJSON
	}
	return map[string]any{}
}

func buildArchivedAIStudioBatchPayload(job *GoogleBatchArchiveJob, snapshotBody []byte) []byte {
	if len(snapshotBody) > 0 {
		return snapshotBody
	}
	if job == nil {
		payload, _ := json.Marshal(map[string]any{})
		return payload
	}
	state := strings.TrimSpace(job.State)
	if state == "" {
		state = GoogleBatchArchiveStateUnknown
	}
	payload := map[string]any{
		"name":  job.PublicBatchName,
		"state": state,
	}
	if strings.TrimSpace(job.RequestedModel) != "" {
		payload["model"] = "models/" + strings.TrimPrefix(strings.TrimSpace(job.RequestedModel), "models/")
	}
	if createdAt := job.CreatedAt.UTC(); !createdAt.IsZero() {
		payload["createTime"] = createdAt.Format(time.RFC3339)
	}
	if updatedAt := job.UpdatedAt.UTC(); !updatedAt.IsZero() {
		payload["updateTime"] = updatedAt.Format(time.RFC3339)
	}
	if publicResultFileName, ok := metadataString(job.MetadataJSON, "public_result_file_name"); ok {
		payload["dest"] = map[string]any{"fileName": publicResultFileName}
	}
	body, _ := json.Marshal(payload)
	return body
}

func buildArchivedAIStudioFilePayload(job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject) []byte {
	payload := map[string]any{}
	if object != nil {
		payload["name"] = object.PublicResourceName
		if strings.TrimSpace(object.ContentType) != "" {
			payload["mimeType"] = object.ContentType
		}
		if object.SizeBytes > 0 {
			payload["sizeBytes"] = strconv.FormatInt(object.SizeBytes, 10)
		}
		if strings.TrimSpace(object.SHA256) != "" {
			payload["sha256Hash"] = object.SHA256
		}
	}
	if job != nil {
		if createdAt := job.CreatedAt.UTC(); !createdAt.IsZero() {
			payload["createTime"] = createdAt.Format(time.RFC3339)
		}
		if job.OfficialExpiresAt != nil {
			payload["expirationTime"] = job.OfficialExpiresAt.UTC().Format(time.RFC3339)
		}
	}
	if _, ok := payload["name"]; !ok {
		payload["name"] = ""
	}
	body, _ := json.Marshal(payload)
	return body
}

func metadataInt64(metadata map[string]any, key string) (int64, bool) {
	if metadata == nil {
		return 0, false
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		value, err := typed.Int64()
		return value, err == nil
	case string:
		value, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return value, err == nil
	default:
		return 0, false
	}
}

func metadataString(metadata map[string]any, key string) (string, bool) {
	if metadata == nil {
		return "", false
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		return trimmed, trimmed != ""
	default:
		trimmed := strings.TrimSpace(gjson.ParseBytes(mustJSONBytes(typed)).String())
		return trimmed, trimmed != ""
	}
}

func metadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	return parseExtraBool(metadata[key])
}

func mustJSONBytes(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func (s *GeminiMessagesCompatService) enrichBindingMetadata(ctx context.Context, resourceKind string, resourceName string, merge map[string]any) error {
	if s == nil || s.resourceBindingRepo == nil || strings.TrimSpace(resourceKind) == "" || strings.TrimSpace(resourceName) == "" || len(merge) == 0 {
		return nil
	}
	binding, err := s.resourceBindingRepo.Get(ctx, resourceKind, resourceName)
	if err != nil || binding == nil {
		return err
	}
	if binding.MetadataJSON == nil {
		binding.MetadataJSON = map[string]any{}
	}
	for key, value := range merge {
		binding.MetadataJSON[key] = value
	}
	binding.MetadataJSON = normalizeGoogleBatchBindingMetadata(binding.MetadataJSON)
	return s.resourceBindingRepo.Upsert(ctx, binding)
}
