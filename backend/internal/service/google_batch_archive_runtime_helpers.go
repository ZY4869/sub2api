package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	googleBatchBindingMetadataArchiveJobID         = "archive_job_id"
	googleBatchBindingMetadataPublicProtocol       = "public_protocol"
	googleBatchBindingMetadataExecutionProtocol    = "execution_protocol"
	googleBatchBindingMetadataLegacyUpstream       = "upstream_protocol"
	googleBatchBindingMetadataVirtualResource      = "virtual_resource"
	googleBatchBindingMetadataPublicResultFileName = "public_result_file_name"
	googleBatchBindingMetadataOfficialResultName   = "official_result_file_name"
	googleBatchBindingMetadataConversionDirection  = "conversion_direction"
)

const (
	googleBatchArchiveSourceLocal             = "local"
	googleBatchArchiveSourceUpstreamFetchable = "upstream_fetchable"
	googleBatchArchiveSourceUnavailable       = "unavailable"
)

type googleBatchArchiveManifest struct {
	JobID                int64   `json:"job_id"`
	PublicBatchName      string  `json:"public_batch_name"`
	PublicProtocol       string  `json:"public_protocol"`
	ExecutionProtocol    string  `json:"execution_protocol"`
	ExecutionBatchName   string  `json:"execution_batch_name"`
	PublicResultFileName string  `json:"public_result_file_name"`
	ContentType          string  `json:"content_type"`
	SizeBytes            int64   `json:"size_bytes"`
	SHA256               string  `json:"sha256"`
	ArchivedAt           string  `json:"archived_at"`
	RetentionExpiresAt   *string `json:"retention_expires_at"`
}

type googleBatchArchiveResponse struct {
	State              string  `json:"state"`
	ResultFileName     string  `json:"result_file_name"`
	Downloadable       bool    `json:"downloadable"`
	Source             string  `json:"source"`
	DownloadPath       string  `json:"download_path"`
	RetentionExpiresAt *string `json:"retention_expires_at"`
}

func normalizeGoogleBatchBindingMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		metadata = map[string]any{}
	}
	if executionProtocol, ok := metadataString(metadata, googleBatchBindingMetadataExecutionProtocol); !ok || executionProtocol == "" {
		if legacyProtocol, ok := metadataString(metadata, googleBatchBindingMetadataLegacyUpstream); ok && legacyProtocol != "" {
			metadata[googleBatchBindingMetadataExecutionProtocol] = legacyProtocol
		}
	}
	return metadata
}

func bindingExecutionProtocol(binding *UpstreamResourceBinding) string {
	metadata := bindingMetadata(binding)
	if value, ok := metadataString(metadata, googleBatchBindingMetadataExecutionProtocol); ok {
		return value
	}
	if value, ok := metadataString(metadata, googleBatchBindingMetadataLegacyUpstream); ok {
		return value
	}
	if binding != nil {
		return strings.TrimSpace(binding.ProviderFamily)
	}
	return ""
}

func bindingVirtualResource(binding *UpstreamResourceBinding) bool {
	return metadataBool(bindingMetadata(binding), googleBatchBindingMetadataVirtualResource)
}

func bindingArchiveJobID(binding *UpstreamResourceBinding) int64 {
	value, _ := metadataInt64(bindingMetadata(binding), googleBatchBindingMetadataArchiveJobID)
	return value
}

func archiveRetentionRFC3339(job *GoogleBatchArchiveJob) *string {
	if job == nil || job.RetentionExpiresAt == nil || job.RetentionExpiresAt.IsZero() {
		return nil
	}
	value := job.RetentionExpiresAt.UTC().Format(time.RFC3339)
	return &value
}

func archiveDownloadPath(publicResultFileName string) string {
	name := strings.Trim(strings.TrimSpace(publicResultFileName), "/")
	if name == "" {
		return ""
	}
	return "/google/batch/archive/v1beta/" + name + ":download"
}

func archiveResultFileName(job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject) string {
	if object != nil && strings.TrimSpace(object.PublicResourceName) != "" {
		return strings.TrimSpace(object.PublicResourceName)
	}
	if value, ok := metadataString(metadataMapForArchiveJob(job), googleBatchBindingMetadataPublicResultFileName); ok {
		return value
	}
	if job != nil && strings.EqualFold(strings.TrimSpace(job.ExecutionProviderFamily), UpstreamProviderVertexAI) {
		return googleBatchArchiveDeriveResultFileName(job)
	}
	return ""
}

func metadataMapForArchiveJob(job *GoogleBatchArchiveJob) map[string]any {
	if job == nil || job.MetadataJSON == nil {
		return map[string]any{}
	}
	return job.MetadataJSON
}

func archiveExecutionProtocol(job *GoogleBatchArchiveJob) string {
	if job == nil {
		return ""
	}
	if value, ok := metadataString(metadataMapForArchiveJob(job), googleBatchBindingMetadataExecutionProtocol); ok {
		return value
	}
	return strings.TrimSpace(job.ExecutionProviderFamily)
}

func archivePublicProtocol(job *GoogleBatchArchiveJob) string {
	if job == nil {
		return ""
	}
	if value, ok := metadataString(metadataMapForArchiveJob(job), googleBatchBindingMetadataPublicProtocol); ok {
		return value
	}
	return strings.TrimSpace(job.PublicProtocol)
}

func archiveVirtualResource(job *GoogleBatchArchiveJob) bool {
	return metadataBool(metadataMapForArchiveJob(job), googleBatchBindingMetadataVirtualResource)
}

func (s *GeminiMessagesCompatService) persistGoogleBatchArchiveManifest(ctx context.Context, settings *GoogleBatchArchiveSettings, job *GoogleBatchArchiveJob) error {
	if s == nil || s.googleBatchArchiveStorage == nil || job == nil {
		return nil
	}
	resultObject, _ := s.findGoogleBatchArchiveResultObject(ctx, job)
	manifest := &googleBatchArchiveManifest{
		JobID:                job.ID,
		PublicBatchName:      strings.TrimSpace(job.PublicBatchName),
		PublicProtocol:       archivePublicProtocol(job),
		ExecutionProtocol:    archiveExecutionProtocol(job),
		ExecutionBatchName:   strings.TrimSpace(job.ExecutionBatchName),
		PublicResultFileName: archiveResultFileName(job, resultObject),
		ArchivedAt:           time.Now().UTC().Format(time.RFC3339),
		RetentionExpiresAt:   archiveRetentionRFC3339(job),
	}
	if resultObject != nil {
		manifest.ContentType = strings.TrimSpace(resultObject.ContentType)
		manifest.SizeBytes = resultObject.SizeBytes
		manifest.SHA256 = strings.TrimSpace(resultObject.SHA256)
	}
	return s.googleBatchArchiveStorage.StoreManifest(ctx, settings, job, manifest)
}

func (s *GeminiMessagesCompatService) findGoogleBatchArchiveResultObject(ctx context.Context, job *GoogleBatchArchiveJob) (*GoogleBatchArchiveObject, error) {
	if s == nil || s.googleBatchArchiveObjectRepo == nil || job == nil || job.ID <= 0 {
		return nil, nil
	}
	objects, err := s.googleBatchArchiveObjectRepo.ListByJobID(ctx, job.ID)
	if err != nil {
		return nil, err
	}
	for _, object := range objects {
		if object != nil && object.IsResultPayload {
			return object, nil
		}
	}
	return nil, nil
}

func (s *GeminiMessagesCompatService) buildGoogleBatchArchiveStatus(ctx context.Context, job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject) googleBatchArchiveResponse {
	resultFileName := archiveResultFileName(job, object)
	source := googleBatchArchiveSourceUnavailable
	downloadable := false
	switch {
	case object != nil && strings.TrimSpace(object.RelativePath) != "":
		source = googleBatchArchiveSourceLocal
		downloadable = true
	case s.canFetchGoogleBatchResultFromExecution(ctx, job, object):
		source = googleBatchArchiveSourceUpstreamFetchable
		downloadable = true
	case s.canFetchGoogleBatchResultFromOfficial(job):
		source = googleBatchArchiveSourceUpstreamFetchable
		downloadable = true
	}
	return googleBatchArchiveResponse{
		State:              defaultArchiveLifecycle(job),
		ResultFileName:     resultFileName,
		Downloadable:       downloadable,
		Source:             source,
		DownloadPath:       archiveDownloadPath(resultFileName),
		RetentionExpiresAt: archiveRetentionRFC3339(job),
	}
}

func defaultArchiveLifecycle(job *GoogleBatchArchiveJob) string {
	if job == nil {
		return GoogleBatchArchiveLifecyclePending
	}
	state := strings.TrimSpace(job.ArchiveState)
	if state == "" {
		return GoogleBatchArchiveLifecyclePending
	}
	return state
}

func (s *GeminiMessagesCompatService) buildArchiveBatchPayload(job *GoogleBatchArchiveJob, snapshotBody []byte, archive googleBatchArchiveResponse) []byte {
	payload := buildArchivedAIStudioBatchPayload(job, snapshotBody)
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil || body == nil {
		body = map[string]any{}
	}
	body["archive"] = archive
	result, err := json.Marshal(body)
	if err != nil {
		return payload
	}
	return result
}

func (s *GeminiMessagesCompatService) canFetchGoogleBatchResultFromOfficial(job *GoogleBatchArchiveJob) bool {
	if job == nil || strings.EqualFold(strings.TrimSpace(job.ExecutionProviderFamily), UpstreamProviderVertexAI) {
		return false
	}
	publicResultFileName := archiveResultFileName(job, nil)
	if publicResultFileName == "" {
		return false
	}
	return job.OfficialExpiresAt == nil || job.OfficialExpiresAt.After(time.Now().UTC())
}

func (s *GeminiMessagesCompatService) canFetchGoogleBatchResultFromExecution(ctx context.Context, job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject) bool {
	if job == nil {
		return false
	}
	if googleBatchArchiveTargetForJob(job) != googleBatchTargetVertex {
		return false
	}
	if archiveResultFileName(job, object) == "" {
		return false
	}
	profileID, _ := metadataString(job.MetadataJSON, "staging_profile_id")
	outputPrefix, _ := metadataString(job.MetadataJSON, "vertex_output_prefix_object")
	return strings.TrimSpace(profileID) != "" && strings.TrimSpace(outputPrefix) != ""
}

func (s *GeminiMessagesCompatService) ensureGoogleBatchArchiveResultStream(ctx context.Context, input GoogleBatchForwardInput, job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject, allowOfficialFetch bool, allowExecutionFetch bool) (*UpstreamHTTPStreamResult, *GoogleBatchArchiveObject, *Account, error) {
	if job == nil {
		return nil, object, nil, fmt.Errorf("archive job is required")
	}
	settings := s.getGoogleBatchArchiveSettings(ctx)
	if object == nil {
		object, _ = s.findGoogleBatchArchiveResultObject(ctx, job)
	}
	if object != nil && strings.TrimSpace(object.RelativePath) != "" {
		result, err := s.openGoogleBatchArchiveObjectStreamResult(settings, object, archiveFilenameForPublicResource(archiveResultFileName(job, object), googleBatchArchiveResultFilename))
		if err == nil && result != nil {
			recordGoogleBatchArchiveFetchSource("local")
			return result, object, s.lookupArchiveExecutionAccountByJob(ctx, job), nil
		}
	}
	account := s.lookupArchiveExecutionAccountByJob(ctx, job)
	if allowExecutionFetch && s.canFetchGoogleBatchResultFromExecution(ctx, job, object) {
		body, contentType, _, executionResourceName, fetchErr := s.fetchVertexBatchArchiveResultStream(ctx, job)
		if fetchErr == nil && body != nil {
			defer func() { _ = body.Close() }()
			object = ensureArchiveResultObject(job, object, archiveResultFileName(job, object))
			object.ExecutionResourceName = strings.TrimSpace(executionResourceName)
			if err := s.storeGoogleBatchArchiveObjectReader(ctx, settings, job, object, googleBatchArchiveResultFilename, contentType, body); err != nil {
				return nil, object, account, err
			}
			if err := s.maybeSettleGoogleBatchArchiveJobFromObject(ctx, input, account, job, settings, object); err != nil {
				return nil, object, account, err
			}
			result, err := s.openGoogleBatchArchiveObjectStreamResult(settings, object, archiveFilenameForPublicResource(object.PublicResourceName, googleBatchArchiveResultFilename))
			if err == nil && result != nil {
				recordGoogleBatchArchiveFetchSource("vertex")
			}
			return result, object, account, err
		}
	}
	if allowOfficialFetch && s.canFetchGoogleBatchResultFromOfficial(job) && account != nil {
		resultFileName := archiveResultFileName(job, object)
		downloadInput := googleBatchArchiveInputFromJob(job, http.MethodGet, googleBatchArchivePublicFileDownloadPath(resultFileName), "alt=media")
		upstream, err := s.forwardGoogleBatchToAccountStream(ctx, downloadInput, account, googleBatchTargetAIStudio)
		if err == nil && upstream != nil && upstream.StatusCode >= 200 && upstream.StatusCode < 300 {
			defer func() { _ = upstream.Body.Close() }()
			object = ensureArchiveResultObject(job, object, resultFileName)
			if err := s.storeGoogleBatchArchiveObjectReader(ctx, settings, job, object, archiveFilenameForPublicResource(resultFileName, googleBatchArchiveResultFilename), headerValue(upstream.Headers, "Content-Type"), upstream.Body); err != nil {
				return nil, object, account, err
			}
			if err := s.maybeSettleGoogleBatchArchiveJobFromObject(ctx, input, account, job, settings, object); err != nil {
				return nil, object, account, err
			}
			result, err := s.openGoogleBatchArchiveObjectStreamResult(settings, object, archiveFilenameForPublicResource(resultFileName, googleBatchArchiveResultFilename))
			if err == nil && result != nil {
				recordGoogleBatchArchiveFetchSource("official")
			}
			return result, object, account, err
		}
		if err == nil && upstream != nil {
			recordGoogleBatchArchiveFetchSource("official")
			return upstream, object, account, nil
		}
	}
	recordGoogleBatchArchiveFetchSource("unavailable")
	return nil, object, account, infraerrors.NotFound("GOOGLE_ARCHIVE_FILE_NOT_FOUND", "archive file not found")
}

func ensureArchiveResultObject(job *GoogleBatchArchiveJob, object *GoogleBatchArchiveObject, publicResultFileName string) *GoogleBatchArchiveObject {
	if object != nil {
		if strings.TrimSpace(object.PublicResourceName) == "" {
			object.PublicResourceName = strings.TrimSpace(publicResultFileName)
		}
		if strings.TrimSpace(object.ExecutionResourceName) == "" {
			object.ExecutionResourceName = strings.TrimSpace(publicResultFileName)
		}
		return object
	}
	return &GoogleBatchArchiveObject{
		JobID:                 job.ID,
		PublicResourceKind:    GoogleBatchArchiveResourceKindFile,
		PublicResourceName:    strings.TrimSpace(publicResultFileName),
		ExecutionResourceName: strings.TrimSpace(publicResultFileName),
		IsResultPayload:       true,
		MetadataJSON: map[string]any{
			"public_batch_name": job.PublicBatchName,
		},
	}
}
