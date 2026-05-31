package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func applyClaudeCapabilityToForwardResult(result *ForwardResult, capability ClaudeRequestCapability) {
	if result == nil {
		return
	}
	result.RequestedModelRaw = strings.TrimSpace(capability.RequestedModelRaw)
	result.RequestedModelNormalized = strings.TrimSpace(capability.RequestedModelNormalized)
	result.MillionContextRequested = capability.MillionContextRequested
	result.MillionContextEffective = capability.MillionContextEffective
	result.MillionContextSource = strings.TrimSpace(capability.MillionContextSource)
	if capability.MillionContextEffective {
		result.MillionContextBetaToken = strings.TrimSpace(capability.MillionContextBetaToken)
		return
	}
	result.MillionContextBetaToken = ""
}

func (s *GeminiCompatGatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, body []byte) (*ForwardResult, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil
	}
	account = ResolveProtocolGatewayInboundAccount(account, PlatformGemini)
	startTime := time.Now()
	var req struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("missing model")
	}
	originalModel := req.Model
	claudeCapability := RecordClaudeCapabilityMetadataRequestedOnly(ctx, originalModel, strings.TrimSpace(gjson.GetBytes(body, "effortLevel").String()))
	normalizedModel := firstNonEmptyString(claudeCapability.RequestedModelNormalized, originalModel)
	req.Model = normalizedModel
	mappedModel := normalizedModel
	if account.Type == AccountTypeAPIKey {
		mappedModel = account.GetMappedModel(normalizedModel)
	}
	simulatedClient := ""
	if route := MatchGatewayClientRoute(account, PlatformGemini, mappedModel); route != nil {
		simulatedClient = route.ClientProfile
	}
	shouldMimicGeminiCLI := simulatedClient == GatewayClientProfileGeminiCLI
	geminiReq, _, err := ConvertAnthropicMessagesToGeminiGenerateContentRuntime(body, geminiTransformOptions{
		AllowURLContext: account == nil || !account.IsGeminiVertexSource(),
	})
	if err != nil {
		if message, ok := response.LocalizedCompatErrorMessage(c, err); ok {
			return nil, s.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", message)
		}
		return nil, s.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
	}
	geminiReq = ensureGeminiFunctionCallThoughtSignatures(geminiReq)
	originalClaudeBody := body
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	var requestIDHeader string
	var buildReq func(ctx context.Context) (*http.Request, string, error)
	useUpstreamStream := req.Stream
	if account.Type == AccountTypeOAuth && !req.Stream && !account.IsGeminiVertexAI() && strings.TrimSpace(account.GetCredential("project_id")) != "" {
		useUpstreamStream = true
	}
	switch account.Type {
	case AccountTypeAPIKey:
		buildReq = func(ctx context.Context) (*http.Request, string, error) {
			action := "generateContent"
			if req.Stream {
				action = "streamGenerateContent"
			}
			return s.buildGeminiAPIKeyUpstreamRequest(ctx, account, mappedModel, action, geminiReq, shouldMimicGeminiCLI)
		}
		requestIDHeader = "x-request-id"
	case AccountTypeOAuth:
		buildReq = func(ctx context.Context) (*http.Request, string, error) {
			projectID := strings.TrimSpace(account.GetCredential("project_id"))
			action := "generateContent"
			if useUpstreamStream {
				action = "streamGenerateContent"
			}
			return s.buildGeminiOAuthCompatUpstreamRequest(ctx, account, mappedModel, action, useUpstreamStream, geminiReq, projectID, shouldMimicGeminiCLI)
		}
		requestIDHeader = "x-request-id"
	default:
		return nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}
	var resp *http.Response
	signatureRetryStage := 0
	for attempt := 1; attempt <= geminiMaxRetries; attempt++ {
		upstreamReq, idHeader, err := buildReq(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			if isGeminiCredentialConfigError(err) {
				return nil, s.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
			}
			return nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", err.Error())
		}
		requestIDHeader = idHeader
		setOpsUpstreamRequestBody(c, geminiReq)
		resp, err = s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
		if err != nil {
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "request_error", Message: safeErr})
			if attempt < geminiMaxRetries {
				logger.LegacyPrintf("service.gemini_messages_compat", "Gemini account %d: upstream request failed, retry %d/%d: %v", account.ID, attempt, geminiMaxRetries, err)
				sleepGeminiBackoff(attempt)
				continue
			}
			setOpsUpstreamError(c, 0, safeErr, "")
			return nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed after retries: "+safeErr)
		}
		if resp.StatusCode == http.StatusBadRequest && signatureRetryStage < 2 {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			if isGeminiSignatureRelatedError(respBody) {
				upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
				upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
				upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(respBody), maxBytes)
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: upstreamReqID, Kind: "signature_error", Message: upstreamMsg, Detail: upstreamDetail})
				var strippedClaudeBody []byte
				stageName := ""
				switch signatureRetryStage {
				case 0:
					strippedClaudeBody = FilterThinkingBlocksForRetry(originalClaudeBody)
					stageName = "thinking-only"
					signatureRetryStage = 1
				default:
					strippedClaudeBody = FilterSignatureSensitiveBlocksForRetry(originalClaudeBody)
					stageName = "thinking+tools"
					signatureRetryStage = 2
				}
				retryGeminiReq, _, txErr := ConvertAnthropicMessagesToGeminiGenerateContentRuntime(strippedClaudeBody, geminiTransformOptions{
					AllowURLContext: account == nil || !account.IsGeminiVertexSource(),
				})
				if txErr == nil {
					logger.LegacyPrintf("service.gemini_messages_compat", "Gemini account %d: detected signature-related 400, retrying with downgraded Claude blocks (%s)", account.ID, stageName)
					geminiReq = retryGeminiReq
					sleepGeminiBackoff(1)
					continue
				}
			}
			resp = &http.Response{StatusCode: http.StatusBadRequest, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}
			break
		}
		if matched, rebuilt := s.checkErrorPolicyInLoop(ctx, account, resp); matched {
			resp = rebuilt
			break
		} else {
			resp = rebuilt
		}
		if resp.StatusCode >= 400 && s.shouldRetryGeminiUpstreamError(account, resp.StatusCode) {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			if resp.StatusCode == 403 && isGeminiInsufficientScope(resp.Header, respBody) {
				resp = &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}
				break
			}
			if resp.StatusCode == 429 {
				s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			if attempt < geminiMaxRetries {
				upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
				upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
				upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(respBody), maxBytes)
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: upstreamReqID, Kind: "retry", Message: upstreamMsg, Detail: upstreamDetail})
				logger.LegacyPrintf("service.gemini_messages_compat", "Gemini account %d: upstream status %d, retry %d/%d", account.ID, resp.StatusCode, attempt, geminiMaxRetries)
				sleepGeminiBackoff(attempt)
				continue
			}
			resp = &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}
			break
		}
		break
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		SetOpsTraceUpstreamResponse(c, "gemini_upstream_error_response", respBody, resp.Header.Get("Content-Type"), false)
		if s.rateLimitService != nil {
			switch s.rateLimitService.CheckErrorPolicy(ctx, account, resp.StatusCode, respBody) {
			case ErrorPolicySkipped:
				upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
				return nil, s.writeGeminiMappedError(c, account, http.StatusInternalServerError, upstreamReqID, respBody)
			case ErrorPolicyMatched, ErrorPolicyTempUnscheduled:
				s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
				upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
				upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
				upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(respBody), maxBytes)
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: upstreamReqID, Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
				return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody}
			}
		}
		s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		RecordPublicModelCatalogRuntimeFailureIfModelCapabilityError(
			ctx,
			s.modelCatalogService,
			resp.StatusCode,
			sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody))),
			PlatformGemini,
			geminiEndpointKeyForAction(ProtocolCapabilityActionGenerateContent),
			"text",
		)
		if resp.StatusCode == http.StatusBadRequest {
			msg400 := strings.ToLower(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
			if isGoogleProjectConfigError(msg400) {
				upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
				upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(respBody), maxBytes)
				}
				log.Printf("[Gemini] status=400 google_config_error failover=true upstream_message=%q account=%d", upstreamMsg, account.ID)
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: upstreamReqID, Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
				return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: true}
			}
		}
		if s.shouldFailoverGeminiUpstreamError(resp.StatusCode) {
			upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
			upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
			upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
			upstreamDetail := ""
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
				if maxBytes <= 0 {
					maxBytes = 2048
				}
				upstreamDetail = truncateString(string(respBody), maxBytes)
			}
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: upstreamReqID, Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody}
		}
		upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
		return nil, s.writeGeminiMappedError(c, account, resp.StatusCode, upstreamReqID, respBody)
	}
	requestID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
	if requestID != "" {
		c.Header("x-request-id", requestID)
	}
	var usage *ClaudeUsage
	var firstTokenMs *int
	requestedServiceTier := extractGeminiRequestedServiceTierFromBody(body)
	var resolvedServiceTier *string
	if req.Stream {
		streamRes, err := s.handleStreamingResponse(c, resp, startTime, originalModel)
		if err != nil {
			return nil, err
		}
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
		resolvedServiceTier = streamRes.resolvedServiceTier
		if requestID == "" && strings.TrimSpace(streamRes.responseID) != "" {
			requestID = strings.TrimSpace(streamRes.responseID)
			c.Header("x-request-id", requestID)
		}
	} else {
		if useUpstreamStream {
			collected, usageObj, err := collectGeminiSSE(resp.Body, true)
			if err != nil {
				return nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", "Failed to read upstream stream")
			}
			collectedBytes, _ := json.Marshal(collected)
			SetOpsTraceUpstreamResponse(c, "gemini_upstream_response", collectedBytes, "application/json", false)
			if candidate := extractGeminiResolvedServiceTierFromResponse(collectedBytes, resp.Header); candidate != nil {
				resolvedServiceTier = candidate
			}
			claudeResp, usageObj2, convErr := convertGeminiToClaudeMessage(collected, originalModel, collectedBytes)
			if convErr != nil {
				var compatErr *geminiCompatResponseError
				if errors.As(convErr, &compatErr) {
					if requestID == "" && strings.TrimSpace(compatErr.responseID) != "" {
						requestID = strings.TrimSpace(compatErr.responseID)
						c.Header("x-request-id", requestID)
					}
					return nil, s.writeClaudeError(c, compatErr.statusCode, compatErr.errorType, compatErr.message)
				}
				return nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", convErr.Error())
			}
			if requestID == "" {
				if responseID := strings.TrimSpace(stringValueFromAny(claudeResp["google_response_id"])); responseID != "" {
					requestID = responseID
					c.Header("x-request-id", requestID)
				}
			}
			c.JSON(http.StatusOK, claudeResp)
			usage = usageObj2
			if usageObj != nil && (usageObj.InputTokens > 0 || usageObj.OutputTokens > 0) {
				usage = usageObj
			}
		} else {
			responseID := ""
			usage, responseID, resolvedServiceTier, err = s.handleNonStreamingResponse(c, resp, originalModel)
			if err != nil {
				return nil, err
			}
			if requestID == "" && strings.TrimSpace(responseID) != "" {
				requestID = strings.TrimSpace(responseID)
				c.Header("x-request-id", requestID)
			}
		}
	}
	imageCount := 0
	imageSize := s.extractImageSize(body)
	if isImageGenerationModel(originalModel) {
		imageCount = 1
	}
	if resolvedServiceTier == nil {
		resolvedServiceTier = extractGeminiResolvedServiceTierFromResponse(nil, resp.Header)
	}
	if resolvedServiceTier == nil {
		resolvedServiceTier = requestedServiceTier
	}
	result := &ForwardResult{RequestID: requestID, Usage: *usage, Model: originalModel, UpstreamModel: mappedModel, RequestedServiceTier: requestedServiceTier, ServiceTier: resolvedServiceTier, SimulatedClient: simulatedClient, Stream: req.Stream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ImageCount: imageCount, ImageSize: imageSize}
	applyClaudeCapabilityToForwardResult(result, claudeCapability)
	return result, nil
}

func (s *GeminiNativeGatewayService) ForwardNative(ctx context.Context, c *gin.Context, account *Account, originalModel string, action string, stream bool, body []byte) (*ForwardResult, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil
	}
	account = ResolveProtocolGatewayInboundAccount(account, PlatformGemini)
	startTime := time.Now()
	if strings.TrimSpace(originalModel) == "" {
		return nil, s.writeGoogleError(c, http.StatusBadRequest, "Missing model in URL")
	}
	if strings.TrimSpace(action) == "" {
		return nil, s.writeGoogleError(c, http.StatusBadRequest, "Missing action in URL")
	}
	if len(body) == 0 {
		return nil, s.writeGoogleError(c, http.StatusBadRequest, "Request body is empty")
	}
	if filteredBody, err := filterEmptyPartsFromGeminiRequest(body); err == nil {
		body = filteredBody
	}
	switch action {
	case "generateContent", "generateAnswer", "streamGenerateContent", "countTokens":
	default:
		return nil, s.writeGoogleError(c, http.StatusNotFound, "Unsupported action: "+action)
	}
	body = ensureGeminiFunctionCallThoughtSignatures(body)
	runtimeRequestedModel := originalModel
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		if sourceModel := strings.TrimSpace(entry.SourceModelID); sourceModel != "" {
			runtimeRequestedModel = sourceModel
		}
	}
	claudeCapability := RecordClaudeCapabilityMetadataRequestedOnly(ctx, runtimeRequestedModel, strings.TrimSpace(gjson.GetBytes(body, "effortLevel").String()))
	normalizedModel := firstNonEmptyString(claudeCapability.RequestedModelNormalized, runtimeRequestedModel)
	mappedModel := normalizedModel
	if account.Type == AccountTypeAPIKey {
		mappedModel = account.GetMappedModel(normalizedModel)
	}
	simulatedClient := ""
	if route := MatchGatewayClientRoute(account, PlatformGemini, mappedModel); route != nil {
		simulatedClient = route.ClientProfile
	}
	shouldMimicGeminiCLI := simulatedClient == GatewayClientProfileGeminiCLI
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	useUpstreamStream := stream
	upstreamAction := action
	if account.Type == AccountTypeOAuth && !stream && action == "generateContent" && !account.IsGeminiVertexAI() && strings.TrimSpace(account.GetCredential("project_id")) != "" {
		useUpstreamStream = true
		upstreamAction = "streamGenerateContent"
	}
	forceAIStudio := action == "countTokens"
	var requestIDHeader string
	var buildReq func(ctx context.Context) (*http.Request, string, error)
	switch account.Type {
	case AccountTypeAPIKey:
		buildReq = func(ctx context.Context) (*http.Request, string, error) {
			return s.buildGeminiAPIKeyUpstreamRequest(ctx, account, mappedModel, upstreamAction, body, shouldMimicGeminiCLI)
		}
		requestIDHeader = "x-request-id"
	case AccountTypeOAuth:
		buildReq = func(ctx context.Context) (*http.Request, string, error) {
			projectID := strings.TrimSpace(account.GetCredential("project_id"))
			return s.buildGeminiOAuthNativeUpstreamRequest(ctx, account, mappedModel, upstreamAction, useUpstreamStream, body, projectID, forceAIStudio, shouldMimicGeminiCLI)
		}
		requestIDHeader = "x-request-id"
	default:
		return nil, s.writeGoogleError(c, http.StatusBadGateway, "Unsupported account type: "+account.Type)
	}
	var resp *http.Response
	for attempt := 1; attempt <= geminiMaxRetries; attempt++ {
		upstreamReq, idHeader, err := buildReq(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			if isGeminiCredentialConfigError(err) {
				return nil, s.writeGoogleError(c, http.StatusBadRequest, err.Error())
			}
			return nil, s.writeGoogleError(c, http.StatusBadGateway, err.Error())
		}
		requestIDHeader = idHeader
		setOpsUpstreamRequestBody(c, body)
		resp, err = s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
		if err != nil {
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "request_error", Message: safeErr})
			if attempt < geminiMaxRetries {
				logger.LegacyPrintf("service.gemini_messages_compat", "Gemini account %d: upstream request failed, retry %d/%d: %v", account.ID, attempt, geminiMaxRetries, err)
				sleepGeminiBackoff(attempt)
				continue
			}
			if action == "countTokens" {
				return s.finishGeminiEstimatedCountTokensResponse(c, account, originalModel, mappedModel, simulatedClient, "", body, 0, safeErr, "", startTime)
			}
			setOpsUpstreamError(c, 0, safeErr, "")
			return nil, s.writeGoogleError(c, http.StatusBadGateway, "Upstream request failed after retries: "+safeErr)
		}
		if matched, rebuilt := s.checkErrorPolicyInLoop(ctx, account, resp); matched {
			resp = rebuilt
			break
		} else {
			resp = rebuilt
		}
		if resp.StatusCode >= 400 && s.shouldRetryGeminiUpstreamError(account, resp.StatusCode) {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			if resp.StatusCode == 403 && isGeminiInsufficientScope(resp.Header, respBody) {
				resp = &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}
				break
			}
			if resp.StatusCode == 429 {
				s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			if attempt < geminiMaxRetries {
				upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
				upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
				upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(respBody), maxBytes)
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: upstreamReqID, Kind: "retry", Message: upstreamMsg, Detail: upstreamDetail})
				logger.LegacyPrintf("service.gemini_messages_compat", "Gemini account %d: upstream status %d, retry %d/%d", account.ID, resp.StatusCode, attempt, geminiMaxRetries)
				sleepGeminiBackoff(attempt)
				continue
			}
			if action == "countTokens" {
				upstreamReqID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
				upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
				upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(respBody), maxBytes)
				}
				return s.finishGeminiEstimatedCountTokensResponse(c, account, originalModel, mappedModel, simulatedClient, upstreamReqID, body, resp.StatusCode, upstreamMsg, upstreamDetail, startTime)
			}
			resp = &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}
			break
		}
		break
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	requestID := getGeminiUpstreamRequestID(resp.Header, requestIDHeader)
	if requestID != "" {
		c.Header("x-request-id", requestID)
	}
	isOAuth := account.Type == AccountTypeOAuth
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		SetOpsTraceUpstreamResponse(c, "gemini_native_upstream_error_response", unwrapIfNeeded(isOAuth, respBody), resp.Header.Get("Content-Type"), false)
		if action == "countTokens" {
			s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			evBody := unwrapIfNeeded(isOAuth, respBody)
			upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(evBody))
			upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
			RecordPublicModelCatalogRuntimeFailureIfModelCapabilityError(
				ctx,
				s.modelCatalogService,
				resp.StatusCode,
				upstreamMsg,
				PlatformGemini,
				geminiEndpointKeyForAction(action),
				geminiCapabilityForAction(action),
			)
			upstreamDetail := ""
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
				if maxBytes <= 0 {
					maxBytes = 2048
				}
				upstreamDetail = truncateString(string(evBody), maxBytes)
			}
			return s.finishGeminiEstimatedCountTokensResponse(c, account, originalModel, mappedModel, simulatedClient, requestID, body, resp.StatusCode, upstreamMsg, upstreamDetail, startTime)
		}
		if s.rateLimitService != nil {
			switch s.rateLimitService.CheckErrorPolicy(ctx, account, resp.StatusCode, respBody) {
			case ErrorPolicySkipped:
				respBody = unwrapIfNeeded(isOAuth, respBody)
				contentType := resp.Header.Get("Content-Type")
				if contentType == "" {
					contentType = "application/json"
				}
				c.Data(http.StatusInternalServerError, contentType, respBody)
				return nil, fmt.Errorf("gemini upstream error: %d (skipped by error policy)", resp.StatusCode)
			case ErrorPolicyMatched, ErrorPolicyTempUnscheduled:
				s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
				evBody := unwrapIfNeeded(isOAuth, respBody)
				upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(evBody))
				upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(evBody), maxBytes)
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: requestID, Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
				return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody}
			}
		}
		s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		RecordPublicModelCatalogRuntimeFailureIfModelCapabilityError(
			ctx,
			s.modelCatalogService,
			resp.StatusCode,
			sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(unwrapIfNeeded(isOAuth, respBody)))),
			PlatformGemini,
			geminiEndpointKeyForAction(action),
			geminiCapabilityForAction(action),
		)
		if resp.StatusCode == http.StatusBadRequest {
			msg400 := strings.ToLower(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
			if isGoogleProjectConfigError(msg400) {
				evBody := unwrapIfNeeded(isOAuth, respBody)
				upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(evBody)))
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(evBody), maxBytes)
				}
				log.Printf("[Gemini] status=400 google_config_error failover=true upstream_message=%q account=%d", upstreamMsg, account.ID)
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: requestID, Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
				return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: evBody, RetryableOnSameAccount: true}
			}
		}
		if s.shouldFailoverGeminiUpstreamError(resp.StatusCode) {
			evBody := unwrapIfNeeded(isOAuth, respBody)
			upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(evBody))
			upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
			upstreamDetail := ""
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
				if maxBytes <= 0 {
					maxBytes = 2048
				}
				upstreamDetail = truncateString(string(evBody), maxBytes)
			}
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: requestID, Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: evBody}
		}
		respBody = unwrapIfNeeded(isOAuth, respBody)
		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		upstreamDetail := ""
		if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
			maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
			if maxBytes <= 0 {
				maxBytes = 2048
			}
			upstreamDetail = truncateString(string(respBody), maxBytes)
			logger.LegacyPrintf("service.gemini_messages_compat", "[Gemini] native upstream error %d: %s", resp.StatusCode, truncateForLog(respBody, s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes))
		}
		setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: requestID, Kind: "http_error", Message: upstreamMsg, Detail: upstreamDetail})
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(resp.StatusCode, contentType, respBody)
		if upstreamMsg == "" {
			return nil, fmt.Errorf("gemini upstream error: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("gemini upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
	}
	if action == "countTokens" {
		setGeminiCountTokensSourceHeader(c, geminiCountTokensSourceUpstream)
	}
	var usage *ClaudeUsage
	var firstTokenMs *int
	requestedServiceTier := extractGeminiRequestedServiceTierFromBody(body)
	var resolvedServiceTier *string
	if stream {
		streamRes, err := s.handleNativeStreamingResponse(c, resp, startTime, isOAuth)
		if err != nil {
			return nil, err
		}
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
		resolvedServiceTier = streamRes.resolvedServiceTier
		if requestID == "" && strings.TrimSpace(streamRes.responseID) != "" {
			requestID = strings.TrimSpace(streamRes.responseID)
		}
		if requestID == "" && c != nil && c.Writer != nil {
			requestID = strings.TrimSpace(c.Writer.Header().Get("x-request-id"))
		}
	} else {
		if useUpstreamStream {
			collected, usageObj, err := collectGeminiSSE(resp.Body, isOAuth)
			if err != nil {
				return nil, s.writeGoogleError(c, http.StatusBadGateway, "Failed to read upstream stream")
			}
			collectedBytes, _ := json.Marshal(collected)
			SetOpsTraceUpstreamResponse(c, "gemini_native_upstream_response", collectedBytes, "application/json", false)
			if candidate := extractGeminiResolvedServiceTierFromResponse(collectedBytes, resp.Header); candidate != nil {
				resolvedServiceTier = candidate
			}
			if requestID == "" {
				if responseID := strings.TrimSpace(stringValueFromAny(collected["responseId"])); responseID != "" {
					requestID = responseID
					c.Header("x-request-id", requestID)
				}
			}
			c.Data(http.StatusOK, "application/json", collectedBytes)
			usage = usageObj
		} else {
			usageResp, resolvedTier, err := s.handleNativeNonStreamingResponse(c, resp, isOAuth)
			if err != nil {
				return nil, err
			}
			usage = usageResp
			resolvedServiceTier = resolvedTier
			if requestID == "" && c != nil && c.Writer != nil {
				requestID = strings.TrimSpace(c.Writer.Header().Get("x-request-id"))
			}
		}
	}
	if usage == nil {
		usage = &ClaudeUsage{}
	}
	imageCount := 0
	imageSize := s.extractImageSize(body)
	if isImageGenerationModel(originalModel) {
		imageCount = 1
	}
	if resolvedServiceTier == nil {
		resolvedServiceTier = extractGeminiResolvedServiceTierFromResponse(nil, resp.Header)
	}
	if resolvedServiceTier == nil {
		resolvedServiceTier = requestedServiceTier
	}
	result := &ForwardResult{RequestID: requestID, Usage: *usage, Model: originalModel, UpstreamModel: mappedModel, RequestedServiceTier: requestedServiceTier, ServiceTier: resolvedServiceTier, SimulatedClient: simulatedClient, Stream: stream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ImageCount: imageCount, ImageSize: imageSize}
	applyClaudeCapabilityToForwardResult(result, claudeCapability)
	return result, nil
}

func geminiEndpointKeyForAction(action string) string {
	switch action {
	case ProtocolCapabilityActionGeminiCountTokens:
		return "gemini.countTokens"
	case ProtocolCapabilityActionGeminiEmbedContent:
		return "gemini.embedContent"
	default:
		return "gemini.generateContent"
	}
}

func geminiCapabilityForAction(action string) string {
	switch action {
	case ProtocolCapabilityActionGeminiCountTokens:
		return "count_tokens"
	case ProtocolCapabilityActionGeminiEmbedContent:
		return "embeddings"
	default:
		return "text"
	}
}
