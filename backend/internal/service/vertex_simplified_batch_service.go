package service

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type simplifiedVertexBatchTarget struct {
	account       *Account
	executionName string
	path          string
}

func (s *GeminiMessagesCompatService) ForwardSimplifiedVertexBatchPredictionJobs(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	parsedPath, err := parseSimplifiedVertexBatchPath(input.Path)
	if err != nil {
		return nil, nil, err
	}
	switch {
	case parsedPath.jobName == "" && strings.EqualFold(input.Method, http.MethodGet):
		return s.forwardSimplifiedVertexBatchList(ctx, input)
	case parsedPath.jobName == "" && strings.EqualFold(input.Method, http.MethodPost):
		return s.forwardSimplifiedVertexBatchCreate(ctx, input)
	case parsedPath.jobName != "":
		return s.forwardSimplifiedVertexBatchJob(ctx, input, parsedPath)
	default:
		return nil, nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BATCH_PATH_INVALID", "invalid simplified Vertex batch path")
	}
}

func (s *GeminiMessagesCompatService) forwardSimplifiedVertexBatchList(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	accounts, err := s.listSimplifiedVertexBatchAccounts(ctx, input.GroupID)
	if err != nil {
		return nil, nil, err
	}
	merged := make(map[string]map[string]any)
	for _, account := range accounts {
		listInput := input
		listInput.Path = buildSimplifiedVertexBatchAccountPath(account, "", "")
		result, forwardErr := s.forwardGoogleBatchToAccount(ctx, listInput, account, googleBatchTargetVertex)
		if forwardErr != nil || result == nil || result.StatusCode >= http.StatusBadRequest {
			continue
		}
		for _, item := range extractNamedListItems(result.Body, "batchPredictionJobs") {
			rewriteSimplifiedVertexBatchNameField(item)
			name := strings.TrimSpace(stringMapValue(item, "name"))
			if name != "" {
				merged[name] = item
			}
		}
	}
	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]map[string]any, 0, len(names))
	for _, name := range names {
		items = append(items, merged[name])
	}
	body, err := json.Marshal(map[string]any{"batchPredictionJobs": items})
	if err != nil {
		return nil, nil, err
	}
	return s.buildGoogleBatchJSONResult(http.StatusOK, body), nil, nil
}

func (s *GeminiMessagesCompatService) forwardSimplifiedVertexBatchCreate(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	spec, err := s.buildSimplifiedVertexBatchCreateSpec(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	account, err := s.selectSimplifiedVertexBatchCreateAccount(ctx, input, spec)
	if err != nil {
		return nil, nil, err
	}
	createInput := input
	createInput.Path = buildSimplifiedVertexBatchAccountPath(account, "", "")
	createInput.RawQuery = ""
	createInput.Body = spec.body
	createInput.ContentLength = int64(len(spec.body))
	result, err := s.forwardGoogleBatchToAccount(ctx, createInput, account, googleBatchTargetVertex)
	if err != nil {
		return nil, nil, err
	}
	if result != nil && result.StatusCode >= http.StatusOK && result.StatusCode < http.StatusMultipleChoices {
		s.persistSimplifiedVertexBatchArchive(ctx, input, account, spec.requestedModel, spec.managedOutput, result)
		result.Body = rewriteSimplifiedVertexBatchBody(result.Body)
	}
	return result, account, nil
}

func (s *GeminiMessagesCompatService) forwardSimplifiedVertexBatchJob(ctx context.Context, input GoogleBatchForwardInput, parsedPath simplifiedVertexBatchPath) (GoogleBatchUpstreamResult, *Account, error) {
	target, err := s.resolveSimplifiedVertexBatchTarget(ctx, input.GroupID, parsedPath.jobName)
	if err != nil {
		return nil, nil, err
	}
	jobInput := input
	jobInput.Path = buildSimplifiedVertexBatchAccountPath(target.account, parsedPath.jobName, parsedPath.action)
	result, err := s.forwardGoogleBatchToAccount(ctx, jobInput, target.account, googleBatchTargetVertex)
	if err != nil {
		return nil, nil, err
	}
	if result != nil && result.StatusCode >= http.StatusOK && result.StatusCode < http.StatusMultipleChoices {
		result.Body = rewriteSimplifiedVertexBatchBody(result.Body)
		if strings.EqualFold(input.Method, http.MethodDelete) && s.resourceBindingRepo != nil {
			_ = s.resourceBindingRepo.SoftDelete(ctx, UpstreamResourceKindVertexBatchJob, target.executionName)
			s.releaseGoogleBatchQuota(ctx, target.executionName, GoogleBatchQuotaReservationStatusReleased)
		}
	}
	return result, target.account, nil
}

func (s *GeminiMessagesCompatService) resolveSimplifiedVertexBatchTarget(ctx context.Context, groupID *int64, jobName string) (*simplifiedVertexBatchTarget, error) {
	jobName = strings.TrimSpace(jobName)
	if jobName == "" {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BATCH_PATH_INVALID", "invalid simplified Vertex batch path")
	}
	if s.googleBatchArchiveJobRepo != nil {
		if job, _ := s.googleBatchArchiveJobRepo.GetByPublicBatchName(ctx, "batches/"+jobName); job != nil {
			if account := s.lookupArchiveExecutionAccountByJob(ctx, job); account != nil {
				return &simplifiedVertexBatchTarget{account: account, executionName: job.ExecutionBatchName, path: googleBatchArchiveVertexBatchPath(job.ExecutionBatchName)}, nil
			}
		}
	}
	accounts, err := s.listSimplifiedVertexBatchAccounts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 1 {
		path := buildSimplifiedVertexBatchAccountPath(accounts[0], jobName, "")
		return &simplifiedVertexBatchTarget{account: accounts[0], executionName: buildVertexBatchJobResourceName(accounts[0].GetGeminiVertexProjectID(), accounts[0].GetGeminiVertexLocation(), jobName), path: path}, nil
	}
	if s.resourceBindingRepo == nil {
		return nil, infraerrors.NotFound("VERTEX_SIMPLIFIED_BATCH_NOT_FOUND", "Vertex batch job not found")
	}
	candidateNames := make([]string, 0, len(accounts))
	accountByResource := make(map[string]*Account, len(accounts))
	for _, account := range accounts {
		resourceName := buildVertexBatchJobResourceName(account.GetGeminiVertexProjectID(), account.GetGeminiVertexLocation(), jobName)
		candidateNames = append(candidateNames, resourceName)
		accountByResource[strings.ToLower(resourceName)] = account
	}
	bindings, err := s.resourceBindingRepo.GetByNames(ctx, UpstreamResourceKindVertexBatchJob, candidateNames)
	if err != nil {
		return nil, err
	}
	if len(bindings) == 1 && bindings[0] != nil {
		resourceName := strings.TrimSpace(bindings[0].ResourceName)
		account := accountByResource[strings.ToLower(resourceName)]
		if account == nil {
			account, _ = s.getSchedulableAccount(ctx, bindings[0].AccountID)
		}
		if account == nil {
			return nil, infraerrors.NotFound("VERTEX_SIMPLIFIED_BATCH_NOT_FOUND", "Vertex batch job not found")
		}
		return &simplifiedVertexBatchTarget{account: account, executionName: resourceName, path: googleBatchArchiveVertexBatchPath(resourceName)}, nil
	}
	if len(bindings) > 1 {
		return nil, infraerrors.Conflict("VERTEX_SIMPLIFIED_BATCH_AMBIGUOUS", "Vertex batch job is ambiguous across multiple accounts")
	}
	return nil, infraerrors.NotFound("VERTEX_SIMPLIFIED_BATCH_NOT_FOUND", "Vertex batch job not found")
}

func (s *GeminiMessagesCompatService) listSimplifiedVertexBatchAccounts(ctx context.Context, groupID *int64) ([]*Account, error) {
	accounts, err := s.listEligibleGoogleBatchAccounts(ctx, groupID, googleBatchTargetVertex, nil)
	if err != nil {
		return nil, err
	}
	filtered := make([]*Account, 0, len(accounts))
	for _, account := range accounts {
		if account != nil && strings.TrimSpace(buildVertexBatchPredictionJobsPath(account)) != "" {
			filtered = append(filtered, account)
		}
	}
	return filtered, nil
}

func (s *GeminiMessagesCompatService) selectSimplifiedVertexBatchCreateAccount(ctx context.Context, input GoogleBatchForwardInput, spec *simplifiedVertexBatchCreateSpec) (*Account, error) {
	if spec == nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_INVALID", "invalid simplified Vertex batch body")
	}
	selectorInput := input
	selectorInput.Path = "/v1/vertex/batchPredictionJobs"
	selectorInput.RawQuery = ""
	selectorInput.Body = spec.body
	selectorInput.ContentLength = int64(len(spec.body))

	selector, err := s.buildGoogleBatchSelector(ctx, selectorInput)
	if err != nil {
		return nil, err
	}
	selector.accountID = input.AccountID

	accounts, err := s.listEligibleGoogleBatchAccounts(ctx, input.GroupID, googleBatchTargetVertex, selector)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if account != nil && strings.TrimSpace(buildVertexBatchPredictionJobsPath(account)) != "" {
			return account, nil
		}
	}
	return nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_NO_ACCOUNT", "no available Google batch accounts")
}
