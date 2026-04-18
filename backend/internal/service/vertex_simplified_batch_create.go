package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/tidwall/gjson"
)

type simplifiedVertexBatchCreateSpec struct {
	body           []byte
	managedOutput  *simplifiedVertexManagedOutput
	requestedModel string
}

type simplifiedVertexManagedOutput struct {
	profileID          string
	inputObject        string
	outputPrefixObject string
}

func (s *GeminiMessagesCompatService) buildSimplifiedVertexBatchCreateSpec(ctx context.Context, input GoogleBatchForwardInput) (*simplifiedVertexBatchCreateSpec, error) {
	var payload map[string]any
	if err := json.Unmarshal(input.Body, &payload); err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_INVALID", "invalid JSON body")
	}

	if isNativeVertexBatchCreatePayload(payload) {
		return s.buildNativeSimplifiedVertexBatchCreateSpec(ctx, payload)
	}
	return s.buildFriendlySimplifiedVertexBatchCreateSpec(ctx, payload)
}

func (s *GeminiMessagesCompatService) buildNativeSimplifiedVertexBatchCreateSpec(ctx context.Context, payload map[string]any) (*simplifiedVertexBatchCreateSpec, error) {
	requestedModel := canonicalVertexBatchModelName(stringValueFromAny(payload["model"]))
	if requestedModel == "" {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_MODEL_REQUIRED", "model is required")
	}
	payload["model"] = requestedModel
	managedOutput, err := s.ensureSimplifiedVertexBatchOutput(ctx, payload, generateRequestID())
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_INVALID", "invalid Vertex batch body")
	}
	return &simplifiedVertexBatchCreateSpec{body: body, managedOutput: managedOutput, requestedModel: requestedModel}, nil
}

func (s *GeminiMessagesCompatService) buildFriendlySimplifiedVertexBatchCreateSpec(ctx context.Context, payload map[string]any) (*simplifiedVertexBatchCreateSpec, error) {
	requestedModel := canonicalVertexBatchModelName(stringValueFromAny(payload["model"]))
	if requestedModel == "" {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_MODEL_REQUIRED", "model is required")
	}

	requestID := generateRequestID()
	outputPrefix := strings.TrimSpace(stringValueFromAny(payload["output_uri_prefix"]))
	inputURI := strings.TrimSpace(stringValueFromAny(payload["input_uri"]))
	requests, hasRequests := payload["requests"].([]any)
	if inputURI == "" && !hasRequests {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BATCH_SOURCE_REQUIRED", "provide either requests or input_uri")
	}
	if inputURI != "" && hasRequests {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BATCH_SOURCE_CONFLICT", "requests and input_uri cannot be used together")
	}

	var managedOutput *simplifiedVertexManagedOutput
	if inputURI == "" || outputPrefix == "" {
		profile, err := s.getActiveGoogleBatchGCSProfile(ctx)
		if err != nil {
			return nil, err
		}
		if profile == nil {
			return nil, infraerrors.ServiceUnavailable("VERTEX_SIMPLIFIED_GCS_PROFILE_UNAVAILABLE", "no active Google Batch GCS profile is available")
		}
		managedOutput = &simplifiedVertexManagedOutput{
			profileID:          strings.TrimSpace(profile.ProfileID),
			inputObject:        googleBatchGCSObjectPath(profile, "vertex-"+requestID, "input.jsonl"),
			outputPrefixObject: googleBatchGCSObjectPath(profile, "vertex-"+requestID, "output"),
		}
		if inputURI == "" {
			lines, err := normalizeSimplifiedVertexBatchRequests(requestedModel, requests)
			if err != nil {
				return nil, err
			}
			if err := s.uploadGoogleBatchGCSObject(ctx, profile, managedOutput.inputObject, "application/x-ndjson", lines); err != nil {
				return nil, infraerrors.ServiceUnavailable("VERTEX_SIMPLIFIED_GCS_UPLOAD_FAILED", "failed to upload managed Vertex batch input").WithCause(err)
			}
			inputURI = googleBatchGCSURI(profile, managedOutput.inputObject)
		}
		if outputPrefix == "" {
			outputPrefix = googleBatchGCSURI(profile, managedOutput.outputPrefixObject)
		}
	}

	vertexPayload := map[string]any{
		"displayName": firstNonEmptyString(stringValueFromAny(payload["display_name"]), stringValueFromAny(payload["displayName"])),
		"model":       requestedModel,
		"inputConfig": map[string]any{
			"instancesFormat": "jsonl",
			"gcsSource": map[string]any{
				"uris": []string{inputURI},
			},
		},
		"outputConfig": map[string]any{
			"predictionsFormat": "jsonl",
			"gcsDestination": map[string]any{
				"outputUriPrefix": outputPrefix,
			},
		},
	}
	copyGeminiRequestFieldExact(vertexPayload, payload, "labels")
	copyGeminiRequestFieldExact(vertexPayload, payload, "metadata")
	body, err := json.Marshal(vertexPayload)
	if err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_INVALID", "invalid simplified Vertex batch body")
	}
	return &simplifiedVertexBatchCreateSpec{body: body, managedOutput: managedOutput, requestedModel: requestedModel}, nil
}

func normalizeSimplifiedVertexBatchRequests(requestedModel string, requests []any) ([]byte, error) {
	lines := make([]map[string]any, 0, len(requests))
	for index, rawItem := range requests {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_REQUESTS_INVALID", "requests must be an array of objects")
		}
		nativeRequest, err := normalizeSimplifiedVertexBatchRequestItem(requestedModel, item["request"])
		if err != nil {
			return nil, err
		}
		key := strings.TrimSpace(stringValueFromAny(item["key"]))
		if key == "" {
			key = "request-" + time.Now().UTC().Format("150405")
		}
		lines = append(lines, map[string]any{"key": firstNonEmptyString(key, "request-"+stringValueFromAny(index+1)), "request": nativeRequest})
	}
	return marshalVertexBatchJSONLLines(lines)
}

func normalizeSimplifiedVertexBatchRequestItem(requestedModel string, raw any) (map[string]any, error) {
	if raw == nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_REQUESTS_INVALID", "each request item must include request")
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_REQUESTS_INVALID", "invalid request item")
	}
	normalized, err := NormalizeSimplifiedVertexModelRequest(requestedModel, "generateContent", body)
	if err != nil {
		return nil, err
	}
	var request map[string]any
	if err := json.Unmarshal(normalized, &request); err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_REQUESTS_INVALID", "invalid normalized request item")
	}
	return request, nil
}

func (s *GeminiMessagesCompatService) ensureSimplifiedVertexBatchOutput(ctx context.Context, payload map[string]any, requestID string) (*simplifiedVertexManagedOutput, error) {
	outputConfig, _ := firstNonNil(payload["outputConfig"], payload["output_config"]).(map[string]any)
	if outputConfig == nil {
		outputConfig = map[string]any{"predictionsFormat": "jsonl", "gcsDestination": map[string]any{}}
		payload["outputConfig"] = outputConfig
	}
	gcsDestination, _ := firstNonNil(outputConfig["gcsDestination"], outputConfig["gcs_destination"]).(map[string]any)
	if gcsDestination == nil {
		gcsDestination = map[string]any{}
		outputConfig["gcsDestination"] = gcsDestination
	}
	if strings.TrimSpace(stringValueFromAny(firstNonNil(gcsDestination["outputUriPrefix"], gcsDestination["output_uri_prefix"]))) != "" {
		return nil, nil
	}
	profile, err := s.getActiveGoogleBatchGCSProfile(ctx)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, infraerrors.ServiceUnavailable("VERTEX_SIMPLIFIED_GCS_PROFILE_UNAVAILABLE", "no active Google Batch GCS profile is available")
	}
	managed := &simplifiedVertexManagedOutput{
		profileID:          strings.TrimSpace(profile.ProfileID),
		outputPrefixObject: googleBatchGCSObjectPath(profile, "vertex-"+requestID, "output"),
	}
	gcsDestination["outputUriPrefix"] = googleBatchGCSURI(profile, managed.outputPrefixObject)
	return managed, nil
}

func (s *GeminiMessagesCompatService) persistSimplifiedVertexBatchArchive(ctx context.Context, input GoogleBatchForwardInput, account *Account, requestedModel string, managed *simplifiedVertexManagedOutput, result *UpstreamHTTPResult) {
	if managed == nil || result == nil || result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return
	}
	if s.googleBatchArchiveJobRepo == nil || s.googleBatchArchiveObjectRepo == nil {
		return
	}
	executionName := strings.TrimSpace(gjson.GetBytes(result.Body, "name").String())
	jobName := extractVertexBatchJobName(executionName)
	if jobName == "" {
		return
	}
	settings := s.getGoogleBatchArchiveSettings(ctx)
	now := time.Now().UTC()
	publicBatchName := "batches/" + jobName
	publicResultFileName := publicResultFileNameForBatch(publicBatchName)
	nextPollAt := now.Add(time.Duration(settings.PollMinIntervalSeconds) * time.Second)
	retentionAt := now.AddDate(0, 0, account.GetBatchArchiveRetentionDays())
	job := &GoogleBatchArchiveJob{
		PublicBatchName:         publicBatchName,
		PublicProtocol:          GoogleBatchArchivePublicProtocolAIStudio,
		ExecutionProviderFamily: UpstreamProviderVertexAI,
		ExecutionBatchName:      executionName,
		SourceAccountID:         account.ID,
		ExecutionAccountID:      account.ID,
		APIKeyID:                int64Ptr(input.APIKeyID),
		GroupID:                 input.GroupID,
		UserID:                  int64Ptr(input.UserID),
		RequestedModel:          strings.TrimPrefix(requestedModel, "publishers/google/models/"),
		ConversionDirection:     GoogleBatchArchiveConversionNone,
		State:                   normalizeVertexBatchState(strings.TrimSpace(gjson.GetBytes(result.Body, "state").String())),
		NextPollAt:              &nextPollAt,
		ArchiveState:            GoogleBatchArchiveLifecyclePending,
		BillingSettlementState:  GoogleBatchArchiveBillingPending,
		RetentionExpiresAt:      &retentionAt,
		MetadataJSON: buildGoogleBatchBindingMetadata(map[string]any{
			googleBatchBindingMetadataPublicProtocol:       UpstreamProviderVertexAI,
			googleBatchBindingMetadataExecutionProtocol:    UpstreamProviderVertexAI,
			googleBatchBindingMetadataVirtualResource:      false,
			googleBatchBindingMetadataConversionDirection:  GoogleBatchArchiveConversionNone,
			googleBatchBindingMetadataPublicResultFileName: publicResultFileName,
			googleBatchBindingMetadataOfficialResultName:   "",
			googleBatchBindingMetadataRequestedModel:       strings.TrimPrefix(requestedModel, "publishers/google/models/"),
			googleBatchBindingMetadataModelFamily:          normalizeGoogleBatchModelFamily(strings.TrimPrefix(requestedModel, "publishers/google/models/")),
			googleBatchBindingMetadataSourceProtocol:       UpstreamProviderVertexAI,
			"staging_profile_id":                           managed.profileID,
			"vertex_input_object":                          managed.inputObject,
			"vertex_output_prefix_object":                  managed.outputPrefixObject,
			"billing_type":                                 int(input.BillingType),
			"subscription_id":                              derefInt64(input.SubscriptionID),
		}),
	}
	if job.State == "" {
		job.State = GoogleBatchArchiveStateCreated
	}
	if err := s.upsertGoogleBatchArchiveJob(ctx, job); err != nil {
		return
	}
	storedJob, err := s.getGoogleBatchArchiveJobByPublicBatchName(ctx, publicBatchName)
	if err == nil && storedJob != nil {
		job = storedJob
	}
	_ = s.upsertGoogleBatchArchiveObject(ctx, &GoogleBatchArchiveObject{
		JobID:              job.ID,
		PublicResourceKind: GoogleBatchArchiveResourceKindFile,
		PublicResourceName: publicResultFileName,
		IsResultPayload:    true,
		MetadataJSON: map[string]any{
			"public_batch_name":           publicBatchName,
			"staging_profile_id":          managed.profileID,
			"vertex_output_prefix_object": managed.outputPrefixObject,
		},
	})
	_ = s.storeGoogleBatchSnapshot(ctx, settings, job, translateVertexBatchPayloadToAIStudio(job, result.Body))
	_ = s.persistGoogleBatchArchiveManifest(ctx, settings, job)
}

func canonicalVertexBatchModelName(value string) string {
	modelID := vertexSimplifiedCanonicalModelID(value)
	if modelID == "" {
		return ""
	}
	return "publishers/google/models/" + modelID
}

func isNativeVertexBatchCreatePayload(payload map[string]any) bool {
	return payload != nil && (payload["inputConfig"] != nil || payload["input_config"] != nil || payload["outputConfig"] != nil || payload["output_config"] != nil)
}
