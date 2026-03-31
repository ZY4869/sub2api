package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/tidwall/gjson"
)

func (s *GeminiMessagesCompatService) ForwardGoogleFileDownload(ctx context.Context, input GoogleBatchForwardInput) (*UpstreamHTTPResult, *Account, error) {
	resourceName := extractAIStudioDownloadFileName(input.Path)
	if resourceName == "" {
		return nil, nil, infraerrors.NotFound("GOOGLE_FILE_DOWNLOAD_NOT_FOUND", "archive file not found")
	}
	object, _ := s.getGoogleBatchArchiveObject(ctx, GoogleBatchArchiveResourceKindFile, resourceName)

	var binding *UpstreamResourceBinding
	if s.resourceBindingRepo != nil {
		binding, _ = s.resourceBindingRepo.Get(ctx, UpstreamResourceKindGeminiFile, resourceName)
	}
	job := s.lookupArchiveJobForObject(ctx, object)
	if job == nil && bindingArchiveJobID(binding) > 0 && s.googleBatchArchiveJobRepo != nil {
		job, _ = s.googleBatchArchiveJobRepo.GetByID(ctx, bindingArchiveJobID(binding))
	}
	virtualResource := bindingVirtualResource(binding) || archiveVirtualResource(job) || strings.EqualFold(bindingExecutionProtocol(binding), UpstreamProviderVertexAI)
	settings := s.getGoogleBatchArchiveSettings(ctx)
	if !virtualResource {
		account, err := s.resolveGoogleBatchAccount(ctx, input.GroupID, googleBatchTargetAIStudio, binding, nil)
		if err == nil && account != nil {
			result, forwardErr := s.forwardGoogleBatchToAccount(ctx, input, account, googleBatchTargetAIStudio)
			if forwardErr == nil && result != nil && result.StatusCode >= 200 && result.StatusCode < 300 {
				if job != nil {
					_ = s.touchArchiveAccess(ctx, job)
					if object != nil && object.RelativePath == "" && s.googleBatchArchiveStorage != nil {
						filename := archiveFilenameForPublicResource(resourceName, googleBatchArchiveResultFilename)
						_ = s.storeGoogleBatchArchiveObjectBytes(ctx, settings, job, object, filename, headerValue(result.Headers, "Content-Type"), result.Body)
						_ = s.maybeSettleGoogleBatchArchiveJob(ctx, input, account, job, result.Body)
					}
				}
				s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, binding), UsageOperationOfficialResultDownload, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-download:"+resourceName+":"+generateRequestID())
				return result, account, nil
			}
			if result != nil && result.StatusCode != http.StatusNotFound && object == nil {
				return result, account, nil
			}
		}
	}

	if job != nil {
		body, updatedObject, account, err := s.ensureGoogleBatchArchiveResult(ctx, input, job, object, false, virtualResource)
		if err == nil {
			_ = s.touchArchiveAccess(ctx, job)
			if account != nil {
				s.recordArchiveDownloadUsage(ctx, input, account, job)
			}
			return s.buildGoogleBatchBinaryResult(updatedObject.ContentType, archiveFilenameForPublicResource(resourceName, googleBatchArchiveResultFilename), body), account, nil
		}
	}
	if object != nil && strings.TrimSpace(object.RelativePath) != "" && s.googleBatchArchiveStorage != nil {
		body, err := s.googleBatchArchiveStorage.ReadAll(settings, object.RelativePath)
		if err == nil {
			account := s.lookupArchiveExecutionAccount(ctx, object, binding)
			if job != nil {
				_ = s.touchArchiveAccess(ctx, job)
				if account != nil {
					s.recordArchiveDownloadUsage(ctx, input, account, job)
				}
			}
			return s.buildGoogleBatchBinaryResult(object.ContentType, archiveFilenameForPublicResource(resourceName, googleBatchArchiveResultFilename), body), account, nil
		}
	}
	return nil, nil, infraerrors.NotFound("GOOGLE_FILE_DOWNLOAD_NOT_FOUND", "archive file not found")
}

func (s *GeminiMessagesCompatService) ForwardGoogleArchiveBatch(ctx context.Context, input GoogleBatchForwardInput) (*UpstreamHTTPResult, *Account, error) {
	resourceName := extractAIStudioArchiveBatchName(input.Path)
	if resourceName == "" {
		return nil, nil, infraerrors.NotFound("GOOGLE_BATCH_ARCHIVE_NOT_FOUND", "archive batch not found")
	}
	job, err := s.getGoogleBatchArchiveJobByPublicBatchName(ctx, resourceName)
	if err != nil || job == nil {
		return nil, nil, infraerrors.NotFound("GOOGLE_BATCH_ARCHIVE_NOT_FOUND", "archive batch not found")
	}
	settings := s.getGoogleBatchArchiveSettings(ctx)
	var snapshotBody []byte
	if object, _ := s.getGoogleBatchArchiveObject(ctx, GoogleBatchArchiveResourceKindBatch, resourceName); object != nil && strings.TrimSpace(object.RelativePath) != "" && s.googleBatchArchiveStorage != nil {
		snapshotBody, _ = s.googleBatchArchiveStorage.ReadAll(settings, object.RelativePath)
	}
	account := s.lookupArchiveExecutionAccountByJob(ctx, job)
	if account != nil {
		switch googleBatchArchiveTargetForJob(job) {
		case googleBatchTargetVertex:
			vertexInput := googleBatchArchiveInputFromJob(job, http.MethodGet, googleBatchArchiveVertexBatchPath(job.ExecutionBatchName), "")
			if result, err := s.forwardGoogleBatchToAccount(ctx, vertexInput, account, googleBatchTargetVertex); err == nil && result != nil && result.StatusCode >= 200 && result.StatusCode < 300 {
				snapshotBody = translateVertexBatchPayloadToAIStudio(job, result.Body)
				_ = s.syncArchiveJobFromBatchPayload(ctx, vertexInput, account, job.PublicBatchName, snapshotBody)
			}
		case googleBatchTargetAIStudio:
			if s.canFetchGoogleBatchResultFromOfficial(job) || job.OfficialExpiresAt == nil || job.OfficialExpiresAt.After(time.Now().UTC()) {
				batchInput := googleBatchArchiveInputFromJob(job, http.MethodGet, googleBatchArchivePublicBatchPath(job.PublicBatchName), "")
				if result, err := s.forwardGoogleBatchToAccount(ctx, batchInput, account, googleBatchTargetAIStudio); err == nil && result != nil && result.StatusCode >= 200 && result.StatusCode < 300 {
					snapshotBody = result.Body
					_ = s.syncArchiveJobFromBatchPayload(ctx, batchInput, account, job.PublicBatchName, snapshotBody)
				}
			}
		}
	}
	resultObject, _ := s.findGoogleBatchArchiveResultObject(ctx, job)
	return s.buildGoogleBatchJSONResult(http.StatusOK, s.buildArchiveBatchPayload(job, snapshotBody, s.buildGoogleBatchArchiveStatus(ctx, job, resultObject))), account, nil
}

func (s *GeminiMessagesCompatService) ForwardGoogleArchiveFileDownload(ctx context.Context, input GoogleBatchForwardInput) (*UpstreamHTTPResult, *Account, error) {
	resourceName := extractAIStudioArchiveFileName(input.Path)
	if resourceName == "" {
		return nil, nil, infraerrors.NotFound("GOOGLE_ARCHIVE_FILE_NOT_FOUND", "archive file not found")
	}
	object, err := s.getGoogleBatchArchiveObject(ctx, GoogleBatchArchiveResourceKindFile, resourceName)
	var binding *UpstreamResourceBinding
	if s.resourceBindingRepo != nil {
		binding, _ = s.resourceBindingRepo.Get(ctx, UpstreamResourceKindGeminiFile, resourceName)
	}
	var job *GoogleBatchArchiveJob
	if object != nil {
		job = s.lookupArchiveJobForObject(ctx, object)
	}
	if job == nil && bindingArchiveJobID(binding) > 0 && s.googleBatchArchiveJobRepo != nil {
		job, _ = s.googleBatchArchiveJobRepo.GetByID(ctx, bindingArchiveJobID(binding))
	}
	if job == nil {
		return nil, nil, infraerrors.NotFound("GOOGLE_ARCHIVE_FILE_NOT_FOUND", "archive file not found")
	}
	body, object, account, err := s.ensureGoogleBatchArchiveResult(ctx, input, job, object, true, true)
	if err != nil {
		return nil, account, err
	}
	_ = s.touchArchiveAccess(ctx, job)
	if account != nil {
		s.recordArchiveDownloadUsage(ctx, input, account, job)
	}
	return s.buildGoogleBatchBinaryResult(object.ContentType, archiveFilenameForPublicResource(resourceName, googleBatchArchiveResultFilename), body), account, nil
}

func (s *GeminiMessagesCompatService) forwardGoogleBatchCreateWithArchive(ctx context.Context, input GoogleBatchForwardInput) (*UpstreamHTTPResult, *Account, error) {
	selector := buildGoogleBatchSelectorFromInput(input)
	selector.accountID = input.AccountID
	accounts, err := s.listEligibleGoogleBatchAccounts(ctx, input.GroupID, googleBatchTargetAIStudio, selector)
	if err != nil {
		return nil, nil, err
	}
	overflowSelection, _ := s.resolveAIStudioOverflowSelection(ctx, input, selector)
	if len(accounts) == 0 {
		if overflowSelection != nil {
			return s.forwardGoogleBatchCreateViaVertexOverflow(ctx, input, overflowSelection)
		}
		return nil, nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_NO_ACCOUNT", "no available Google batch accounts")
	}
	var lastQuotaResult *UpstreamHTTPResult
	var lastQuotaAccount *Account
	for _, account := range accounts {
		result, _, forwardErr := s.forwardAndBindGoogleBatch(ctx, input, account, googleBatchTargetAIStudio, UpstreamResourceKindGeminiBatch)
		if forwardErr != nil {
			return nil, nil, forwardErr
		}
		if result == nil {
			continue
		}
		if result.StatusCode >= 200 && result.StatusCode < 300 {
			if err := s.archiveGoogleBatchCreateResult(ctx, input, account, googleBatchTargetAIStudio, UpstreamResourceKindGeminiBatch, result); err != nil {
				return nil, nil, err
			}
			s.recordGoogleBatchUsageEvent(ctx, input, account, extractGoogleBatchModelID(input.Path, input.Body), UsageOperationBatchCreate, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-create:"+generateRequestID())
			return result, account, nil
		}
		if isGoogleBatchQuotaFallbackResponse(result) {
			lastQuotaResult = result
			lastQuotaAccount = account
			continue
		}
		return result, account, nil
	}
	if lastQuotaResult != nil && overflowSelection != nil {
		return s.forwardGoogleBatchCreateViaVertexOverflow(ctx, input, overflowSelection)
	}
	if lastQuotaResult != nil {
		return lastQuotaResult, lastQuotaAccount, nil
	}
	return nil, nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_CREATE_FAILED", "google batch create failed")
}

func (s *GeminiMessagesCompatService) archiveGoogleBatchCreateResult(ctx context.Context, input GoogleBatchForwardInput, account *Account, target googleBatchTarget, resourceKind string, result *UpstreamHTTPResult) error {
	if account == nil || result == nil || result.StatusCode < 200 || result.StatusCode >= 300 || target != googleBatchTargetAIStudio || resourceKind != UpstreamResourceKindGeminiBatch {
		return nil
	}
	if !account.IsBatchArchiveEnabled() || s.googleBatchArchiveJobRepo == nil {
		return nil
	}
	names := extractCreatedResourceNames(resourceKind, result.Body)
	if len(names) == 0 {
		return nil
	}
	settings := s.getGoogleBatchArchiveSettings(ctx)
	now := time.Now().UTC()
	requestedModel := extractGoogleBatchModelID(input.Path, input.Body)
	for _, publicBatchName := range names {
		nextPollAt := now.Add(time.Duration(settings.PollMinIntervalSeconds) * time.Second)
		prefetchAt := now.Add(time.Duration(settings.PrefetchAfterHours) * time.Hour)
		retentionAt := now.AddDate(0, 0, account.GetBatchArchiveRetentionDays())
		job := &GoogleBatchArchiveJob{
			PublicBatchName:         publicBatchName,
			PublicProtocol:          GoogleBatchArchivePublicProtocolAIStudio,
			ExecutionProviderFamily: UpstreamProviderAIStudio,
			ExecutionBatchName:      publicBatchName,
			SourceAccountID:         account.ID,
			ExecutionAccountID:      account.ID,
			APIKeyID:                int64Ptr(input.APIKeyID),
			GroupID:                 input.GroupID,
			UserID:                  int64Ptr(input.UserID),
			RequestedModel:          requestedModel,
			ConversionDirection:     GoogleBatchArchiveConversionNone,
			State:                   strings.TrimSpace(gjson.GetBytes(result.Body, "state").String()),
			OfficialExpiresAt:       timePtr(now.Add(48 * time.Hour)),
			PrefetchDueAt:           &prefetchAt,
			NextPollAt:              &nextPollAt,
			ArchiveState:            GoogleBatchArchiveLifecyclePending,
			BillingSettlementState:  GoogleBatchArchiveBillingPending,
			RetentionExpiresAt:      &retentionAt,
			MetadataJSON: buildGoogleBatchBindingMetadata(map[string]any{
				googleBatchBindingMetadataPublicProtocol:      UpstreamProviderAIStudio,
				googleBatchBindingMetadataExecutionProtocol:   UpstreamProviderAIStudio,
				googleBatchBindingMetadataVirtualResource:     false,
				googleBatchBindingMetadataConversionDirection: GoogleBatchArchiveConversionNone,
				"requested_model":                             requestedModel,
				"source_resource_names":                       uniqueStrings(collectStringFieldsByKey(input.Body, "fileName")),
			}),
		}
		if job.State == "" {
			job.State = GoogleBatchArchiveStateCreated
		}
		if err := s.upsertGoogleBatchArchiveJob(ctx, job); err != nil {
			return err
		}
		stored, err := s.googleBatchArchiveJobRepo.GetByPublicBatchName(ctx, publicBatchName)
		if err == nil && stored != nil {
			job = stored
		}
		if err := s.storeGoogleBatchSnapshot(ctx, settings, job, result.Body); err != nil {
			return err
		}
		if err := s.enrichBindingMetadata(ctx, UpstreamResourceKindGeminiBatch, publicBatchName, map[string]any{
			googleBatchBindingMetadataArchiveJobID:         job.ID,
			googleBatchBindingMetadataPublicProtocol:       UpstreamProviderAIStudio,
			googleBatchBindingMetadataExecutionProtocol:    UpstreamProviderAIStudio,
			googleBatchBindingMetadataVirtualResource:      false,
			googleBatchBindingMetadataConversionDirection:  GoogleBatchArchiveConversionNone,
			googleBatchBindingMetadataPublicResultFileName: "",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *GeminiMessagesCompatService) forwardAIStudioBatchBoundResourceWithArchive(ctx context.Context, input GoogleBatchForwardInput) (*UpstreamHTTPResult, *Account, error) {
	resourceName := extractResourceNameFromPath(UpstreamResourceKindGeminiBatch, input.Path)
	if resourceName == "" {
		return s.forwardGoogleBoundResource(ctx, input, googleBatchTargetAIStudio, UpstreamResourceKindGeminiBatch)
	}
	job, _ := s.getGoogleBatchArchiveJobByPublicBatchName(ctx, resourceName)
	var binding *UpstreamResourceBinding
	if s.resourceBindingRepo != nil {
		binding, _ = s.resourceBindingRepo.Get(ctx, UpstreamResourceKindGeminiBatch, resourceName)
	}
	if job == nil && bindingArchiveJobID(binding) > 0 && s.googleBatchArchiveJobRepo != nil {
		job, _ = s.googleBatchArchiveJobRepo.GetByID(ctx, bindingArchiveJobID(binding))
	}
	if job != nil && (archiveVirtualResource(job) || bindingVirtualResource(binding) || strings.EqualFold(job.ExecutionProviderFamily, UpstreamProviderVertexAI)) {
		account := s.lookupArchiveExecutionAccountByJob(ctx, job)
		if account != nil {
			vertexInput := googleBatchArchiveInputFromJob(job, http.MethodGet, googleBatchArchiveVertexBatchPath(job.ExecutionBatchName), "")
			result, err := s.forwardGoogleBatchToAccount(ctx, vertexInput, account, googleBatchTargetVertex)
			if err == nil && result != nil && result.StatusCode >= 200 && result.StatusCode < 300 {
				body := translateVertexBatchPayloadToAIStudio(job, result.Body)
				_ = s.syncArchiveJobFromBatchPayload(ctx, vertexInput, account, resourceName, body)
				s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, binding), UsageOperationBatchStatus, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-status:"+resourceName+":"+generateRequestID())
				return s.buildGoogleBatchJSONResult(http.StatusOK, body), account, nil
			}
		}
	}
	result, account, err := s.forwardGoogleBoundResource(ctx, input, googleBatchTargetAIStudio, UpstreamResourceKindGeminiBatch)
	if err == nil && result != nil && strings.EqualFold(input.Method, http.MethodGet) && result.StatusCode >= 200 && result.StatusCode < 300 {
		account = resolveNilAccount(account, func() *Account { return s.lookupArchiveExecutionAccountByJob(ctx, job) })
		if account != nil {
			_ = s.syncArchiveJobFromBatchPayload(ctx, input, account, resourceName, result.Body)
			s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, binding), UsageOperationBatchStatus, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-status:"+resourceName+":"+generateRequestID())
		}
		return result, account, nil
	}
	if strings.EqualFold(input.Method, http.MethodGet) && job != nil && s.googleBatchArchiveStorage != nil {
		settings := s.getGoogleBatchArchiveSettings(ctx)
		var body []byte
		if object, _ := s.getGoogleBatchArchiveObject(ctx, GoogleBatchArchiveResourceKindBatch, resourceName); object != nil && strings.TrimSpace(object.RelativePath) != "" {
			body, _ = s.googleBatchArchiveStorage.ReadAll(settings, object.RelativePath)
		}
		account = s.lookupArchiveExecutionAccountByJob(ctx, job)
		if account != nil {
			s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, binding), UsageOperationBatchStatus, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-status-local:"+resourceName+":"+generateRequestID())
		}
		return s.buildGoogleBatchJSONResult(http.StatusOK, buildArchivedAIStudioBatchPayload(job, body)), account, nil
	}
	return result, account, err
}

func (s *GeminiMessagesCompatService) forwardAIStudioFileBoundResourceWithArchive(ctx context.Context, input GoogleBatchForwardInput) (*UpstreamHTTPResult, *Account, error) {
	resourceName := extractResourceNameFromPath(UpstreamResourceKindGeminiFile, input.Path)
	if resourceName == "" {
		return s.forwardGoogleBoundResource(ctx, input, googleBatchTargetAIStudio, UpstreamResourceKindGeminiFile)
	}
	var binding *UpstreamResourceBinding
	if s.resourceBindingRepo != nil {
		binding, _ = s.resourceBindingRepo.Get(ctx, UpstreamResourceKindGeminiFile, resourceName)
	}
	object, _ := s.getGoogleBatchArchiveObject(ctx, GoogleBatchArchiveResourceKindFile, resourceName)
	job := s.lookupArchiveJobForObject(ctx, object)
	if job == nil && bindingArchiveJobID(binding) > 0 && s.googleBatchArchiveJobRepo != nil {
		job, _ = s.googleBatchArchiveJobRepo.GetByID(ctx, bindingArchiveJobID(binding))
	}
	if job != nil && (archiveVirtualResource(job) || bindingVirtualResource(binding) || strings.EqualFold(bindingExecutionProtocol(binding), UpstreamProviderVertexAI)) {
		account := s.lookupArchiveExecutionAccount(ctx, object, binding)
		if account != nil {
			s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, binding), UsageOperationGetFileMetadata, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-file-meta-virtual:"+resourceName+":"+generateRequestID())
		}
		return s.buildGoogleBatchJSONResult(http.StatusOK, buildArchivedAIStudioFilePayload(job, ensureArchiveResultObject(job, object, resourceName))), account, nil
	}
	result, account, err := s.forwardGoogleBoundResource(ctx, input, googleBatchTargetAIStudio, UpstreamResourceKindGeminiFile)
	if err == nil && result != nil && strings.EqualFold(input.Method, http.MethodGet) && result.StatusCode >= 200 && result.StatusCode < 300 {
		account = resolveNilAccount(account, func() *Account { return s.lookupArchiveExecutionAccount(ctx, object, nil) })
		if account != nil {
			s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, binding), UsageOperationGetFileMetadata, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-file-meta:"+resourceName+":"+generateRequestID())
		}
		return result, account, nil
	}
	if strings.EqualFold(input.Method, http.MethodGet) && object != nil {
		account = s.lookupArchiveExecutionAccount(ctx, object, nil)
		if account != nil {
			s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, binding), UsageOperationGetFileMetadata, UsageChargeSourceNone, UsageTokens{}, &CostBreakdown{}, "google-batch-file-meta-local:"+resourceName+":"+generateRequestID())
		}
		return s.buildGoogleBatchJSONResult(http.StatusOK, buildArchivedAIStudioFilePayload(job, object)), account, nil
	}
	return result, account, err
}

func (s *GeminiMessagesCompatService) syncArchiveJobFromBatchPayload(ctx context.Context, input GoogleBatchForwardInput, account *Account, publicBatchName string, payload []byte) error {
	job, _ := s.getGoogleBatchArchiveJobByPublicBatchName(ctx, publicBatchName)
	if job == nil {
		if !account.IsBatchArchiveEnabled() {
			return nil
		}
		return s.archiveGoogleBatchCreateResult(ctx, input, account, googleBatchTargetAIStudio, UpstreamResourceKindGeminiBatch, s.buildGoogleBatchJSONResult(http.StatusOK, payload))
	}
	settings := s.getGoogleBatchArchiveSettings(ctx)
	if job.MetadataJSON == nil {
		job.MetadataJSON = map[string]any{}
	}
	state := strings.TrimSpace(gjson.GetBytes(payload, "state").String())
	if state != "" {
		job.State = state
	}
	if isGoogleBatchTerminalState(job.State) {
		job.NextPollAt = nil
	} else {
		nextPollAt := time.Now().UTC().Add(time.Duration(settings.PollMinIntervalSeconds) * time.Second)
		job.NextPollAt = &nextPollAt
	}
	if fileName := extractGoogleBatchResultFileName(payload, job); fileName != "" {
		job.MetadataJSON[googleBatchBindingMetadataPublicResultFileName] = fileName
		officialResultName := fileName
		if archiveVirtualResource(job) || strings.EqualFold(strings.TrimSpace(job.ExecutionProviderFamily), UpstreamProviderVertexAI) {
			officialResultName = ""
		}
		job.MetadataJSON[googleBatchBindingMetadataOfficialResultName] = officialResultName
		object := &GoogleBatchArchiveObject{
			JobID:                 job.ID,
			PublicResourceKind:    GoogleBatchArchiveResourceKindFile,
			PublicResourceName:    fileName,
			ExecutionResourceName: fileName,
			ContentType:           "application/json",
			IsResultPayload:       true,
			MetadataJSON:          map[string]any{"public_batch_name": publicBatchName},
		}
		if archiveVirtualResource(job) || strings.EqualFold(strings.TrimSpace(job.ExecutionProviderFamily), UpstreamProviderVertexAI) {
			object.ExecutionResourceName = strings.TrimSpace(object.ExecutionResourceName)
			object.MetadataJSON["staging_profile_id"] = job.MetadataJSON["staging_profile_id"]
			object.MetadataJSON["vertex_output_prefix_object"] = job.MetadataJSON["vertex_output_prefix_object"]
		}
		if err := s.upsertGoogleBatchArchiveObject(ctx, object); err != nil {
			return err
		}
		_ = s.enrichBindingMetadata(ctx, UpstreamResourceKindGeminiBatch, publicBatchName, map[string]any{
			googleBatchBindingMetadataPublicResultFileName: fileName,
			googleBatchBindingMetadataOfficialResultName:   officialResultName,
		})
		if s.resourceBindingRepo != nil {
			accountID := account.ID
			apiKeyID := input.APIKeyID
			userID := input.UserID
			_ = s.resourceBindingRepo.Upsert(ctx, &UpstreamResourceBinding{
				ResourceKind:   UpstreamResourceKindGeminiFile,
				ResourceName:   fileName,
				ProviderFamily: strings.TrimSpace(job.ExecutionProviderFamily),
				AccountID:      accountID,
				APIKeyID:       &apiKeyID,
				GroupID:        input.GroupID,
				UserID:         &userID,
				MetadataJSON: buildGoogleBatchBindingMetadata(map[string]any{
					googleBatchBindingMetadataArchiveJobID:         job.ID,
					googleBatchBindingMetadataPublicProtocol:       archivePublicProtocol(job),
					googleBatchBindingMetadataExecutionProtocol:    archiveExecutionProtocol(job),
					googleBatchBindingMetadataVirtualResource:      archiveVirtualResource(job),
					"public_batch_name":                            publicBatchName,
					googleBatchBindingMetadataPublicResultFileName: fileName,
					googleBatchBindingMetadataOfficialResultName:   officialResultName,
					googleBatchBindingMetadataConversionDirection:  job.ConversionDirection,
				}),
			})
		}
	}
	if err := s.upsertGoogleBatchArchiveJob(ctx, job); err != nil {
		return err
	}
	if err := s.storeGoogleBatchSnapshot(ctx, settings, job, payload); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(job.ExecutionProviderFamily), UpstreamProviderVertexAI) && strings.EqualFold(strings.TrimSpace(job.State), GoogleBatchArchiveStateSucceeded) {
		downloadInput := googleBatchArchiveInputFromJob(job, http.MethodGet, googleBatchArchivePublicFileDownloadPath(archiveResultFileName(job, nil)), "alt=media")
		if _, _, _, err := s.ensureGoogleBatchArchiveResult(ctx, downloadInput, job, nil, false, true); err != nil {
			return err
		}
	}
	return nil
}

func (s *GeminiMessagesCompatService) maybeSettleGoogleBatchArchiveJob(ctx context.Context, input GoogleBatchForwardInput, account *Account, job *GoogleBatchArchiveJob, payload []byte) error {
	if job == nil || account == nil || job.BillingSettlementState == GoogleBatchArchiveBillingSettled {
		return nil
	}
	tokens := googleBatchAggregateUsageFromJSONL(payload)
	cost := s.calculateGoogleBatchSettlementCost(job.RequestedModel, account, tokens)
	s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, nil), UsageOperationBatchSettlement, UsageChargeSourceModelBatch, tokens, cost, "google-batch-settlement:"+job.PublicBatchName)
	job.BillingSettlementState = GoogleBatchArchiveBillingSettled
	if err := s.upsertGoogleBatchArchiveJob(ctx, job); err != nil {
		return err
	}
	return nil
}

func (s *GeminiMessagesCompatService) touchArchiveAccess(ctx context.Context, job *GoogleBatchArchiveJob) error {
	if job == nil || s.googleBatchArchiveJobRepo == nil {
		return nil
	}
	return s.googleBatchArchiveJobRepo.TouchLastPublicResultAccess(ctx, job.ID, time.Now().UTC())
}

func (s *GeminiMessagesCompatService) recordArchiveDownloadUsage(ctx context.Context, input GoogleBatchForwardInput, account *Account, job *GoogleBatchArchiveJob) {
	if account == nil || job == nil {
		return
	}
	cost := &CostBreakdown{}
	chargeSource := UsageChargeSourceNone
	if account.GetBatchArchiveBillingMode() == GoogleBatchArchiveBillingModeArchiveCharge && account.GetBatchArchiveDownloadPriceUSD() > 0 {
		cost.TotalCost = account.GetBatchArchiveDownloadPriceUSD()
		cost.ActualCost = account.GetBatchArchiveDownloadPriceUSD()
		chargeSource = UsageChargeSourceArchiveDownload
	}
	s.recordGoogleBatchUsageEvent(ctx, input, account, archiveRequestedModel(job, nil), UsageOperationLocalArchiveDownload, chargeSource, UsageTokens{}, cost, "google-batch-local-download:"+job.PublicBatchName+":"+generateRequestID())
}

func extractGoogleBatchResultFileName(payload []byte, job *GoogleBatchArchiveJob) string {
	for _, path := range []string{
		"dest.fileName",
		"destination.fileName",
		"result.fileName",
		"output.fileName",
		"responseFile.fileName",
		"response_file.file_name",
	} {
		if value := strings.TrimSpace(gjson.GetBytes(payload, path).String()); value != "" {
			return value
		}
	}
	fileNames := uniqueStrings(collectStringFieldsByKey(payload, "fileName"))
	sourceNames := map[string]struct{}{}
	if job != nil {
		if raw, ok := job.MetadataJSON["source_resource_names"].([]string); ok {
			for _, value := range raw {
				sourceNames[strings.TrimSpace(value)] = struct{}{}
			}
		}
		if raw, ok := job.MetadataJSON["source_resource_names"].([]any); ok {
			for _, value := range raw {
				if text, ok := value.(string); ok {
					sourceNames[strings.TrimSpace(text)] = struct{}{}
				}
			}
		}
	}
	for _, value := range fileNames {
		if _, exists := sourceNames[strings.TrimSpace(value)]; !exists {
			return value
		}
	}
	return ""
}

func extractAIStudioDownloadFileName(path string) string {
	return extractAIStudioResourceName(path, "/download/v1beta/files/")
}

func extractAIStudioArchiveBatchName(path string) string {
	return extractAIStudioResourceName(path, "/google/batch/archive/v1beta/batches/")
}

func extractAIStudioArchiveFileName(path string) string {
	return extractAIStudioResourceName(path, "/google/batch/archive/v1beta/files/")
}

func lookupArchiveAccountID(binding *UpstreamResourceBinding, object *GoogleBatchArchiveObject, job *GoogleBatchArchiveJob) int64 {
	switch {
	case binding != nil && binding.AccountID > 0:
		return binding.AccountID
	case job != nil && job.ExecutionAccountID > 0:
		return job.ExecutionAccountID
	case object != nil:
		if value, ok := metadataInt64(object.MetadataJSON, "execution_account_id"); ok {
			return value
		}
	}
	return 0
}

func (s *GeminiMessagesCompatService) lookupArchiveExecutionAccount(ctx context.Context, object *GoogleBatchArchiveObject, binding *UpstreamResourceBinding) *Account {
	job := s.lookupArchiveJobForObject(ctx, object)
	if job != nil {
		return s.lookupArchiveExecutionAccountByJob(ctx, job)
	}
	accountID := lookupArchiveAccountID(binding, object, job)
	if accountID <= 0 {
		return nil
	}
	account, err := s.getSchedulableAccount(ctx, accountID)
	if err == nil && account != nil {
		return account
	}
	account, err = s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil
	}
	return account
}

func (s *GeminiMessagesCompatService) lookupArchiveExecutionAccountByJob(ctx context.Context, job *GoogleBatchArchiveJob) *Account {
	if job == nil || job.ExecutionAccountID <= 0 {
		return nil
	}
	account, err := s.getSchedulableAccount(ctx, job.ExecutionAccountID)
	if err == nil && account != nil {
		return account
	}
	account, err = s.accountRepo.GetByID(ctx, job.ExecutionAccountID)
	if err != nil {
		return nil
	}
	return account
}

func (s *GeminiMessagesCompatService) lookupArchiveJobForObject(ctx context.Context, object *GoogleBatchArchiveObject) *GoogleBatchArchiveJob {
	if object == nil || object.JobID <= 0 || s.googleBatchArchiveJobRepo == nil {
		return nil
	}
	job, err := s.googleBatchArchiveJobRepo.GetByID(ctx, object.JobID)
	if err != nil {
		return nil
	}
	return job
}

func archiveRequestedModel(job *GoogleBatchArchiveJob, binding *UpstreamResourceBinding) string {
	if job != nil && strings.TrimSpace(job.RequestedModel) != "" {
		return job.RequestedModel
	}
	if binding != nil {
		if value, ok := metadataString(binding.MetadataJSON, "requested_model"); ok {
			return value
		}
	}
	return ""
}

func bindingMetadata(binding *UpstreamResourceBinding) map[string]any {
	if binding == nil || binding.MetadataJSON == nil {
		return map[string]any{}
	}
	binding.MetadataJSON = normalizeGoogleBatchBindingMetadata(binding.MetadataJSON)
	return binding.MetadataJSON
}

func headerValue(headers http.Header, key string) string {
	if headers == nil {
		return ""
	}
	return strings.TrimSpace(headers.Get(key))
}

func archiveFilenameForPublicResource(resourceName string, fallback string) string {
	trimmed := strings.TrimSpace(resourceName)
	if trimmed == "" {
		return fallback
	}
	if idx := strings.LastIndex(trimmed, "/"); idx >= 0 && idx+1 < len(trimmed) {
		return trimmed[idx+1:] + ".jsonl"
	}
	return fallback
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func resolveNilAccount(account *Account, fallback func() *Account) *Account {
	if account != nil {
		return account
	}
	if fallback == nil {
		return nil
	}
	return fallback()
}

func isGoogleBatchTerminalState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "SUCCEEDED", "FAILED", "CANCELLED", "JOB_STATE_SUCCEEDED", "JOB_STATE_FAILED", "JOB_STATE_CANCELLED":
		return true
	default:
		return false
	}
}
