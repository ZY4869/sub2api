package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"golang.org/x/sync/errgroup"
)

const googleBatchResponseReadLimit = 32 << 20
const googleBatchListFanoutLimit = 4
const googleBatchListRequestTimeout = 15 * time.Second

type GoogleBatchForwardInput struct {
	GroupID        *int64
	APIKeyID       int64
	APIKey         *APIKey
	UserID         int64
	BillingType    int8
	SubscriptionID *int64
	Method         string
	Path           string
	RawQuery       string
	Headers        http.Header
	Body           []byte
	OpenBody       func() (io.ReadCloser, error)
	ContentLength  int64
	AccountID      *int64
}

type googleBatchTarget string

const (
	googleBatchTargetAIStudio googleBatchTarget = "ai_studio"
	googleBatchTargetVertex   googleBatchTarget = "vertex"
)

func (s *GeminiMessagesCompatService) ForwardGoogleFiles(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	path := strings.TrimSpace(input.Path)
	switch {
	case strings.HasPrefix(path, "/download/v1beta/files/"):
		return s.ForwardGoogleFileDownload(ctx, input)
	case path == "/v1beta/files" && strings.EqualFold(input.Method, http.MethodGet):
		return s.forwardAggregatedAIStudioList(ctx, input, path, "files")
	case path == "/v1beta/files" || path == "/upload/v1beta/files" || path == "/v1beta/files:register":
		return s.forwardGoogleCreate(ctx, input, googleBatchTargetAIStudio, UpstreamResourceKindGeminiFile)
	case strings.HasPrefix(path, "/v1beta/files/"):
		return s.forwardAIStudioFileBoundResourceWithArchive(ctx, input)
	default:
		return nil, nil, infraerrors.NotFound("GOOGLE_FILES_PATH_UNSUPPORTED", "unsupported Gemini Files path")
	}
}

func (s *GeminiMessagesCompatService) ForwardGoogleBatches(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	path := strings.TrimSpace(input.Path)
	switch {
	case strings.HasPrefix(path, "/v1beta/models/") && strings.Contains(path, ":batchGenerateContent"):
		accountID, err := s.resolveAIStudioBatchCreateAccountID(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		input.AccountID = accountID
		return s.forwardGoogleBatchCreateWithArchive(ctx, input)
	case path == "/v1beta/batches" && strings.EqualFold(input.Method, http.MethodGet):
		return s.forwardAggregatedAIStudioList(ctx, input, path, "batches")
	case strings.HasPrefix(path, "/v1beta/batches/"):
		return s.forwardAIStudioBatchBoundResourceWithArchive(ctx, input)
	default:
		return nil, nil, infraerrors.NotFound("GOOGLE_BATCH_PATH_UNSUPPORTED", "unsupported Gemini Batch path")
	}
}

func (s *GeminiMessagesCompatService) ForwardVertexBatchPredictionJobs(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	projectID, location, jobName, err := parseVertexBatchPredictionJobPath(input.Path)
	if err != nil {
		return nil, nil, infraerrors.BadRequest("VERTEX_BATCH_PATH_INVALID", err.Error())
	}
	if jobName == "" && strings.EqualFold(input.Method, http.MethodGet) {
		return s.forwardAggregatedVertexList(ctx, input, projectID, location)
	}
	if strings.EqualFold(input.Method, http.MethodPost) && jobName == "" {
		selector, err := s.buildGoogleBatchSelector(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		selector.projectID = projectID
		selector.location = location
		selector.accountID = input.AccountID
		account, err := s.selectGoogleBatchAccount(ctx, input.GroupID, googleBatchTargetVertex, selector)
		if err != nil {
			return nil, nil, err
		}
		result, boundAccount, err := s.forwardAndBindGoogleBatch(ctx, input, account, googleBatchTargetVertex, UpstreamResourceKindVertexBatchJob)
		recordGoogleBatchCreateOutcome(err == nil && result != nil && result.StatusCode >= 200 && result.StatusCode < 300)
		return result, boundAccount, err
	}
	return s.forwardGoogleBoundResource(ctx, input, googleBatchTargetVertex, UpstreamResourceKindVertexBatchJob)
}

func (s *GeminiMessagesCompatService) forwardGoogleCreate(ctx context.Context, input GoogleBatchForwardInput, target googleBatchTarget, resourceKind string) (*UpstreamHTTPResult, *Account, error) {
	selector, err := s.buildGoogleBatchSelector(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	selector.accountID = input.AccountID
	account, err := s.selectGoogleBatchAccount(ctx, input.GroupID, target, selector)
	if err != nil {
		return nil, nil, err
	}
	return s.forwardAndBindGoogleBatch(ctx, input, account, target, resourceKind)
}

func (s *GeminiMessagesCompatService) forwardGoogleBoundResource(ctx context.Context, input GoogleBatchForwardInput, target googleBatchTarget, resourceKind string) (*UpstreamHTTPResult, *Account, error) {
	resourceName := extractResourceNameFromPath(resourceKind, input.Path)
	var binding *UpstreamResourceBinding
	if resourceName != "" && s.resourceBindingRepo != nil {
		binding, _ = s.resourceBindingRepo.Get(ctx, resourceKind, resourceName)
	}
	var selector *vertexBatchSelector
	if target == googleBatchTargetVertex {
		projectID, location, _, err := parseVertexBatchPredictionJobPath(input.Path)
		if err != nil {
			return nil, nil, infraerrors.BadRequest("VERTEX_BATCH_PATH_INVALID", err.Error())
		}
		selector = s.buildGoogleBatchSelectorBestEffort(ctx, input)
		selector.projectID = projectID
		selector.location = location
	}
	account, err := s.resolveGoogleBatchAccount(ctx, input.GroupID, target, binding, selector)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.forwardGoogleBatchToAccount(ctx, input, account, target)
	if err != nil {
		return nil, nil, err
	}
	if binding != nil && shouldSoftDeleteBinding(input.Method, input.Path) && resourceName != "" && s.resourceBindingRepo != nil {
		_ = s.resourceBindingRepo.SoftDelete(ctx, resourceKind, resourceName)
		s.releaseGoogleBatchQuota(ctx, resourceName, GoogleBatchQuotaReservationStatusReleased)
	}
	return result, account, nil
}

func (s *GeminiMessagesCompatService) forwardAndBindGoogleBatch(ctx context.Context, input GoogleBatchForwardInput, account *Account, target googleBatchTarget, resourceKind string) (*UpstreamHTTPResult, *Account, error) {
	result, err := s.forwardGoogleBatchToAccount(ctx, input, account, target)
	if err != nil {
		return nil, nil, err
	}
	createdMetadata := s.buildGoogleBatchCreatedBindingMetadata(ctx, input, target, resourceKind)
	if result.StatusCode >= 200 && result.StatusCode < 300 && s.resourceBindingRepo != nil {
		for _, resourceName := range extractCreatedResourceNames(resourceKind, result.Body) {
			accountID := account.ID
			apiKeyID := input.APIKeyID
			userID := input.UserID
			binding := &UpstreamResourceBinding{
				ResourceKind:   resourceKind,
				ResourceName:   resourceName,
				ProviderFamily: providerFamilyForTarget(target),
				AccountID:      accountID,
				APIKeyID:       &apiKeyID,
				GroupID:        input.GroupID,
				UserID:         &userID,
				MetadataJSON:   buildGoogleBatchBindingMetadata(createdMetadata),
			}
			if err := s.resourceBindingRepo.Upsert(ctx, binding); err != nil {
				return nil, nil, err
			}
			if resourceKind == UpstreamResourceKindGeminiBatch || resourceKind == UpstreamResourceKindVertexBatchJob {
				if err := s.reserveGoogleBatchQuota(ctx, input, account, target, resourceName); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	return result, account, nil
}

func (s *GeminiMessagesCompatService) forwardAggregatedAIStudioList(ctx context.Context, input GoogleBatchForwardInput, path string, listKey string) (*UpstreamHTTPResult, *Account, error) {
	accounts, err := s.listEligibleGoogleBatchAccounts(ctx, input.GroupID, googleBatchTargetAIStudio, nil)
	if err != nil {
		return nil, nil, err
	}
	_ = path
	return s.forwardAggregatedGoogleList(ctx, input, accounts, googleBatchTargetAIStudio, listKey)
}

func (s *GeminiMessagesCompatService) forwardAggregatedVertexList(ctx context.Context, input GoogleBatchForwardInput, projectID string, location string) (*UpstreamHTTPResult, *Account, error) {
	accounts, err := s.listEligibleGoogleBatchAccounts(ctx, input.GroupID, googleBatchTargetVertex, &vertexBatchSelector{
		projectID: projectID,
		location:  location,
	})
	if err != nil {
		return nil, nil, err
	}
	return s.forwardAggregatedGoogleList(ctx, input, accounts, googleBatchTargetVertex, "batchPredictionJobs")
}

func (s *GeminiMessagesCompatService) forwardAggregatedGoogleList(ctx context.Context, input GoogleBatchForwardInput, accounts []*Account, target googleBatchTarget, listKey string) (*UpstreamHTTPResult, *Account, error) {
	if len(accounts) == 0 {
		return nil, nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_NO_ACCOUNT", "no available Google batch accounts")
	}
	if len(accounts) == 1 {
		result, err := s.forwardGoogleBatchToAccount(ctx, input, accounts[0], target)
		return result, accounts[0], err
	}
	type googleBatchListResult struct {
		response *UpstreamHTTPResult
		err      error
	}
	startedAt := time.Now()
	defer recordGoogleBatchListFanoutLatency(time.Since(startedAt))
	results := make([]googleBatchListResult, len(accounts))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(googleBatchListFanoutLimit)
	for idx, account := range accounts {
		idx := idx
		account := account
		g.Go(func() error {
			requestCtx, cancel := context.WithTimeout(gctx, googleBatchListRequestTimeout)
			defer cancel()
			result, err := s.forwardGoogleBatchToAccount(requestCtx, input, account, target)
			results[idx] = googleBatchListResult{response: result, err: err}
			return nil
		})
	}
	_ = g.Wait()
	var (
		headers      http.Header
		lastErr      error
		orderedItems [][]map[string]any
	)
	orderedItems = make([][]map[string]any, len(accounts))
	for idx := range results {
		result := results[idx]
		if result.err != nil {
			lastErr = result.err
			continue
		}
		if result.response == nil {
			continue
		}
		if result.response.StatusCode >= 400 {
			lastErr = infraerrors.ServiceUnavailable("GOOGLE_BATCH_LIST_UPSTREAM_ERROR", "upstream list failed")
			continue
		}
		if headers == nil {
			headers = result.response.Headers.Clone()
		}
		orderedItems[idx] = extractNamedListItems(result.response.Body, listKey)
	}
	mergedItems := mergeGoogleBatchNamedListItems(orderedItems)
	if len(mergedItems) == 0 {
		if lastErr == nil {
			lastErr = infraerrors.ServiceUnavailable("GOOGLE_BATCH_LIST_EMPTY", "no upstream resources available")
		}
		return nil, nil, lastErr
	}
	payload := map[string]any{listKey: mergedItems}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Set("Content-Type", "application/json")
	return &UpstreamHTTPResult{StatusCode: http.StatusOK, Headers: headers, Body: body}, nil, nil
}

func (s *GeminiMessagesCompatService) resolveAIStudioBatchCreateAccountID(ctx context.Context, input GoogleBatchForwardInput) (*int64, error) {
	fileNames := uniqueStrings(collectStringFieldsByKey(input.Body, "fileName"))
	if len(fileNames) == 0 || s.resourceBindingRepo == nil {
		return nil, nil
	}
	bindings, err := s.resolveGoogleBatchReferencedFileBindings(ctx, fileNames)
	if err != nil {
		return nil, err
	}
	var accountID int64
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		if accountID == 0 {
			accountID = binding.AccountID
			continue
		}
		if binding.AccountID != accountID {
			return nil, infraerrors.Conflict("GEMINI_BATCH_FILE_ACCOUNT_CONFLICT", formatGeminiBatchConflictMessage(UpstreamResourceKindGeminiFile))
		}
	}
	if accountID == 0 {
		return nil, nil
	}
	return &accountID, nil
}

func (s *GeminiMessagesCompatService) buildGoogleBatchCreatedBindingMetadata(ctx context.Context, input GoogleBatchForwardInput, target googleBatchTarget, resourceKind string) map[string]any {
	resolvedMetadata, err := s.resolveGoogleBatchInputMetadata(ctx, input)
	if err != nil {
		resolvedMetadata = googleBatchResolvedInputMetadata{
			requestedModel:      strings.TrimSpace(extractGoogleBatchModelID(input.Path, input.Body)),
			modelFamily:         normalizeGoogleBatchModelFamily(extractGoogleBatchModelID(input.Path, input.Body)),
			estimatedTokens:     estimateGoogleBatchTokensFromPayload(input.Body),
			sourceProtocol:      publicGoogleBatchProtocol(input.Path),
			sourceResourceNames: uniqueStrings(collectStringFieldsByKey(input.Body, "fileName")),
		}
	}
	metadata := map[string]any{
		googleBatchBindingMetadataPublicProtocol:      publicGoogleBatchProtocol(input.Path),
		googleBatchBindingMetadataExecutionProtocol:   providerFamilyForTarget(target),
		"mirror_resource_name":                        "",
		"staging_profile_id":                          "",
		"staging_object_uri_masked":                   "",
		googleBatchBindingMetadataSourceResourceNames: resolvedMetadata.sourceResourceNames,
		googleBatchBindingMetadataEstimatedTokens:     resolvedMetadata.estimatedTokens,
		googleBatchBindingMetadataModelFamily:         resolvedMetadata.modelFamily,
		googleBatchBindingMetadataRequestedModel:      resolvedMetadata.requestedModel,
		googleBatchBindingMetadataSourceProtocol:      resolvedMetadata.sourceProtocol,
	}
	if resourceKind == UpstreamResourceKindGeminiFile {
		for key, value := range s.buildGoogleBatchFileBindingMetadata(input) {
			metadata[key] = value
		}
	}
	return metadata
}

func (s *GeminiMessagesCompatService) resolveGoogleBatchAccount(ctx context.Context, groupID *int64, target googleBatchTarget, binding *UpstreamResourceBinding, selector *vertexBatchSelector) (*Account, error) {
	if binding != nil {
		account, err := s.getSchedulableAccount(ctx, binding.AccountID)
		if err == nil && googleBatchAccountEligible(account, target, selector) && !account.IsQuotaExceeded() {
			return account, nil
		}
	}
	return s.selectGoogleBatchAccount(ctx, groupID, target, selector)
}

type vertexBatchSelector struct {
	projectID       string
	location        string
	accountID       *int64
	modelFamily     string
	estimatedTokens int64
}

func (s *GeminiMessagesCompatService) selectGoogleBatchAccount(ctx context.Context, groupID *int64, target googleBatchTarget, selector *vertexBatchSelector) (*Account, error) {
	accounts, err := s.listEligibleGoogleBatchAccounts(ctx, groupID, target, selector)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, infraerrors.ServiceUnavailable("GOOGLE_BATCH_NO_ACCOUNT", "no available Google batch accounts")
	}
	return accounts[0], nil
}

func (s *GeminiMessagesCompatService) listEligibleGoogleBatchAccounts(ctx context.Context, groupID *int64, target googleBatchTarget, selector *vertexBatchSelector) ([]*Account, error) {
	accounts, err := s.listSchedulableAccountsOnce(ctx, groupID, PlatformGemini, false)
	if err != nil {
		return nil, err
	}
	var eligible []*Account
	for i := range accounts {
		account := &accounts[i]
		if selector != nil && selector.accountID != nil && *selector.accountID > 0 && account.ID != *selector.accountID {
			continue
		}
		if account.IsQuotaExceeded() || !googleBatchAccountEligible(account, target, selector) || !s.googleBatchAccountHasQuotaCapacity(ctx, account, target, selector) {
			continue
		}
		eligible = append(eligible, account)
	}
	sortGoogleBatchAccounts(eligible, target)
	return eligible, nil
}

func googleBatchAccountEligible(account *Account, target googleBatchTarget, selector *vertexBatchSelector) bool {
	if account == nil {
		return false
	}
	switch target {
	case googleBatchTargetAIStudio:
		return SupportsAIStudioBatch(account)
	case googleBatchTargetVertex:
		if !SupportsVertexBatch(account) {
			return false
		}
		if account.IsGeminiVertexAI() && selector != nil {
			return strings.EqualFold(strings.TrimSpace(account.GetGeminiVertexProjectID()), strings.TrimSpace(selector.projectID)) &&
				strings.EqualFold(normalizeVertexLocation(account.GetGeminiVertexLocation()), normalizeVertexLocation(selector.location))
		}
		return true
	default:
		return false
	}
}

func sortGoogleBatchAccounts(accounts []*Account, target googleBatchTarget) {
	for i := 0; i < len(accounts); i++ {
		for j := i + 1; j < len(accounts); j++ {
			if compareGoogleBatchAccount(accounts[j], accounts[i], target) < 0 {
				accounts[i], accounts[j] = accounts[j], accounts[i]
			}
		}
	}
}

func compareGoogleBatchAccount(left *Account, right *Account, target googleBatchTarget) int {
	leftRank := googleBatchAccountRank(left, target)
	rightRank := googleBatchAccountRank(right, target)
	if leftRank != rightRank {
		return leftRank - rightRank
	}
	if left.Priority != right.Priority {
		return left.Priority - right.Priority
	}
	switch {
	case left.LastUsedAt == nil && right.LastUsedAt != nil:
		return -1
	case left.LastUsedAt != nil && right.LastUsedAt == nil:
		return 1
	case left.LastUsedAt != nil && right.LastUsedAt != nil:
		if left.LastUsedAt.Before(*right.LastUsedAt) {
			return -1
		}
		if right.LastUsedAt.Before(*left.LastUsedAt) {
			return 1
		}
	}
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	return 0
}

func googleBatchAccountRank(account *Account, target googleBatchTarget) int {
	if account == nil {
		return 99
	}
	switch target {
	case googleBatchTargetAIStudio:
		if account.GeminiBatchCapability() == GeminiBatchCapabilityAIStudio {
			return 0
		}
	case googleBatchTargetVertex:
		if account.GeminiBatchCapability() == GeminiBatchCapabilityVertex {
			return 0
		}
	}
	if SupportsProtocolGatewayGeminiBatch(account) {
		return 1
	}
	return 9
}

func (s *GeminiMessagesCompatService) forwardGoogleBatchToAccount(ctx context.Context, input GoogleBatchForwardInput, account *Account, target googleBatchTarget) (*UpstreamHTTPResult, error) {
	account = ResolveProtocolGatewayInboundAccount(account, PlatformGemini)
	req, proxyURL, err := s.buildGoogleBatchRequest(ctx, account, target, input)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, googleBatchResponseReadLimit))
	if err != nil {
		return nil, err
	}
	filteredHeaders := responseheaders.FilterHeaders(resp.Header, s.responseHeaderFilter)
	_ = s.accountRepo.UpdateLastUsed(ctx, account.ID)
	return &UpstreamHTTPResult{StatusCode: resp.StatusCode, Headers: filteredHeaders, Body: body}, nil
}

func (s *GeminiMessagesCompatService) forwardGoogleBatchToAccountStream(ctx context.Context, input GoogleBatchForwardInput, account *Account, target googleBatchTarget) (*UpstreamHTTPStreamResult, error) {
	account = ResolveProtocolGatewayInboundAccount(account, PlatformGemini)
	req, proxyURL, err := s.buildGoogleBatchRequest(ctx, account, target, input)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, err
	}
	filteredHeaders := responseheaders.FilterHeaders(resp.Header, s.responseHeaderFilter)
	_ = s.accountRepo.UpdateLastUsed(ctx, account.ID)
	return &UpstreamHTTPStreamResult{
		StatusCode:    resp.StatusCode,
		Headers:       filteredHeaders,
		Body:          resp.Body,
		ContentLength: resp.ContentLength,
	}, nil
}

func (s *GeminiMessagesCompatService) buildGoogleBatchRequest(ctx context.Context, account *Account, target googleBatchTarget, input GoogleBatchForwardInput) (*http.Request, string, error) {
	if account == nil {
		return nil, "", infraerrors.BadRequest("GOOGLE_BATCH_ACCOUNT_NIL", "account is nil")
	}
	baseURL, err := s.googleBatchBaseURL(account, target)
	if err != nil {
		return nil, "", err
	}
	fullURL := strings.TrimRight(baseURL, "/") + strings.TrimSpace(input.Path)
	if strings.TrimSpace(input.RawQuery) != "" {
		fullURL += "?" + strings.TrimPrefix(strings.TrimSpace(input.RawQuery), "?")
	}
	body, err := input.OpenRequestBody()
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(strings.TrimSpace(input.Method)), fullURL, body)
	if err != nil {
		if body != nil {
			_ = body.Close()
		}
		return nil, "", err
	}
	if input.ContentLength > 0 || (input.ContentLength == 0 && len(input.Body) == 0) {
		req.ContentLength = input.ContentLength
	}
	copyGoogleForwardHeaders(req.Header, input.Headers)
	if err := s.applyGoogleBatchAuth(ctx, req, account); err != nil {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, "", err
	}
	var proxyURL string
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return req, proxyURL, nil
}

func (input GoogleBatchForwardInput) OpenRequestBody() (io.ReadCloser, error) {
	if input.OpenBody != nil {
		return input.OpenBody()
	}
	if len(input.Body) == 0 {
		return http.NoBody, nil
	}
	return io.NopCloser(bytes.NewReader(input.Body)), nil
}

func mergeGoogleBatchNamedListItems(orderedItems [][]map[string]any) []map[string]any {
	seen := make(map[string]map[string]any)
	for _, items := range orderedItems {
		for _, item := range items {
			name := strings.TrimSpace(stringMapValue(item, "name"))
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = item
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	merged := make([]map[string]any, 0, len(names))
	for _, name := range names {
		merged = append(merged, seen[name])
	}
	return merged
}

func (s *GeminiMessagesCompatService) googleBatchBaseURL(account *Account, target googleBatchTarget) (string, error) {
	switch {
	case SupportsProtocolGatewayGeminiBatch(account):
		return s.validateUpstreamBaseURL(account.GetGeminiBaseURL(geminicli.AIStudioBaseURL))
	case target == googleBatchTargetAIStudio:
		return s.validateUpstreamBaseURL(account.GetGeminiBaseURL(geminicli.AIStudioBaseURL))
	case target == googleBatchTargetVertex:
		return s.validateUpstreamBaseURL(account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL))
	default:
		return "", infraerrors.BadRequest("GOOGLE_BATCH_TARGET_INVALID", "invalid Google batch target")
	}
}

func (s *GeminiMessagesCompatService) applyGoogleBatchAuth(ctx context.Context, req *http.Request, account *Account) error {
	switch account.Type {
	case AccountTypeAPIKey:
		apiKey := strings.TrimSpace(account.GetCredential("api_key"))
		if apiKey == "" {
			return infraerrors.BadRequest("GOOGLE_BATCH_API_KEY_MISSING", "gemini api_key not configured")
		}
		req.Header.Set("x-goog-api-key", apiKey)
		req.Header.Del("Authorization")
		return nil
	case AccountTypeOAuth:
		if s.tokenProvider == nil {
			return infraerrors.ServiceUnavailable("GOOGLE_BATCH_TOKEN_PROVIDER_MISSING", "gemini token provider not configured")
		}
		accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Del("x-goog-api-key")
		return nil
	default:
		return infraerrors.BadRequest("GOOGLE_BATCH_ACCOUNT_TYPE_UNSUPPORTED", "unsupported account type for Google batch")
	}
}

func copyGoogleForwardHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "authorization", "x-goog-api-key", "host", "content-length", "connection", "proxy-connection", "keep-alive", "transfer-encoding", "upgrade", "te", "trailer", "proxy-authenticate", "proxy-authorization":
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func shouldSoftDeleteBinding(method string, path string) bool {
	return strings.EqualFold(strings.TrimSpace(method), http.MethodDelete) &&
		(strings.HasPrefix(path, "/v1beta/files/") || strings.HasPrefix(path, "/v1beta/batches/") || strings.Contains(path, "/batchPredictionJobs/"))
}

func providerFamilyForTarget(target googleBatchTarget) string {
	if target == googleBatchTargetVertex {
		return UpstreamProviderVertexAI
	}
	return UpstreamProviderAIStudio
}

func extractResourceNameFromPath(resourceKind string, path string) string {
	switch resourceKind {
	case UpstreamResourceKindGeminiFile:
		return extractAIStudioResourceName(path, "/v1beta/files/")
	case UpstreamResourceKindGeminiBatch:
		return extractAIStudioResourceName(path, "/v1beta/batches/")
	case UpstreamResourceKindVertexBatchJob:
		projectID, location, jobName, err := parseVertexBatchPredictionJobPath(path)
		if err != nil || jobName == "" {
			return ""
		}
		return buildVertexBatchJobResourceName(projectID, location, jobName)
	default:
		return ""
	}
}

func extractAIStudioResourceName(path string, prefix string) string {
	trimmed := strings.TrimSpace(path)
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	resourceID := strings.TrimPrefix(trimmed, prefix)
	for _, sep := range []string{":", "/"} {
		if idx := strings.Index(resourceID, sep); idx >= 0 {
			resourceID = resourceID[:idx]
		}
	}
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return ""
	}
	base := strings.Trim(strings.TrimPrefix(strings.TrimSuffix(prefix, "/"), "/v1beta/"), "/")
	return base + "/" + resourceID
}

func parseVertexBatchPredictionJobPath(path string) (string, string, string, error) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 6 || parts[0] != "v1" || parts[1] != "projects" || parts[3] != "locations" || parts[5] != "batchPredictionJobs" {
		return "", "", "", fmt.Errorf("unsupported Vertex batchPredictionJobs path")
	}
	projectID := strings.TrimSpace(parts[2])
	location := strings.TrimSpace(parts[4])
	jobName := ""
	if len(parts) >= 7 && len(parts) > 6 {
		jobName = strings.TrimSpace(parts[6])
		if idx := strings.Index(jobName, ":"); idx >= 0 {
			jobName = jobName[:idx]
		}
	}
	return projectID, location, jobName, nil
}

func buildVertexBatchJobResourceName(projectID string, location string, jobName string) string {
	return "projects/" + strings.TrimSpace(projectID) + "/locations/" + strings.TrimSpace(location) + "/batchPredictionJobs/" + strings.TrimSpace(jobName)
}

func extractCreatedResourceNames(resourceKind string, body []byte) []string {
	switch resourceKind {
	case UpstreamResourceKindGeminiFile:
		return append(extractTopLevelNames(body), extractListNames(body, "files")...)
	case UpstreamResourceKindGeminiBatch:
		return append(extractTopLevelNames(body), extractListNames(body, "batches")...)
	case UpstreamResourceKindVertexBatchJob:
		return append(extractTopLevelNames(body), extractListNames(body, "batchPredictionJobs")...)
	default:
		return nil
	}
}

func extractTopLevelNames(body []byte) []string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	name := strings.TrimSpace(stringMapValue(payload, "name"))
	if name == "" {
		return nil
	}
	return []string{name}
}

func extractListNames(body []byte, listKey string) []string {
	items := extractNamedListItems(body, listKey)
	names := make([]string, 0, len(items))
	for _, item := range items {
		if name := strings.TrimSpace(stringMapValue(item, "name")); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func extractNamedListItems(body []byte, listKey string) []map[string]any {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	rawItems, ok := payload[listKey].([]any)
	if !ok {
		return nil
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]any)
		if ok {
			items = append(items, item)
		}
	}
	return items
}

func collectStringFieldsByKey(body []byte, key string) []string {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	var values []string
	collectStringFieldsByKeyRecursive(payload, key, &values)
	return values
}

func collectStringFieldsByKeyRecursive(value any, key string, out *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for itemKey, itemValue := range typed {
			if strings.EqualFold(itemKey, key) {
				switch item := itemValue.(type) {
				case string:
					if strings.TrimSpace(item) != "" {
						*out = append(*out, strings.TrimSpace(item))
					}
				case []any:
					for _, nested := range item {
						if str, ok := nested.(string); ok && strings.TrimSpace(str) != "" {
							*out = append(*out, strings.TrimSpace(str))
						}
					}
				}
			}
			collectStringFieldsByKeyRecursive(itemValue, key, out)
		}
	case []any:
		for _, item := range typed {
			collectStringFieldsByKeyRecursive(item, key, out)
		}
	}
}

func stringMapValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}
