package service

import (
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	googleBatchArchiveRuntimePollWorkerName     = "google_batch_archive_poller"
	googleBatchArchiveRuntimePrefetchWorkerName = "google_batch_archive_prefetch"
	googleBatchArchiveRuntimeCleanupWorkerName  = "google_batch_archive_cleanup"
)

func googleBatchArchivePollInterval(settings *GoogleBatchArchiveSettings) time.Duration {
	normalized := NormalizeGoogleBatchArchiveSettings(settings)
	return time.Duration(normalized.PollMinIntervalSeconds) * time.Second
}

func googleBatchArchiveCleanupInterval(settings *GoogleBatchArchiveSettings) time.Duration {
	normalized := NormalizeGoogleBatchArchiveSettings(settings)
	return time.Duration(normalized.CleanupIntervalMinutes) * time.Minute
}

func googleBatchArchiveNextRetryAt(settings *GoogleBatchArchiveSettings, attempts int) time.Time {
	normalized := NormalizeGoogleBatchArchiveSettings(settings)
	if attempts < 1 {
		attempts = 1
	}
	baseSeconds := float64(normalized.PollMinIntervalSeconds)
	maxSeconds := float64(normalized.PollMaxIntervalSeconds)
	factor := float64(normalized.PollBackoffFactor)
	if factor <= 1 {
		factor = 2
	}
	backoff := baseSeconds * math.Pow(factor, float64(attempts-1))
	if backoff > maxSeconds {
		backoff = maxSeconds
	}
	if normalized.PollJitterSeconds > 0 {
		backoff += float64(rand.Intn(normalized.PollJitterSeconds + 1))
	}
	return time.Now().UTC().Add(time.Duration(backoff) * time.Second)
}

func googleBatchArchivePublicBatchPath(publicBatchName string) string {
	name := strings.Trim(strings.TrimSpace(publicBatchName), "/")
	if name == "" {
		return ""
	}
	return "/v1beta/" + name
}

func googleBatchArchivePublicFileMetadataPath(publicFileName string) string {
	name := strings.Trim(strings.TrimSpace(publicFileName), "/")
	if name == "" {
		return ""
	}
	return "/v1beta/" + name
}

func googleBatchArchivePublicFileDownloadPath(publicFileName string) string {
	name := strings.Trim(strings.TrimSpace(publicFileName), "/")
	if name == "" {
		return ""
	}
	return "/download/v1beta/" + name + ":download"
}

func googleBatchArchiveVertexBatchPath(executionBatchName string) string {
	name := strings.Trim(strings.TrimSpace(executionBatchName), "/")
	if name == "" {
		return ""
	}
	return "/v1/" + name
}

func googleBatchArchiveInputFromJob(job *GoogleBatchArchiveJob, method string, path string, rawQuery string) GoogleBatchForwardInput {
	var (
		apiKeyID       int64
		userID         int64
		billingType    int8
		subscriptionID *int64
		groupID        *int64
	)
	if job != nil {
		if job.APIKeyID != nil {
			apiKeyID = *job.APIKeyID
		}
		if job.UserID != nil {
			userID = *job.UserID
		}
		if job.GroupID != nil {
			groupID = job.GroupID
		}
		if value, ok := metadataInt64(job.MetadataJSON, "subscription_id"); ok && value > 0 {
			subscriptionID = &value
		}
		if value, ok := metadataInt64(job.MetadataJSON, "billing_type"); ok {
			billingType = int8(value)
		}
	}
	return GoogleBatchForwardInput{
		GroupID:        groupID,
		APIKeyID:       apiKeyID,
		UserID:         userID,
		BillingType:    billingType,
		SubscriptionID: subscriptionID,
		Method:         strings.ToUpper(strings.TrimSpace(method)),
		Path:           path,
		RawQuery:       rawQuery,
		Headers:        make(http.Header),
	}
}

func googleBatchArchiveTargetForJob(job *GoogleBatchArchiveJob) googleBatchTarget {
	if job != nil && strings.EqualFold(strings.TrimSpace(job.ExecutionProviderFamily), UpstreamProviderVertexAI) {
		return googleBatchTargetVertex
	}
	return googleBatchTargetAIStudio
}

func translateVertexBatchPayloadToAIStudio(job *GoogleBatchArchiveJob, payload []byte) []byte {
	if len(payload) == 0 {
		return buildArchivedAIStudioBatchPayload(job, nil)
	}
	var vertexPayload map[string]any
	if err := json.Unmarshal(payload, &vertexPayload); err != nil {
		return buildArchivedAIStudioBatchPayload(job, nil)
	}
	state := normalizeVertexBatchState(stringMapValue(vertexPayload, "state"))
	resultFileName, _ := metadataString(job.MetadataJSON, "public_result_file_name")
	if resultFileName == "" {
		if fileName := googleBatchArchiveDeriveResultFileName(job); fileName != "" {
			resultFileName = fileName
		}
	}
	response := map[string]any{
		"name":  strings.TrimSpace(job.PublicBatchName),
		"state": state,
	}
	if strings.TrimSpace(job.RequestedModel) != "" {
		response["model"] = "models/" + strings.TrimPrefix(strings.TrimSpace(job.RequestedModel), "models/")
	}
	if created := strings.TrimSpace(stringMapValue(vertexPayload, "createTime")); created != "" {
		response["createTime"] = created
	}
	if updated := strings.TrimSpace(stringMapValue(vertexPayload, "updateTime")); updated != "" {
		response["updateTime"] = updated
	}
	if resultFileName != "" {
		response["dest"] = map[string]any{"fileName": resultFileName}
	}
	body, err := json.Marshal(response)
	if err != nil {
		return buildArchivedAIStudioBatchPayload(job, nil)
	}
	return body
}

func normalizeVertexBatchState(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "JOB_STATE_SUCCEEDED", "SUCCEEDED":
		return GoogleBatchArchiveStateSucceeded
	case "JOB_STATE_FAILED", "FAILED":
		return GoogleBatchArchiveStateFailed
	case "JOB_STATE_CANCELLED", "CANCELLED":
		return GoogleBatchArchiveStateCancelled
	case "JOB_STATE_PENDING", "PENDING":
		return GoogleBatchArchiveStateCreated
	case "JOB_STATE_QUEUED", "QUEUED", "JOB_STATE_RUNNING", "RUNNING":
		return GoogleBatchArchiveStateRunning
	default:
		return GoogleBatchArchiveStateUnknown
	}
}

func googleBatchArchiveDeriveResultFileName(job *GoogleBatchArchiveJob) string {
	if job == nil {
		return ""
	}
	base := strings.TrimPrefix(strings.TrimSpace(job.PublicBatchName), "batches/")
	if base == "" {
		return ""
	}
	return "files/" + base + "-results"
}
