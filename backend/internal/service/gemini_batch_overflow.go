package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/tidwall/gjson"
)

type googleBatchOverflowSelection struct {
	sourceAccount *Account
	targetAccount *Account
	gcsProfile    *GoogleBatchGCSProfile
}

type googleBatchGCSObject struct {
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        string `json:"size"`
}

type googleBatchGCSListResponse struct {
	Items []googleBatchGCSObject `json:"items"`
}

func (s *GeminiMessagesCompatService) isGoogleBatchMixedOverflowEnabled(ctx context.Context, groupID *int64) bool {
	if geminiMixedProtocolEnabledFromContext(ctx) {
		return true
	}
	if groupID == nil || *groupID <= 0 || s == nil || s.groupRepo == nil {
		return false
	}
	group, err := s.groupRepo.GetByIDLite(ctx, *groupID)
	return err == nil && group != nil && group.Platform == PlatformGemini && group.GeminiMixedProtocolEnabled
}

func (s *GeminiMessagesCompatService) resolveAIStudioOverflowSelection(ctx context.Context, input GoogleBatchForwardInput, selector *vertexBatchSelector) (*googleBatchOverflowSelection, error) {
	if s == nil || !s.isGoogleBatchMixedOverflowEnabled(ctx, input.GroupID) {
		return nil, nil
	}
	sourceAccount := s.resolveAIStudioOverflowSourceAccount(ctx, input, selector)
	if sourceAccount == nil || !sourceAccount.AllowVertexBatchOverflow() {
		return nil, nil
	}
	targetAccount, err := s.selectOverflowVertexAccount(ctx, input.GroupID)
	if err != nil || targetAccount == nil {
		return nil, err
	}
	gcsProfile, err := s.getActiveGoogleBatchGCSProfile(ctx)
	if err != nil || gcsProfile == nil {
		return nil, err
	}
	return &googleBatchOverflowSelection{
		sourceAccount: sourceAccount,
		targetAccount: targetAccount,
		gcsProfile:    gcsProfile,
	}, nil
}

func (s *GeminiMessagesCompatService) resolveAIStudioOverflowSourceAccount(ctx context.Context, input GoogleBatchForwardInput, selector *vertexBatchSelector) *Account {
	if selector != nil && selector.accountID != nil && *selector.accountID > 0 {
		if account, _ := s.accountRepo.GetByID(ctx, *selector.accountID); account != nil && SupportsAIStudioBatch(account) {
			return account
		}
	}
	accounts, err := s.listSchedulableAccountsOnce(ctx, input.GroupID, PlatformGemini, false)
	if err != nil {
		return nil
	}
	for i := range accounts {
		account := &accounts[i]
		if selector != nil && selector.accountID != nil && *selector.accountID > 0 && account.ID != *selector.accountID {
			continue
		}
		if SupportsAIStudioBatch(account) {
			return account
		}
	}
	return nil
}

func (s *GeminiMessagesCompatService) selectOverflowVertexAccount(ctx context.Context, groupID *int64) (*Account, error) {
	accounts, err := s.listEligibleGoogleBatchAccounts(ctx, groupID, googleBatchTargetVertex, nil)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if account != nil && account.AcceptAIStudioBatchOverflow() {
			return account, nil
		}
	}
	return nil, nil
}

func (s *GeminiMessagesCompatService) getActiveGoogleBatchGCSProfile(ctx context.Context) (*GoogleBatchGCSProfile, error) {
	if s == nil || s.settingService == nil {
		return nil, nil
	}
	profile, err := s.settingService.GetActiveGoogleBatchGCSProfile(ctx)
	if err != nil || profile == nil {
		return nil, err
	}
	if !profile.Enabled || strings.TrimSpace(profile.Bucket) == "" || strings.TrimSpace(profile.ServiceAccountJSON) == "" {
		return nil, nil
	}
	return profile, nil
}

func (s *GeminiMessagesCompatService) getGoogleBatchGCSProfileByID(ctx context.Context, profileID string) (*GoogleBatchGCSProfile, error) {
	if strings.TrimSpace(profileID) == "" {
		return s.getActiveGoogleBatchGCSProfile(ctx)
	}
	if s == nil || s.settingService == nil {
		return nil, nil
	}
	profiles, err := s.settingService.ListGoogleBatchGCSProfiles(ctx)
	if err != nil || profiles == nil {
		return nil, err
	}
	for idx := range profiles.Items {
		profile := profiles.Items[idx]
		if profile.ProfileID == strings.TrimSpace(profileID) && profile.Enabled {
			return &profile, nil
		}
	}
	return nil, nil
}

func isGoogleBatchQuotaFallbackResponse(result *UpstreamHTTPResult) bool {
	if result == nil {
		return false
	}
	switch result.StatusCode {
	case http.StatusTooManyRequests:
		return true
	case http.StatusForbidden:
		body := strings.ToLower(strings.TrimSpace(string(result.Body)))
		return strings.Contains(body, "quota") ||
			strings.Contains(body, "rate limit") ||
			strings.Contains(body, "resource_exhausted") ||
			strings.Contains(body, "resource exhausted") ||
			strings.Contains(body, "exceeded") ||
			strings.Contains(body, "enqueued") ||
			strings.Contains(body, "token")
	default:
		return false
	}
}

func buildVertexBatchPredictionJobsPath(account *Account) string {
	if account == nil {
		return ""
	}
	projectID := strings.TrimSpace(account.GetGeminiVertexProjectID())
	location := strings.TrimSpace(account.GetGeminiVertexLocation())
	if projectID == "" || location == "" {
		return ""
	}
	return "/v1/projects/" + url.PathEscape(projectID) + "/locations/" + url.PathEscape(location) + "/batchPredictionJobs"
}

func normalizePublicBatchName(seed string) string {
	value := strings.TrimSpace(strings.TrimPrefix(seed, "batches/"))
	if value == "" {
		value = generateRequestID()
	}
	return "batches/" + value
}

func publicResultFileNameForBatch(publicBatchName string) string {
	batchID := strings.TrimSpace(strings.TrimPrefix(publicBatchName, "batches/"))
	if batchID == "" {
		batchID = generateRequestID()
	}
	return "files/" + batchID + "-results"
}

func googleBatchGCSObjectPath(profile *GoogleBatchGCSProfile, publicBatchName string, name string) string {
	prefix := strings.Trim(strings.TrimSpace(profile.Prefix), "/")
	batchID := strings.TrimSpace(strings.TrimPrefix(publicBatchName, "batches/"))
	datePath := time.Now().UTC().Format("2006/01/02")
	parts := make([]string, 0, 4)
	if prefix != "" {
		parts = append(parts, prefix)
	}
	if datePath != "" {
		parts = append(parts, datePath)
	}
	if batchID != "" {
		parts = append(parts, batchID)
	}
	if strings.TrimSpace(name) != "" {
		parts = append(parts, strings.Trim(strings.TrimSpace(name), "/"))
	}
	return strings.Join(parts, "/")
}

func googleBatchGCSURI(profile *GoogleBatchGCSProfile, objectPath string) string {
	if profile == nil {
		return ""
	}
	return "gs://" + strings.TrimSpace(profile.Bucket) + "/" + strings.TrimLeft(strings.TrimSpace(objectPath), "/")
}

func (s *GeminiMessagesCompatService) buildVertexBatchOverflowRequest(ctx context.Context, input GoogleBatchForwardInput, sourceAccount *Account, targetAccount *Account, profile *GoogleBatchGCSProfile, publicBatchName string) ([]byte, map[string]any, error) {
	resolvedMetadata, err := s.resolveGoogleBatchInputMetadata(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	inputObject := googleBatchGCSObjectPath(profile, publicBatchName, "input.jsonl")
	outputPrefixObject := googleBatchGCSObjectPath(profile, publicBatchName, "output")
	sourceNames, err := s.stageVertexBatchOverflowInput(ctx, input, sourceAccount, profile, inputObject)
	if err != nil {
		return nil, nil, err
	}
	requestedModel := strings.TrimSpace(resolvedMetadata.requestedModel)
	displayName := strings.TrimSpace(gjson.GetBytes(input.Body, "batch.display_name").String())
	if displayName == "" {
		displayName = strings.TrimSpace(strings.TrimPrefix(publicBatchName, "batches/"))
	}
	vertexPayload := map[string]any{
		"displayName": displayName,
		"model":       "publishers/google/models/" + strings.TrimPrefix(requestedModel, "models/"),
		"inputConfig": map[string]any{
			"instancesFormat": "jsonl",
			"gcsSource": map[string]any{
				"uris": []string{googleBatchGCSURI(profile, inputObject)},
			},
		},
		"outputConfig": map[string]any{
			"predictionsFormat": "jsonl",
			"gcsDestination": map[string]any{
				"outputUriPrefix": googleBatchGCSURI(profile, outputPrefixObject),
			},
		},
	}
	body, err := json.Marshal(vertexPayload)
	if err != nil {
		return nil, nil, err
	}
	return body, map[string]any{
		googleBatchBindingMetadataRequestedModel:      requestedModel,
		googleBatchBindingMetadataModelFamily:         resolvedMetadata.modelFamily,
		googleBatchBindingMetadataEstimatedTokens:     resolvedMetadata.estimatedTokens,
		googleBatchBindingMetadataSourceProtocol:      resolvedMetadata.sourceProtocol,
		googleBatchBindingMetadataSourceResourceNames: sourceNames,
		"staging_profile_id":                          strings.TrimSpace(profile.ProfileID),
		"vertex_input_object":                         inputObject,
		"vertex_output_prefix_object":                 outputPrefixObject,
	}, nil
}

func (s *GeminiMessagesCompatService) stageVertexBatchOverflowInput(ctx context.Context, input GoogleBatchForwardInput, sourceAccount *Account, profile *GoogleBatchGCSProfile, inputObject string) ([]string, error) {
	requestsRaw := strings.TrimSpace(gjson.GetBytes(input.Body, "batch.input_config.requests.requests").Raw)
	if requestsRaw != "" && requestsRaw != "null" {
		var items []any
		if err := json.Unmarshal([]byte(requestsRaw), &items); err != nil {
			return nil, infraerrors.BadRequest("GOOGLE_BATCH_INLINE_REQUESTS_INVALID", "invalid AI Studio batch requests")
		}
		lines, err := marshalVertexBatchJSONLLines(convertInlineBatchItems(items))
		if err != nil {
			return nil, err
		}
		if err := s.uploadGoogleBatchGCSObject(ctx, profile, inputObject, "application/x-ndjson", lines); err != nil {
			return nil, err
		}
		return uniqueStrings(collectStringFieldsByKey(input.Body, "fileName")), nil
	}
	fileName := strings.TrimSpace(gjson.GetBytes(input.Body, "batch.input_config.file_name").String())
	if fileName == "" {
		fileName = strings.TrimSpace(gjson.GetBytes(input.Body, "batch.input_config.fileName").String())
	}
	if fileName == "" {
		return nil, infraerrors.BadRequest("GOOGLE_BATCH_INPUT_CONFIG_UNSUPPORTED", "unsupported AI Studio batch input configuration")
	}
	source, err := s.downloadAIStudioBatchSourceFileStream(ctx, input, sourceAccount, fileName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = source.Close() }()
	pipeReader, pipeWriter := io.Pipe()
	transformErrCh := make(chan error, 1)
	go func() {
		defer close(transformErrCh)
		err := normalizeAIStudioJSONLToVertexStream(source, pipeWriter)
		_ = pipeWriter.CloseWithError(err)
		transformErrCh <- err
	}()
	uploadErr := s.uploadGoogleBatchGCSObjectStream(ctx, profile, inputObject, "application/x-ndjson", pipeReader, -1)
	if uploadErr != nil {
		_ = pipeReader.CloseWithError(uploadErr)
	}
	transformErr := <-transformErrCh
	if uploadErr != nil {
		return nil, uploadErr
	}
	if transformErr != nil {
		return nil, transformErr
	}
	return []string{fileName}, nil
}

func convertInlineBatchItems(items []any) []map[string]any {
	lines := make([]map[string]any, 0, len(items))
	for idx, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		line := map[string]any{}
		if request, ok := item["request"].(map[string]any); ok {
			line["request"] = request
		} else {
			line["request"] = item
		}
		if metadata, ok := item["metadata"].(map[string]any); ok {
			if key := strings.TrimSpace(stringMapValue(metadata, "key")); key != "" {
				line["key"] = key
			}
		}
		if key := strings.TrimSpace(stringMapValue(item, "key")); key != "" {
			line["key"] = key
		}
		if _, ok := line["key"]; !ok {
			line["key"] = fmt.Sprintf("request-%d", idx+1)
		}
		lines = append(lines, line)
	}
	return lines
}

func normalizeAIStudioJSONLToVertex(payload []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := normalizeAIStudioJSONLToVertexStream(bytes.NewReader(payload), &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func normalizeAIStudioJSONLToVertexStream(reader io.Reader, writer io.Writer) error {
	return walkJSONLLines(reader, func(idx int, line []byte) error {
		var item map[string]any
		if err := json.Unmarshal(line, &item); err != nil {
			return infraerrors.BadRequest("GOOGLE_BATCH_SOURCE_FILE_INVALID", "batch source file must be JSONL")
		}
		wrapped := map[string]any{}
		if request, ok := item["request"].(map[string]any); ok {
			wrapped["request"] = request
		} else {
			wrapped["request"] = item
		}
		if key := strings.TrimSpace(stringMapValue(item, "key")); key != "" {
			wrapped["key"] = key
		} else if metadata, ok := item["metadata"].(map[string]any); ok {
			if key := strings.TrimSpace(stringMapValue(metadata, "key")); key != "" {
				wrapped["key"] = key
			}
		}
		if _, ok := wrapped["key"]; !ok {
			wrapped["key"] = fmt.Sprintf("request-%d", idx+1)
		}
		encoded, err := json.Marshal(wrapped)
		if err != nil {
			return err
		}
		if _, err := writer.Write(encoded); err != nil {
			return err
		}
		_, err = writer.Write([]byte{'\n'})
		return err
	})
}

func marshalVertexBatchJSONLLines(lines []map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	for _, line := range lines {
		encoded, err := json.Marshal(line)
		if err != nil {
			return nil, err
		}
		if _, err := buf.Write(encoded); err != nil {
			return nil, err
		}
		if err := buf.WriteByte('\n'); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func walkJSONLLines(reader io.Reader, fn func(idx int, line []byte) error) error {
	buffered := bufio.NewReader(reader)
	lineIndex := 0
	for {
		line, err := buffered.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			if callErr := fn(lineIndex, trimmed); callErr != nil {
				return callErr
			}
		}
		lineIndex++
		if err == io.EOF {
			return nil
		}
	}
}

func (s *GeminiMessagesCompatService) downloadAIStudioBatchSourceFileStream(ctx context.Context, input GoogleBatchForwardInput, sourceAccount *Account, fileName string) (io.ReadCloser, error) {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil, infraerrors.BadRequest("GOOGLE_BATCH_SOURCE_FILE_REQUIRED", "batch source file is required")
	}
	account := sourceAccount
	if binding, err := s.resourceBindingRepo.Get(ctx, UpstreamResourceKindGeminiFile, fileName); err == nil && binding != nil {
		if boundAccount, getErr := s.accountRepo.GetByID(ctx, binding.AccountID); getErr == nil && boundAccount != nil {
			account = boundAccount
		}
	}
	if account == nil {
		return nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_SOURCE_ACCOUNT_UNAVAILABLE", "source batch file account unavailable")
	}
	downloadInput := input
	downloadInput.Method = http.MethodGet
	downloadInput.Path = googleBatchArchivePublicFileDownloadPath(fileName)
	downloadInput.RawQuery = "alt=media"
	downloadInput.Body = nil
	downloadInput.OpenBody = nil
	downloadInput.ContentLength = 0
	result, err := s.forwardGoogleBatchToAccountStream(ctx, downloadInput, account, googleBatchTargetAIStudio)
	if err != nil {
		return nil, err
	}
	if result == nil || result.StatusCode < 200 || result.StatusCode >= 300 {
		if result != nil && result.Body != nil {
			_ = result.Body.Close()
		}
		return nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_SOURCE_FILE_FETCH_FAILED", "failed to fetch batch source file")
	}
	return result.Body, nil
}

func (s *GeminiMessagesCompatService) forwardGoogleBatchCreateViaVertexOverflow(ctx context.Context, input GoogleBatchForwardInput, selection *googleBatchOverflowSelection) (*UpstreamHTTPResult, *Account, error) {
	if selection == nil || selection.sourceAccount == nil || selection.targetAccount == nil || selection.gcsProfile == nil {
		return nil, nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_OVERFLOW_UNAVAILABLE", "AI Studio to Vertex overflow unavailable")
	}
	recordGoogleBatchOverflowHit()
	publicBatchName := normalizePublicBatchName(generateRequestID())
	publicResultFileName := publicResultFileNameForBatch(publicBatchName)
	requestBody, overflowMetadata, err := s.buildVertexBatchOverflowRequest(ctx, input, selection.sourceAccount, selection.targetAccount, selection.gcsProfile, publicBatchName)
	if err != nil {
		return nil, nil, err
	}
	vertexPath := buildVertexBatchPredictionJobsPath(selection.targetAccount)
	if vertexPath == "" {
		return nil, nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_VERTEX_PATH_UNAVAILABLE", "vertex batch path unavailable")
	}
	vertexInput := input
	vertexInput.Method = http.MethodPost
	vertexInput.Path = vertexPath
	vertexInput.RawQuery = ""
	vertexInput.Body = requestBody
	result, err := s.forwardGoogleBatchToAccount(ctx, vertexInput, selection.targetAccount, googleBatchTargetVertex)
	if err != nil || result == nil {
		return result, selection.targetAccount, err
	}
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return result, selection.targetAccount, nil
	}
	executionNames := extractCreatedResourceNames(UpstreamResourceKindVertexBatchJob, result.Body)
	if len(executionNames) == 0 {
		return nil, nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_VERTEX_CREATE_INVALID", "vertex batch create response missing name")
	}
	requestedModel := strings.TrimSpace(fmt.Sprintf("%v", overflowMetadata[googleBatchBindingMetadataRequestedModel]))
	now := time.Now().UTC()
	settings := s.getGoogleBatchArchiveSettings(ctx)
	nextPollAt := now.Add(time.Duration(settings.PollMinIntervalSeconds) * time.Second)
	retentionAt := now.AddDate(0, 0, selection.targetAccount.GetBatchArchiveRetentionDays())
	job := &GoogleBatchArchiveJob{
		PublicBatchName:         publicBatchName,
		PublicProtocol:          GoogleBatchArchivePublicProtocolAIStudio,
		ExecutionProviderFamily: UpstreamProviderVertexAI,
		ExecutionBatchName:      executionNames[0],
		SourceAccountID:         selection.sourceAccount.ID,
		ExecutionAccountID:      selection.targetAccount.ID,
		APIKeyID:                int64Ptr(input.APIKeyID),
		GroupID:                 input.GroupID,
		UserID:                  int64Ptr(input.UserID),
		RequestedModel:          requestedModel,
		ConversionDirection:     GoogleBatchArchiveConversionAIStudioToVertex,
		State:                   normalizeVertexBatchState(strings.TrimSpace(gjson.GetBytes(result.Body, "state").String())),
		NextPollAt:              &nextPollAt,
		ArchiveState:            GoogleBatchArchiveLifecyclePending,
		BillingSettlementState:  GoogleBatchArchiveBillingPending,
		RetentionExpiresAt:      &retentionAt,
		MetadataJSON: map[string]any{
			googleBatchBindingMetadataPublicProtocol:       UpstreamProviderAIStudio,
			googleBatchBindingMetadataExecutionProtocol:    UpstreamProviderVertexAI,
			googleBatchBindingMetadataVirtualResource:      true,
			googleBatchBindingMetadataConversionDirection:  GoogleBatchArchiveConversionAIStudioToVertex,
			googleBatchBindingMetadataPublicResultFileName: publicResultFileName,
			googleBatchBindingMetadataOfficialResultName:   "",
			googleBatchBindingMetadataRequestedModel:       requestedModel,
			googleBatchBindingMetadataModelFamily:          overflowMetadata[googleBatchBindingMetadataModelFamily],
			googleBatchBindingMetadataEstimatedTokens:      overflowMetadata[googleBatchBindingMetadataEstimatedTokens],
			googleBatchBindingMetadataSourceProtocol:       overflowMetadata[googleBatchBindingMetadataSourceProtocol],
			googleBatchBindingMetadataSourceResourceNames:  overflowMetadata[googleBatchBindingMetadataSourceResourceNames],
			"staging_profile_id":                           overflowMetadata["staging_profile_id"],
			"vertex_input_object":                          overflowMetadata["vertex_input_object"],
			"vertex_output_prefix_object":                  overflowMetadata["vertex_output_prefix_object"],
			"billing_type":                                 int(input.BillingType),
			"subscription_id":                              derefInt64(input.SubscriptionID),
		},
	}
	if job.State == "" {
		job.State = GoogleBatchArchiveStateCreated
	}
	if err := s.upsertGoogleBatchArchiveJob(ctx, job); err != nil {
		return nil, nil, err
	}
	storedJob, err := s.getGoogleBatchArchiveJobByPublicBatchName(ctx, publicBatchName)
	if err == nil && storedJob != nil {
		job = storedJob
	}
	virtualBatch := translateVertexBatchPayloadToAIStudio(job, result.Body)
	if err := s.storeGoogleBatchSnapshot(ctx, settings, job, virtualBatch); err != nil {
		return nil, nil, err
	}
	resultObject := &GoogleBatchArchiveObject{
		JobID:                 job.ID,
		PublicResourceKind:    GoogleBatchArchiveResourceKindFile,
		PublicResourceName:    publicResultFileName,
		ExecutionResourceName: "",
		IsResultPayload:       true,
		MetadataJSON: map[string]any{
			"public_batch_name":           publicBatchName,
			"staging_profile_id":          overflowMetadata["staging_profile_id"],
			"vertex_output_prefix_object": overflowMetadata["vertex_output_prefix_object"],
		},
	}
	if err := s.upsertGoogleBatchArchiveObject(ctx, resultObject); err != nil {
		return nil, nil, err
	}
	batchBinding := &UpstreamResourceBinding{
		ResourceKind:   UpstreamResourceKindGeminiBatch,
		ResourceName:   publicBatchName,
		ProviderFamily: UpstreamProviderVertexAI,
		AccountID:      selection.targetAccount.ID,
		APIKeyID:       int64Ptr(input.APIKeyID),
		GroupID:        input.GroupID,
		UserID:         int64Ptr(input.UserID),
		MetadataJSON: buildGoogleBatchBindingMetadata(map[string]any{
			googleBatchBindingMetadataArchiveJobID:         job.ID,
			googleBatchBindingMetadataPublicProtocol:       UpstreamProviderAIStudio,
			googleBatchBindingMetadataExecutionProtocol:    UpstreamProviderVertexAI,
			googleBatchBindingMetadataVirtualResource:      true,
			googleBatchBindingMetadataPublicResultFileName: publicResultFileName,
			googleBatchBindingMetadataOfficialResultName:   "",
			googleBatchBindingMetadataConversionDirection:  GoogleBatchArchiveConversionAIStudioToVertex,
			googleBatchBindingMetadataRequestedModel:       requestedModel,
			googleBatchBindingMetadataModelFamily:          overflowMetadata[googleBatchBindingMetadataModelFamily],
			googleBatchBindingMetadataEstimatedTokens:      overflowMetadata[googleBatchBindingMetadataEstimatedTokens],
			googleBatchBindingMetadataSourceProtocol:       overflowMetadata[googleBatchBindingMetadataSourceProtocol],
			googleBatchBindingMetadataSourceResourceNames:  overflowMetadata[googleBatchBindingMetadataSourceResourceNames],
			"staging_profile_id":                           overflowMetadata["staging_profile_id"],
		}),
	}
	fileBinding := &UpstreamResourceBinding{
		ResourceKind:   UpstreamResourceKindGeminiFile,
		ResourceName:   publicResultFileName,
		ProviderFamily: UpstreamProviderVertexAI,
		AccountID:      selection.targetAccount.ID,
		APIKeyID:       int64Ptr(input.APIKeyID),
		GroupID:        input.GroupID,
		UserID:         int64Ptr(input.UserID),
		MetadataJSON:   buildGoogleBatchArchiveFileBindingMetadata(job, resultObject),
	}
	if s.resourceBindingRepo != nil {
		if err := s.resourceBindingRepo.Upsert(ctx, batchBinding); err != nil {
			return nil, nil, err
		}
		if err := s.resourceBindingRepo.Upsert(ctx, fileBinding); err != nil {
			return nil, nil, err
		}
	}
	if err := s.persistGoogleBatchArchiveManifest(ctx, settings, job); err != nil {
		return nil, nil, err
	}
	if err := s.reserveGoogleBatchQuota(ctx, vertexInput, selection.targetAccount, googleBatchTargetVertex, executionNames[0]); err != nil {
		return nil, nil, err
	}
	_ = s.recordGoogleBatchUsageEvent(ctx, input, selection.targetAccount, requestedModel, UsageOperationBatchCreate, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-create:"+generateRequestID())
	return s.buildGoogleBatchJSONResult(http.StatusOK, virtualBatch), selection.targetAccount, nil
}

func buildGoogleBatchBindingMetadata(extra map[string]any) map[string]any {
	metadata := map[string]any{}
	for key, value := range extra {
		metadata[key] = value
	}
	return normalizeGoogleBatchBindingMetadata(metadata)
}

func (s *GeminiMessagesCompatService) googleBatchGCSAccessToken(ctx context.Context, profile *GoogleBatchGCSProfile) (string, error) {
	if profile == nil {
		return "", fmt.Errorf("google batch gcs profile is nil")
	}
	creds, err := parseVertexServiceAccountCredentials(strings.TrimSpace(profile.ServiceAccountJSON))
	if err != nil {
		return "", err
	}
	assertion, err := buildVertexServiceAccountAssertion(creds, time.Now())
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("grant_type", vertexServiceAccountTokenPath)
	form.Set("assertion", assertion)
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:               20 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ValidateResolvedIP:    true,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, creds.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("google batch gcs token exchange failed with status %d", resp.StatusCode)
	}
	var token vertexServiceAccountTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", err
	}
	return strings.TrimSpace(token.AccessToken), nil
}

func (s *GeminiMessagesCompatService) uploadGoogleBatchGCSObject(ctx context.Context, profile *GoogleBatchGCSProfile, objectPath string, contentType string, payload []byte) error {
	return s.uploadGoogleBatchGCSObjectStream(ctx, profile, objectPath, contentType, bytes.NewReader(payload), int64(len(payload)))
}

func (s *GeminiMessagesCompatService) uploadGoogleBatchGCSObjectStream(ctx context.Context, profile *GoogleBatchGCSProfile, objectPath string, contentType string, body io.Reader, contentLength int64) error {
	if profile == nil {
		return fmt.Errorf("google batch gcs profile is nil")
	}
	token, err := s.googleBatchGCSAccessToken(ctx, profile)
	if err != nil {
		return err
	}
	uploadURL := "https://storage.googleapis.com/upload/storage/v1/b/" + url.PathEscape(strings.TrimSpace(profile.Bucket)) + "/o?uploadType=media&name=" + url.QueryEscape(strings.TrimLeft(strings.TrimSpace(objectPath), "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, body)
	if err != nil {
		return err
	}
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:               30 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ValidateResolvedIP:    true,
	})
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("upload google batch gcs object failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *GeminiMessagesCompatService) listGoogleBatchGCSObjects(ctx context.Context, profile *GoogleBatchGCSProfile, prefix string) ([]googleBatchGCSObject, error) {
	token, err := s.googleBatchGCSAccessToken(ctx, profile)
	if err != nil {
		return nil, err
	}
	listURL := "https://storage.googleapis.com/storage/v1/b/" + url.PathEscape(strings.TrimSpace(profile.Bucket)) + "/o?prefix=" + url.QueryEscape(strings.TrimLeft(strings.TrimSpace(prefix), "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:               30 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ValidateResolvedIP:    true,
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list google batch gcs objects failed with status %d", resp.StatusCode)
	}
	var payload googleBatchGCSListResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

func (s *GeminiMessagesCompatService) downloadGoogleBatchGCSObjectStream(ctx context.Context, profile *GoogleBatchGCSProfile, objectPath string) (*UpstreamHTTPStreamResult, error) {
	token, err := s.googleBatchGCSAccessToken(ctx, profile)
	if err != nil {
		return nil, err
	}
	downloadURL := "https://storage.googleapis.com/storage/v1/b/" + url.PathEscape(strings.TrimSpace(profile.Bucket)) + "/o/" + url.PathEscape(strings.TrimLeft(strings.TrimSpace(objectPath), "/")) + "?alt=media"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:               30 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ValidateResolvedIP:    true,
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download google batch gcs object failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return &UpstreamHTTPStreamResult{
		StatusCode:    resp.StatusCode,
		Headers:       resp.Header.Clone(),
		Body:          resp.Body,
		ContentLength: resp.ContentLength,
	}, nil
}

func selectGoogleBatchResultObject(items []googleBatchGCSObject, inputObjectPath string) *googleBatchGCSObject {
	for idx := range items {
		item := items[idx]
		if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Name) == strings.TrimSpace(inputObjectPath) {
			continue
		}
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(item.Name)), ".jsonl") {
			return &item
		}
	}
	for idx := range items {
		item := items[idx]
		if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Name) == strings.TrimSpace(inputObjectPath) {
			continue
		}
		return &item
	}
	return nil
}

func (s *GeminiMessagesCompatService) fetchVertexBatchArchiveResultStream(ctx context.Context, job *GoogleBatchArchiveJob) (io.ReadCloser, string, int64, string, error) {
	if job == nil {
		return nil, "", 0, "", fmt.Errorf("archive job is nil")
	}
	profileID, _ := metadataString(job.MetadataJSON, "staging_profile_id")
	outputPrefixObject, _ := metadataString(job.MetadataJSON, "vertex_output_prefix_object")
	inputObjectPath, _ := metadataString(job.MetadataJSON, "vertex_input_object")
	profile, err := s.getGoogleBatchGCSProfileByID(ctx, profileID)
	if err != nil || profile == nil {
		return nil, "", 0, "", fmt.Errorf("google batch gcs profile unavailable")
	}
	items, err := s.listGoogleBatchGCSObjects(ctx, profile, outputPrefixObject)
	if err != nil {
		return nil, "", 0, "", err
	}
	resultObject := selectGoogleBatchResultObject(items, inputObjectPath)
	if resultObject == nil {
		return nil, "", 0, "", infraerrors.NotFound("GOOGLE_BATCH_VERTEX_RESULT_NOT_FOUND", "vertex batch result not found")
	}
	stream, err := s.downloadGoogleBatchGCSObjectStream(ctx, profile, resultObject.Name)
	if err != nil {
		return nil, "", 0, "", err
	}
	contentType := headerValue(stream.Headers, "Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = strings.TrimSpace(resultObject.ContentType)
	}
	return stream.Body, contentType, stream.ContentLength, resultObject.Name, nil
}
