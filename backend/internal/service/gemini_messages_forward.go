package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"strings"
	"time"
)

func (s *GeminiMessagesCompatService) Forward(ctx context.Context, c *gin.Context, account *Account, body []byte) (*ForwardResult, error) {
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
	mappedModel := req.Model
	if account.Type == AccountTypeAPIKey {
		mappedModel = account.GetMappedModel(req.Model)
	}
	simulatedClient := ""
	if route := MatchGatewayClientRoute(account, PlatformGemini, mappedModel); route != nil {
		simulatedClient = route.ClientProfile
	}
	shouldMimicGeminiCLI := simulatedClient == GatewayClientProfileGeminiCLI
	geminiReq, err := convertClaudeMessagesToGeminiGenerateContent(body, geminiTransformOptions{
		AllowURLContext: account == nil || !account.IsGeminiVertexSource(),
	})
	if err != nil {
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
			if s.tokenProvider == nil {
				return nil, "", errors.New("gemini token provider not configured")
			}
			accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return nil, "", err
			}
			projectID := strings.TrimSpace(account.GetCredential("project_id"))
			action := "generateContent"
			if useUpstreamStream {
				action = "streamGenerateContent"
			}
			if account.IsGeminiVertexAI() {
				mappedModel = normalizeVertexUpstreamModelID(mappedModel)
				baseURL := account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
				normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
				if err != nil {
					return nil, "", err
				}
				actionPath, err := account.GeminiVertexModelActionPath(mappedModel, action)
				if err != nil {
					return nil, "", err
				}
				fullURL := strings.TrimRight(normalizedBaseURL, "/") + actionPath
				if useUpstreamStream {
					fullURL += "?alt=sse"
				}
				upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(geminiReq))
				if err != nil {
					return nil, "", err
				}
				upstreamReq.Header.Set("Content-Type", "application/json")
				upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
				if shouldMimicGeminiCLI {
					upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
				}
				return upstreamReq, "x-request-id", nil
			}
			if projectID != "" {
				baseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
				if err != nil {
					return nil, "", err
				}
				fullURL := fmt.Sprintf("%s/v1internal:%s", strings.TrimRight(baseURL, "/"), action)
				if useUpstreamStream {
					fullURL += "?alt=sse"
				}
				wrapped := map[string]any{"model": mappedModel, "project": projectID}
				var inner any
				if err := json.Unmarshal(geminiReq, &inner); err != nil {
					return nil, "", fmt.Errorf("failed to parse gemini request: %w", err)
				}
				wrapped["request"] = inner
				wrappedBytes, _ := json.Marshal(wrapped)
				upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(wrappedBytes))
				if err != nil {
					return nil, "", err
				}
				upstreamReq.Header.Set("Content-Type", "application/json")
				upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
				if shouldMimicGeminiCLI {
					upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
				}
				return upstreamReq, "x-request-id", nil
			} else {
				baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
				normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
				if err != nil {
					return nil, "", err
				}
				fullURL := fmt.Sprintf("%s/v1beta/models/%s:%s", strings.TrimRight(normalizedBaseURL, "/"), mappedModel, action)
				if useUpstreamStream {
					fullURL += "?alt=sse"
				}
				upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(geminiReq))
				if err != nil {
					return nil, "", err
				}
				upstreamReq.Header.Set("Content-Type", "application/json")
				upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
				if shouldMimicGeminiCLI {
					upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
				}
				return upstreamReq, "x-request-id", nil
			}
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
				retryGeminiReq, txErr := convertClaudeMessagesToGeminiGenerateContent(strippedClaudeBody, geminiTransformOptions{
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
	if req.Stream {
		streamRes, err := s.handleStreamingResponse(c, resp, startTime, originalModel)
		if err != nil {
			return nil, err
		}
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
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
			usage, responseID, err = s.handleNonStreamingResponse(c, resp, originalModel)
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
	return &ForwardResult{RequestID: requestID, Usage: *usage, Model: originalModel, UpstreamModel: mappedModel, SimulatedClient: simulatedClient, Stream: req.Stream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ImageCount: imageCount, ImageSize: imageSize}, nil
}
func isGeminiSignatureRelatedError(respBody []byte) bool {
	msg := strings.ToLower(strings.TrimSpace(extractAntigravityErrorMessage(respBody)))
	if msg == "" {
		msg = strings.ToLower(string(respBody))
	}
	return strings.Contains(msg, "thought_signature") || strings.Contains(msg, "signature")
}
func (s *GeminiMessagesCompatService) ForwardNative(ctx context.Context, c *gin.Context, account *Account, originalModel string, action string, stream bool, body []byte) (*ForwardResult, error) {
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
	case "generateContent", "streamGenerateContent", "countTokens":
	default:
		return nil, s.writeGoogleError(c, http.StatusNotFound, "Unsupported action: "+action)
	}
	body = ensureGeminiFunctionCallThoughtSignatures(body)
	mappedModel := originalModel
	if account.Type == AccountTypeAPIKey {
		mappedModel = account.GetMappedModel(originalModel)
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
			if s.tokenProvider == nil {
				return nil, "", errors.New("gemini token provider not configured")
			}
			accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return nil, "", err
			}
			if account.IsGeminiVertexAI() {
				baseURL := account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
				normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
				if err != nil {
					return nil, "", err
				}
				actionPath, err := account.GeminiVertexModelActionPath(mappedModel, upstreamAction)
				if err != nil {
					return nil, "", err
				}
				fullURL := strings.TrimRight(normalizedBaseURL, "/") + actionPath
				if useUpstreamStream {
					fullURL += "?alt=sse"
				}
				upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(body))
				if err != nil {
					return nil, "", err
				}
				upstreamReq.Header.Set("Content-Type", "application/json")
				upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
				if shouldMimicGeminiCLI {
					upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
				}
				return upstreamReq, "x-request-id", nil
			}
			projectID := strings.TrimSpace(account.GetCredential("project_id"))
			if projectID != "" && !forceAIStudio {
				baseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
				if err != nil {
					return nil, "", err
				}
				fullURL := fmt.Sprintf("%s/v1internal:%s", strings.TrimRight(baseURL, "/"), upstreamAction)
				if useUpstreamStream {
					fullURL += "?alt=sse"
				}
				wrapped := map[string]any{"model": mappedModel, "project": projectID}
				var inner any
				if err := json.Unmarshal(body, &inner); err != nil {
					return nil, "", fmt.Errorf("failed to parse gemini request: %w", err)
				}
				wrapped["request"] = inner
				wrappedBytes, _ := json.Marshal(wrapped)
				upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(wrappedBytes))
				if err != nil {
					return nil, "", err
				}
				upstreamReq.Header.Set("Content-Type", "application/json")
				upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
				if shouldMimicGeminiCLI {
					upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
				}
				return upstreamReq, "x-request-id", nil
			} else {
				baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
				normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
				if err != nil {
					return nil, "", err
				}
				fullURL := fmt.Sprintf("%s/v1beta/models/%s:%s", strings.TrimRight(normalizedBaseURL, "/"), mappedModel, upstreamAction)
				if useUpstreamStream {
					fullURL += "?alt=sse"
				}
				upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(body))
				if err != nil {
					return nil, "", err
				}
				upstreamReq.Header.Set("Content-Type", "application/json")
				upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
				if shouldMimicGeminiCLI {
					upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
				}
				return upstreamReq, "x-request-id", nil
			}
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
		if action == "countTokens" {
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
	if stream {
		streamRes, err := s.handleNativeStreamingResponse(c, resp, startTime, isOAuth)
		if err != nil {
			return nil, err
		}
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
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
			if requestID == "" {
				if responseID := strings.TrimSpace(stringValueFromAny(collected["responseId"])); responseID != "" {
					requestID = responseID
					c.Header("x-request-id", requestID)
				}
			}
			b, _ := json.Marshal(collected)
			c.Data(http.StatusOK, "application/json", b)
			usage = usageObj
		} else {
			usageResp, err := s.handleNativeNonStreamingResponse(c, resp, isOAuth)
			if err != nil {
				return nil, err
			}
			usage = usageResp
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
	return &ForwardResult{RequestID: requestID, Usage: *usage, Model: originalModel, UpstreamModel: mappedModel, SimulatedClient: simulatedClient, Stream: stream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ImageCount: imageCount, ImageSize: imageSize}, nil
}
func (s *GeminiMessagesCompatService) checkErrorPolicyInLoop(ctx context.Context, account *Account, resp *http.Response) (matched bool, rebuilt *http.Response) {
	if resp.StatusCode < 400 || s.rateLimitService == nil {
		return false, resp
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	_ = resp.Body.Close()
	rebuilt = &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(body))}
	policy := s.rateLimitService.CheckErrorPolicy(ctx, account, resp.StatusCode, body)
	return policy != ErrorPolicyNone, rebuilt
}
func (s *GeminiMessagesCompatService) shouldRetryGeminiUpstreamError(account *Account, statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504, 529:
		return true
	case 403:
		if account == nil || account.Type != AccountTypeOAuth {
			return false
		}
		oauthType := strings.ToLower(strings.TrimSpace(account.GetCredential("oauth_type")))
		if oauthType == "" && strings.TrimSpace(account.GetCredential("project_id")) != "" {
			oauthType = "code_assist"
		}
		return oauthType == "code_assist"
	default:
		return false
	}
}
func (s *GeminiMessagesCompatService) shouldFailoverGeminiUpstreamError(statusCode int) bool {
	switch statusCode {
	case 401, 403, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}
func sleepGeminiBackoff(attempt int) {
	delay := geminiRetryBaseDelay * time.Duration(1<<uint(attempt-1))
	if delay > geminiRetryMaxDelay {
		delay = geminiRetryMaxDelay
	}
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	jitter := time.Duration(float64(delay) * 0.2 * (r.Float64()*2 - 1))
	sleepFor := delay + jitter
	if sleepFor < 0 {
		sleepFor = 0
	}
	time.Sleep(sleepFor)
}
func estimateGeminiCountTokens(reqBody []byte) int {
	total := 0
	gjson.GetBytes(reqBody, "systemInstruction.parts").ForEach(func(_, part gjson.Result) bool {
		if t := strings.TrimSpace(part.Get("text").String()); t != "" {
			total += estimateTokensForText(t)
		}
		return true
	})
	gjson.GetBytes(reqBody, "contents").ForEach(func(_, content gjson.Result) bool {
		content.Get("parts").ForEach(func(_, part gjson.Result) bool {
			if t := strings.TrimSpace(part.Get("text").String()); t != "" {
				total += estimateTokensForText(t)
			}
			return true
		})
		return true
	})
	if total < 0 {
		return 0
	}
	return total
}
func estimateTokensForText(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	runes := []rune(s)
	if len(runes) == 0 {
		return 0
	}
	ascii := 0
	for _, r := range runes {
		if r <= 0x7f {
			ascii++
		}
	}
	asciiRatio := float64(ascii) / float64(len(runes))
	if asciiRatio >= 0.8 {
		return (len(runes) + 3) / 4
	}
	return len(runes)
}

type UpstreamHTTPResult struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func (s *GeminiMessagesCompatService) handleNativeNonStreamingResponse(c *gin.Context, resp *http.Response, isOAuth bool) (*ClaudeUsage, error) {
	if s.cfg != nil && s.cfg.Gateway.GeminiDebugResponseHeaders {
		logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] ========== Response Headers ==========")
		for key, values := range resp.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-ratelimit") {
				logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] %s: %v", key, values)
			}
		}
		logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] ========================================")
	}
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	respBody, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "upstream_error", "message": "Upstream response too large"}})
		}
		return nil, err
	}
	if isOAuth {
		unwrappedBody, uwErr := unwrapGeminiResponse(respBody)
		if uwErr == nil {
			respBody = unwrappedBody
		}
	}
	var geminiResp map[string]any
	_ = json.Unmarshal(respBody, &geminiResp)
	analysis := analyzeGeminiResponse(geminiResp, respBody)
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	if c.Writer.Header().Get("x-request-id") == "" && strings.TrimSpace(analysis.ResponseID) != "" {
		c.Header("x-request-id", analysis.ResponseID)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, respBody)
	if analysis.Usage != nil {
		return analysis.Usage, nil
	}
	return &ClaudeUsage{}, nil
}
func (s *GeminiMessagesCompatService) handleNativeStreamingResponse(c *gin.Context, resp *http.Response, startTime time.Time, isOAuth bool) (*geminiNativeStreamResult, error) {
	if s.cfg != nil && s.cfg.Gateway.GeminiDebugResponseHeaders {
		logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] ========== Streaming Response Headers ==========")
		for key, values := range resp.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-ratelimit") {
				logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] %s: %v", key, values)
			}
		}
		logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] ====================================================")
	}
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Status(resp.StatusCode)
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/event-stream; charset=utf-8"
	}
	c.Header("Content-Type", contentType)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	reader := bufio.NewReader(resp.Body)
	usage := &ClaudeUsage{}
	var firstTokenMs *int
	responseID := ""
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(trimmed, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				if payload == "" || payload == "[DONE]" {
					_, _ = io.WriteString(c.Writer, line)
					flusher.Flush()
				} else {
					var rawToWrite string
					rawToWrite = payload
					var rawBytes []byte
					if isOAuth {
						innerBytes, err := unwrapGeminiResponse([]byte(payload))
						if err == nil {
							rawToWrite = string(innerBytes)
							rawBytes = innerBytes
						}
					} else {
						rawBytes = []byte(payload)
					}
					var geminiResp map[string]any
					if json.Unmarshal(rawBytes, &geminiResp) == nil && geminiResp != nil {
						analysis := analyzeGeminiResponse(geminiResp, rawBytes)
						if analysis.Usage != nil {
							usage = analysis.Usage
						}
						if responseID == "" && strings.TrimSpace(analysis.ResponseID) != "" {
							responseID = analysis.ResponseID
						}
						if c.Writer.Header().Get("x-request-id") == "" && responseID != "" {
							c.Header("x-request-id", responseID)
						}
					} else if u := extractGeminiUsage(rawBytes); u != nil {
						usage = u
					}
					if firstTokenMs == nil {
						ms := int(time.Since(startTime).Milliseconds())
						firstTokenMs = &ms
					}
					if isOAuth {
						_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", rawToWrite)
					} else {
						_, _ = io.WriteString(c.Writer, line)
					}
					flusher.Flush()
				}
			} else {
				_, _ = io.WriteString(c.Writer, line)
				flusher.Flush()
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return &geminiNativeStreamResult{usage: usage, firstTokenMs: firstTokenMs, responseID: responseID}, nil
}

func setGeminiCountTokensSourceHeader(c *gin.Context, source geminiCountTokensSource) {
	if c == nil {
		return
	}
	if value := strings.TrimSpace(string(source)); value != "" {
		c.Header(geminiCountTokensSourceHeader, value)
	}
}

func getGeminiUpstreamRequestID(header http.Header, primaryKey string) string {
	return firstNonEmptyString(
		getGeminiHeaderValue(header, primaryKey),
		getGeminiHeaderValue(header, "x-goog-request-id"),
	)
}

func getGeminiHeaderValue(header http.Header, key string) string {
	key = strings.TrimSpace(key)
	if header == nil || key == "" {
		return ""
	}
	if value := strings.TrimSpace(header.Get(key)); value != "" {
		return value
	}
	for headerKey, values := range header {
		if !strings.EqualFold(headerKey, key) {
			continue
		}
		for _, value := range values {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func (s *GeminiMessagesCompatService) finishGeminiEstimatedCountTokensResponse(
	c *gin.Context,
	account *Account,
	originalModel string,
	mappedModel string,
	simulatedClient string,
	requestID string,
	body []byte,
	upstreamStatusCode int,
	message string,
	detail string,
	startTime time.Time,
) (*ForwardResult, error) {
	requestID = strings.TrimSpace(requestID)
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "countTokens upstream unavailable; estimated fallback used"
	}
	setGeminiCountTokensSourceHeader(c, geminiCountTokensSourceEstimated)
	if c != nil && requestID != "" && c.Writer != nil && c.Writer.Header().Get("x-request-id") == "" {
		c.Header("x-request-id", requestID)
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           RoutingPlatformForAccount(account),
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: upstreamStatusCode,
		UpstreamRequestID:  requestID,
		Kind:               "count_tokens_estimated",
		Message:            message,
		Detail:             strings.TrimSpace(detail),
	})
	logger.LegacyPrintf(
		"service.gemini_messages_compat",
		"Gemini account %d: countTokens fallback source=estimated status=%d request_id=%s model=%s upstream_model=%s reason=%s",
		account.ID,
		upstreamStatusCode,
		requestID,
		originalModel,
		mappedModel,
		message,
	)
	if c != nil {
		c.JSON(http.StatusOK, map[string]any{"totalTokens": estimateGeminiCountTokens(body)})
	}
	return &ForwardResult{
		RequestID:       requestID,
		Usage:           ClaudeUsage{},
		Model:           originalModel,
		UpstreamModel:   mappedModel,
		SimulatedClient: simulatedClient,
		Stream:          false,
		Duration:        time.Since(startTime),
		FirstTokenMs:    nil,
	}, nil
}
