package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
	account = ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)
	startTime := time.Now()
	ctx = EnsureRequestMetadata(ctx)
	restrictionResult := s.detectCodexClientRestriction(c, account)
	apiKeyID := getAPIKeyIDFromContext(c)
	logCodexCLIOnlyDetection(ctx, c, account, apiKeyID, restrictionResult, body)
	if restrictionResult.Enabled && !restrictionResult.Matched {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"type": "forbidden_error", "message": "This account only allows Codex official clients"}})
		return nil, errors.New("codex_cli_only restriction: only codex official clients are allowed")
	}
	originalBody := body
	reqModel, reqStream, promptCacheKey := extractOpenAIRequestMetaFromBody(body)
	originalModel := reqModel
	runtimeRequestedModel := originalModel
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		if sourceModel := strings.TrimSpace(entry.SourceModelID); sourceModel != "" {
			runtimeRequestedModel = sourceModel
		}
	}
	topLevelEffort := strings.TrimSpace(gjson.GetBytes(body, "effortLevel").String())
	claudeCapability := RecordClaudeCapabilityMetadataRequestedOnly(ctx, runtimeRequestedModel, topLevelEffort)
	reqModel = firstNonEmptyString(claudeCapability.RequestedModelNormalized, runtimeRequestedModel, reqModel)
	routingModel := ResolveGatewaySelectionModelFromContext(ctx, reqModel)
	if routingModel == "" {
		routingModel = reqModel
	}
	forceImageHostRouting := false
	if strings.EqualFold(strings.TrimSpace(originalModel), OpenAICompatImageTargetModel) {
		if _, hasImageTool := DetectOpenAIResponsesImageGenerationToolModel(body); hasImageTool {
			routingModel = OpenAICompatImageHostModel
			forceImageHostRouting = true
		}
	}
	isCodexCLI := openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator")) || (s.cfg != nil && s.cfg.Gateway.ForceCodexCLI)
	simulatedClient := ""
	resolveSimulatedClient := func(model string) string {
		route := MatchGatewayClientRoute(account, PlatformOpenAI, model)
		if route == nil {
			return ""
		}
		return route.ClientProfile
	}
	if profile := resolveSimulatedClient(account.GetMappedModel(routingModel)); profile == GatewayClientProfileCodex {
		simulatedClient = profile
		isCodexCLI = true
	}
	wsDecision := s.getOpenAIWSProtocolResolver().Resolve(account)
	clientTransport := GetOpenAIClientTransport(c)
	wsDecision = resolveOpenAIWSDecisionByClientTransport(wsDecision, clientTransport)
	if c != nil {
		c.Set("openai_ws_transport_decision", string(wsDecision.Transport))
		c.Set("openai_ws_transport_reason", wsDecision.Reason)
	}
	if wsDecision.Transport == OpenAIUpstreamTransportResponsesWebsocketV2 {
		logOpenAIWSModeDebug("selected account_id=%d account_type=%s transport=%s reason=%s model=%s stream=%v", account.ID, account.Type, normalizeOpenAIWSLogValue(string(wsDecision.Transport)), normalizeOpenAIWSLogValue(wsDecision.Reason), reqModel, reqStream)
	}
	if wsDecision.Transport == OpenAIUpstreamTransportResponsesWebsocket {
		if c != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "OpenAI WSv1 is temporarily unsupported. Please enable responses_websockets_v2."}})
		}
		return nil, errors.New("openai ws v1 is temporarily unsupported; use ws v2")
	}
	passthroughEnabled := account.IsOpenAIPassthroughEnabled()
	if passthroughEnabled {
		if reqModel != "" && reqModel != strings.TrimSpace(originalModel) {
			if nextBody, setErr := sjson.SetBytes(originalBody, "model", reqModel); setErr == nil {
				originalBody = nextBody
			}
		}
		if routingModel != "" && routingModel != reqModel {
			if nextBody, setErr := sjson.SetBytes(originalBody, "model", routingModel); setErr == nil {
				originalBody = nextBody
			}
			reqModel = routingModel
		}
		normalizedBody, effortResolution, normalizeErr := normalizeOpenAIRequestBodyEffortBytes(originalBody, reqModel)
		if normalizeErr == nil {
			originalBody = normalizedBody
		}
		result, forwardErr := s.forwardOpenAIPassthrough(ctx, c, account, originalBody, originalModel, reqModel, effortResolution, reqStream, startTime)
		if result != nil {
			result.SimulatedClient = simulatedClient
			applyClaudeCapabilityToOpenAIForwardResult(result, claudeCapability)
		}
		return result, forwardErr
	}
	reqBody, err := getOpenAIRequestBodyMap(c, body)
	if err != nil {
		return nil, err
	}
	normalizedRequestModelPatched := false
	if v, ok := reqBody["model"].(string); ok {
		originalModel = v
		runtimeRequestedModel = v
		if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
			if sourceModel := strings.TrimSpace(entry.SourceModelID); sourceModel != "" {
				runtimeRequestedModel = sourceModel
			}
		}
		reqModel = firstNonEmptyString(claudeCapability.RequestedModelNormalized, NormalizeRequestedModelForClaudeCapability(runtimeRequestedModel))
		if reqModel != "" && reqModel != v {
			reqBody["model"] = reqModel
			normalizedRequestModelPatched = true
		}
		routingModel = ResolveGatewaySelectionModelFromContext(ctx, reqModel)
		if routingModel == "" {
			routingModel = reqModel
		}
	}
	if forceImageHostRouting {
		routingModel = OpenAICompatImageHostModel
	}
	if v, ok := reqBody["stream"].(bool); ok {
		reqStream = v
	}
	if promptCacheKey == "" {
		if v, ok := reqBody["prompt_cache_key"].(string); ok {
			promptCacheKey = strings.TrimSpace(v)
		}
	}
	effortResolution := normalizeOpenAIRequestBodyEffort(reqBody, originalModel)
	bodyModified := false
	patchDisabled := false
	patchHasOp := false
	patchDelete := false
	patchPath := ""
	var patchValue any
	markPatchSet := func(path string, value any) {
		if strings.TrimSpace(path) == "" {
			patchDisabled = true
			return
		}
		if patchDisabled {
			return
		}
		if !patchHasOp {
			patchHasOp = true
			patchDelete = false
			patchPath = path
			patchValue = value
			return
		}
		if patchDelete || patchPath != path {
			patchDisabled = true
			return
		}
		patchValue = value
	}
	markPatchDelete := func(path string) {
		if strings.TrimSpace(path) == "" {
			patchDisabled = true
			return
		}
		if patchDisabled {
			return
		}
		if !patchHasOp {
			patchHasOp = true
			patchDelete = true
			patchPath = path
			return
		}
		if !patchDelete || patchPath != path {
			patchDisabled = true
		}
	}
	disablePatch := func() {
		patchDisabled = true
	}
	if normalizedRequestModelPatched {
		bodyModified = true
		markPatchSet("model", reqModel)
	}
	if routingModel != "" && routingModel != reqModel {
		reqBody["model"] = routingModel
		bodyModified = true
		markPatchSet("model", routingModel)
		reqModel = routingModel
	}
	if isInstructionsEmpty(reqBody) {
		reqBody["instructions"] = "You are a helpful coding assistant."
		bodyModified = true
		markPatchSet("instructions", "You are a helpful coding assistant.")
	}
	if effortResolution.Effective != nil && (effortResolution.Source == effortSourceOpenAIField || effortResolution.Source == effortSourceOpenAIAlias || effortResolution.Source == effortSourceTopLevel) {
		bodyModified = true
		disablePatch()
	}
	mappedModel := account.GetMappedModel(reqModel)
	if mappedModel != reqModel {
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Model mapping applied: %s -> %s (account: %s, isCodexCLI: %v)", reqModel, mappedModel, account.Name, isCodexCLI)
		reqBody["model"] = mappedModel
		bodyModified = true
		markPatchSet("model", mappedModel)
	}
	if model, ok := reqBody["model"].(string); ok {
		upstreamModel := normalizeOpenAIModelForUpstream(account, model)
		if upstreamModel != "" && upstreamModel != model {
			logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Upstream model resolved: %s -> %s (account: %s, type: %s, isCodexCLI: %v)", model, upstreamModel, account.Name, account.Type, isCodexCLI)
			reqBody["model"] = upstreamModel
			mappedModel = upstreamModel
			bodyModified = true
			markPatchSet("model", upstreamModel)
		}
		if !SupportsVerbosity(mappedModel) {
			if text, ok := reqBody["text"].(map[string]any); ok {
				delete(text, "verbosity")
			}
		}
	}
	if reasoning, ok := reqBody["reasoning"].(map[string]any); ok {
		if effort, ok := reasoning["effort"].(string); ok && effort == "minimal" {
			reasoning["effort"] = "none"
			bodyModified = true
			markPatchSet("reasoning.effort", "none")
			logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Normalized reasoning.effort: minimal -> none (account: %s)", account.Name)
		}
	}
	if isChatGPTOpenAIOAuthAccount(account) {
		codexResult := applyCodexOAuthTransform(reqBody, isCodexCLI, isOpenAIResponsesCompactPath(c))
		if codexResult.Modified {
			bodyModified = true
			disablePatch()
		}
		if codexResult.NormalizedModel != "" {
			mappedModel = codexResult.NormalizedModel
		}
		if codexResult.PromptCacheKey != "" {
			promptCacheKey = codexResult.PromptCacheKey
		}
	}
	if sanitizeEmptyBase64InputImagesInOpenAIRequestBodyMap(reqBody) {
		bodyModified = true
		disablePatch()
	}

	// Enforce OpenAI Fast/Flex policy (service_tier) after all model/codex normalization.
	if policyModified, policyErr := s.applyOpenAIFastPolicyToRequestBodyMap(ctx, account, reqBody); policyErr != nil {
		msg := "This request is blocked by policy"
		setOpsUpstreamError(c, http.StatusForbidden, msg, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           RoutingPlatformForAccount(account),
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: http.StatusForbidden,
			Kind:               "policy_block",
			Message:            msg,
			Detail:             policyErr.Error(),
		})
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"type": "forbidden_error", "code": "openai_fast_policy_blocked", "message": msg}})
		return nil, policyErr
	} else if policyModified {
		bodyModified = true
		markPatchDelete("service_tier")
	}
	if profile := resolveSimulatedClient(mappedModel); profile == GatewayClientProfileCodex {
		simulatedClient = profile
		isCodexCLI = true
	}
	runtimePlatform := EffectiveProtocol(account)
	if !isCodexCLI {
		if maxOutputTokens, hasMaxOutputTokens := reqBody["max_output_tokens"]; hasMaxOutputTokens {
			switch runtimePlatform {
			case PlatformOpenAI:
				if account.Type == AccountTypeAPIKey {
					delete(reqBody, "max_output_tokens")
					bodyModified = true
					markPatchDelete("max_output_tokens")
				}
			case PlatformAnthropic:
				delete(reqBody, "max_output_tokens")
				markPatchDelete("max_output_tokens")
				if _, hasMaxTokens := reqBody["max_tokens"]; !hasMaxTokens {
					reqBody["max_tokens"] = maxOutputTokens
					disablePatch()
				}
				bodyModified = true
			case PlatformGemini:
				delete(reqBody, "max_output_tokens")
				bodyModified = true
				markPatchDelete("max_output_tokens")
			default:
				delete(reqBody, "max_output_tokens")
				bodyModified = true
				markPatchDelete("max_output_tokens")
			}
		}
		if _, hasMaxCompletionTokens := reqBody["max_completion_tokens"]; hasMaxCompletionTokens {
			if account.Type == AccountTypeAPIKey || runtimePlatform != PlatformOpenAI {
				delete(reqBody, "max_completion_tokens")
				bodyModified = true
				markPatchDelete("max_completion_tokens")
			}
		}
		unsupportedFields := []string{"prompt_cache_retention", "safety_identifier"}
		for _, unsupportedField := range unsupportedFields {
			if _, has := reqBody[unsupportedField]; has {
				delete(reqBody, unsupportedField)
				bodyModified = true
				markPatchDelete(unsupportedField)
			}
		}
	}
	if wsDecision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
		if _, has := reqBody["previous_response_id"]; has {
			delete(reqBody, "previous_response_id")
			bodyModified = true
			markPatchDelete("previous_response_id")
		}
	}
	if bodyModified {
		serializedByPatch := false
		if !patchDisabled && patchHasOp {
			var patchErr error
			if patchDelete {
				body, patchErr = sjson.DeleteBytes(body, patchPath)
			} else {
				body, patchErr = sjson.SetBytes(body, patchPath, patchValue)
			}
			if patchErr == nil {
				serializedByPatch = true
			}
		}
		if !serializedByPatch {
			var marshalErr error
			body, marshalErr = json.Marshal(reqBody)
			if marshalErr != nil {
				return nil, fmt.Errorf("serialize request body: %w", marshalErr)
			}
		}
	}
	if sanitizeOpenAIEmptyThinkingBlocks(reqBody) {
		var marshalErr error
		body, marshalErr = json.Marshal(reqBody)
		if marshalErr != nil {
			return nil, fmt.Errorf("serialize empty thinking block normalized request body: %w", marshalErr)
		}
	}
	ctx = WithOpenAICodexRequestModel(ctx, mappedModel)
	if c != nil && c.Request != nil {
		c.Request = c.Request.WithContext(ctx)
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	setOpsUpstreamRequestBody(c, body)
	if wsDecision.Transport == OpenAIUpstreamTransportResponsesWebsocketV2 {
		wsReqBody := reqBody
		if len(reqBody) > 0 {
			wsReqBody = make(map[string]any, len(reqBody))
			for k, v := range reqBody {
				wsReqBody[k] = v
			}
		}
		_, hasPreviousResponseID := wsReqBody["previous_response_id"]
		logOpenAIWSModeDebug("forward_start account_id=%d account_type=%s model=%s stream=%v has_previous_response_id=%v", account.ID, account.Type, mappedModel, reqStream, hasPreviousResponseID)
		maxAttempts := openAIWSReconnectRetryLimit + 1
		wsAttempts := 0
		var wsResult *OpenAIForwardResult
		var wsErr error
		wsLastFailureReason := ""
		wsPrevResponseRecoveryTried := false
		wsInvalidEncryptedContentRecoveryTried := false
		wsEmptyThinkingBlockRecoveryTried := false
		recoverPrevResponseNotFound := func(attempt int) bool {
			if wsPrevResponseRecoveryTried {
				return false
			}
			previousResponseID := openAIWSPayloadString(wsReqBody, "previous_response_id")
			if previousResponseID == "" {
				logOpenAIWSModeInfo("reconnect_prev_response_recovery_skip account_id=%d attempt=%d reason=missing_previous_response_id previous_response_id_present=false", account.ID, attempt)
				return false
			}
			if HasFunctionCallOutput(wsReqBody) {
				logOpenAIWSModeInfo("reconnect_prev_response_recovery_skip account_id=%d attempt=%d reason=has_function_call_output previous_response_id_present=true", account.ID, attempt)
				return false
			}
			delete(wsReqBody, "previous_response_id")
			wsPrevResponseRecoveryTried = true
			logOpenAIWSModeInfo("reconnect_prev_response_recovery account_id=%d attempt=%d action=drop_previous_response_id retry=1 previous_response_id=%s previous_response_id_kind=%s", account.ID, attempt, truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(ClassifyOpenAIPreviousResponseIDKind(previousResponseID)))
			return true
		}
		recoverInvalidEncryptedContent := func(attempt int) bool {
			if wsInvalidEncryptedContentRecoveryTried {
				return false
			}
			removedReasoningItems := trimOpenAIEncryptedReasoningItems(wsReqBody)
			if !removedReasoningItems {
				logOpenAIWSModeInfo("reconnect_invalid_encrypted_content_recovery_skip account_id=%d attempt=%d reason=missing_encrypted_reasoning_items", account.ID, attempt)
				return false
			}
			previousResponseID := openAIWSPayloadString(wsReqBody, "previous_response_id")
			hasFunctionCallOutput := HasFunctionCallOutput(wsReqBody)
			droppedPreviousResponseID := false
			if previousResponseID != "" && !hasFunctionCallOutput {
				delete(wsReqBody, "previous_response_id")
				droppedPreviousResponseID = true
			}
			wsInvalidEncryptedContentRecoveryTried = true
			logOpenAIWSModeInfo("reconnect_invalid_encrypted_content_recovery account_id=%d attempt=%d action=drop_encrypted_reasoning_items retry=1 previous_response_id_present=%v previous_response_id=%s previous_response_id_kind=%s has_function_call_output=%v dropped_previous_response_id=%v", account.ID, attempt, previousResponseID != "", truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(ClassifyOpenAIPreviousResponseIDKind(previousResponseID)), hasFunctionCallOutput, droppedPreviousResponseID)
			return true
		}
		recoverEmptyThinkingBlock := func(attempt int) bool {
			if wsEmptyThinkingBlockRecoveryTried {
				return false
			}
			if !sanitizeOpenAIEmptyThinkingBlocks(wsReqBody) {
				logOpenAIWSModeInfo("reconnect_empty_thinking_recovery_skip account_id=%d attempt=%d reason=missing_empty_thinking_blocks", account.ID, attempt)
				return false
			}
			wsEmptyThinkingBlockRecoveryTried = true
			logOpenAIWSModeInfo("reconnect_empty_thinking_recovery account_id=%d attempt=%d action=drop_empty_thinking_blocks retry=1", account.ID, attempt)
			return true
		}
		retryBudget := s.openAIWSRetryTotalBudget()
		retryStartedAt := time.Now()
	wsRetryLoop:
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			wsAttempts = attempt
			wsResult, wsErr = s.forwardOpenAIWSV2(ctx, c, account, wsReqBody, token, wsDecision, isCodexCLI, reqStream, originalModel, mappedModel, startTime, attempt, wsLastFailureReason)
			if wsErr == nil {
				break
			}
			if c != nil && c.Writer != nil && c.Writer.Written() {
				break
			}
			reason, retryable := classifyOpenAIWSReconnectReason(wsErr)
			if reason != "" {
				wsLastFailureReason = reason
			}
			if reason == "previous_response_not_found" && recoverPrevResponseNotFound(attempt) {
				continue
			}
			if reason == "invalid_encrypted_content" && recoverInvalidEncryptedContent(attempt) {
				continue
			}
			if reason == "empty_thinking_block" && recoverEmptyThinkingBlock(attempt) {
				continue
			}
			if retryable && attempt < maxAttempts {
				backoff := s.openAIWSRetryBackoff(attempt)
				if retryBudget > 0 && time.Since(retryStartedAt)+backoff > retryBudget {
					s.recordOpenAIWSRetryExhausted()
					logOpenAIWSModeInfo("reconnect_budget_exhausted account_id=%d attempts=%d max_retries=%d reason=%s elapsed_ms=%d budget_ms=%d", account.ID, attempt, openAIWSReconnectRetryLimit, normalizeOpenAIWSLogValue(reason), time.Since(retryStartedAt).Milliseconds(), retryBudget.Milliseconds())
					break
				}
				s.recordOpenAIWSRetryAttempt(backoff)
				logOpenAIWSModeInfo("reconnect_retry account_id=%d retry=%d max_retries=%d reason=%s backoff_ms=%d", account.ID, attempt, openAIWSReconnectRetryLimit, normalizeOpenAIWSLogValue(reason), backoff.Milliseconds())
				if backoff > 0 {
					timer := time.NewTimer(backoff)
					select {
					case <-ctx.Done():
						if !timer.Stop() {
							<-timer.C
						}
						wsErr = wrapOpenAIWSFallback("retry_backoff_canceled", ctx.Err())
						break wsRetryLoop
					case <-timer.C:
					}
				}
				continue
			}
			if retryable {
				s.recordOpenAIWSRetryExhausted()
				logOpenAIWSModeInfo("reconnect_exhausted account_id=%d attempts=%d max_retries=%d reason=%s", account.ID, attempt, openAIWSReconnectRetryLimit, normalizeOpenAIWSLogValue(reason))
			} else if reason != "" {
				s.recordOpenAIWSNonRetryableFastFallback()
				logOpenAIWSModeInfo("reconnect_stop account_id=%d attempt=%d reason=%s", account.ID, attempt, normalizeOpenAIWSLogValue(reason))
			}
			break
		}
		if wsErr == nil {
			if wsResult != nil {
				wsResult.UpstreamModel = mappedModel
				wsResult.SimulatedClient = simulatedClient
				applyClaudeCapabilityToOpenAIForwardResult(wsResult, claudeCapability)
			}
			firstTokenMs := int64(0)
			hasFirstTokenMs := wsResult != nil && wsResult.FirstTokenMs != nil
			if hasFirstTokenMs {
				firstTokenMs = int64(*wsResult.FirstTokenMs)
			}
			requestID := ""
			if wsResult != nil {
				requestID = strings.TrimSpace(wsResult.RequestID)
			}
			logOpenAIWSModeDebug("forward_succeeded account_id=%d request_id=%s stream=%v has_first_token_ms=%v first_token_ms=%d ws_attempts=%d", account.ID, requestID, reqStream, hasFirstTokenMs, firstTokenMs, wsAttempts)
			return wsResult, nil
		}
		s.writeOpenAIWSFallbackErrorResponse(c, account, wsErr)
		return nil, wsErr
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	httpInvalidEncryptedContentRetryTried := false
	httpEmptyThinkingBlockRetryTried := false
	for {
		upstreamReq, err := s.buildUpstreamRequest(ctx, c, account, body, token, reqStream, promptCacheKey, isCodexCLI)
		if err != nil {
			return nil, err
		}
		upstreamStart := time.Now()
		resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
		SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
		if err != nil {
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			setOpsUpstreamError(c, 0, safeErr, "")
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "request_error", Message: safeErr})
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "upstream_error", "message": "Upstream request failed"}})
			return nil, fmt.Errorf("upstream request failed: %s", safeErr)
		}
		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
			upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
			upstreamCode := extractUpstreamErrorCode(respBody)
			if !httpEmptyThinkingBlockRetryTried && resp.StatusCode == http.StatusBadRequest && isOpenAIEmptyThinkingBlockError(resp.StatusCode, upstreamMsg, respBody) {
				if sanitizeOpenAIEmptyThinkingBlocks(reqBody) {
					body, err = json.Marshal(reqBody)
					if err != nil {
						_ = resp.Body.Close()
						return nil, fmt.Errorf("serialize empty thinking block retry body: %w", err)
					}
					setOpsUpstreamRequestBody(c, body)
					httpEmptyThinkingBlockRetryTried = true
					logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Retrying non-WSv2 request once after empty thinking block error (account: %s)", account.Name)
					_ = resp.Body.Close()
					continue
				}
				logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Skip non-WSv2 empty thinking block retry because empty thinking blocks are missing (account: %s)", account.Name)
			}
			if !httpInvalidEncryptedContentRetryTried && resp.StatusCode == http.StatusBadRequest && upstreamCode == "invalid_encrypted_content" {
				if trimOpenAIEncryptedReasoningItems(reqBody) {
					body, err = json.Marshal(reqBody)
					if err != nil {
						_ = resp.Body.Close()
						return nil, fmt.Errorf("serialize invalid_encrypted_content retry body: %w", err)
					}
					setOpsUpstreamRequestBody(c, body)
					httpInvalidEncryptedContentRetryTried = true
					logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Retrying non-WSv2 request once after invalid_encrypted_content (account: %s)", account.Name)
					_ = resp.Body.Close()
					continue
				}
				logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Skip non-WSv2 invalid_encrypted_content retry because encrypted reasoning items are missing (account: %s)", account.Name)
			}
			if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(respBody), maxBytes)
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
				s.handleFailoverSideEffects(ctx, resp, account)
				_ = resp.Body.Close()
				return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && (isPoolModeRetryableStatus(resp.StatusCode) || isOpenAITransientProcessingError(resp.StatusCode, upstreamMsg, respBody))}
			}
			nextResult, nextErr := s.handleErrorResponse(ctx, resp, c, account, body)
			_ = resp.Body.Close()
			return nextResult, nextErr
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		// Success response: process and return (no more retries).
		var usage *OpenAIUsage
		var firstTokenMs *int
		if reqStream {
			streamResult, err := s.handleStreamingResponse(ctx, resp, c, account, startTime, originalModel, mappedModel)
			if err != nil {
				return nil, err
			}
			usage = streamResult.usage
			firstTokenMs = streamResult.firstTokenMs
		} else {
			usage, err = s.handleNonStreamingResponse(ctx, resp, c, account, originalModel, mappedModel)
			if err != nil {
				return nil, err
			}
		}
		if isChatGPTOpenAIOAuthAccount(account) {
			if snapshot := ParseCodexRateLimitHeaders(resp.Header); snapshot != nil {
				s.updateCodexUsageSnapshot(ctx, account.ID, snapshot)
			}
		}
		if usage == nil {
			usage = &OpenAIUsage{}
		}
		serviceTier := extractOpenAIServiceTier(reqBody)
		result := &OpenAIForwardResult{
			RequestID:                resp.Header.Get("x-request-id"),
			Usage:                    *usage,
			Model:                    originalModel,
			UpstreamModel:            mappedModel,
			SimulatedClient:          simulatedClient,
			ServiceTier:              serviceTier,
			ReasoningEffort:          effortResolution.Effective,
			ReasoningEffortRaw:       effortResolution.Raw,
			ReasoningEffortEffective: effortResolution.Effective,
			ReasoningEffortSource:    effortResolution.Source,
			Stream:                   reqStream,
			OpenAIWSMode:             false,
			Duration:                 time.Since(startTime),
			FirstTokenMs:             firstTokenMs,
		}
		applyClaudeCapabilityToOpenAIForwardResult(result, claudeCapability)
		return result, nil
	}
}
