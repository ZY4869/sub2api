package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *OpenAIGatewayService) ForwardNativeImagesGeneration(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
) (*OpenAIForwardResult, error) {
	return s.forwardNativeImages(ctx, c, account, body, "generations")
}

func (s *OpenAIGatewayService) ForwardNativeImagesEdits(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
) (*OpenAIForwardResult, error) {
	return s.forwardNativeImages(ctx, c, account, body, "edits")
}

func (s *OpenAIGatewayService) forwardNativeImages(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	action string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	contentType := "application/json"
	if c != nil && c.Request != nil {
		contentType = strings.TrimSpace(c.Request.Header.Get("Content-Type"))
	}

	originalModel, err := DetectOpenAIImageRequestModel(body, contentType)
	if err != nil {
		return nil, fmt.Errorf("parse image request model: %w", err)
	}
	mappedModel := resolveOpenAIForwardModel(account, originalModel, "")
	requestBody, rewrittenContentType, err := RewriteOpenAIImageRequestModel(body, contentType, mappedModel)
	if err != nil {
		return nil, fmt.Errorf("rewrite image request model: %w", err)
	}
	imageSize := DetectOpenAIImageRequestSize(requestBody, rewrittenContentType)
	normalizedRequest, err := NormalizeOpenAIImageRequest(requestBody, rewrittenContentType, action)
	if err != nil {
		return nil, newOpenAIImageRequestError("image_request_invalid", err.Error())
	}
	normalizedRequest.DisplayModelID = originalModel
	normalizedRequest.TargetModelID = mappedModel
	capabilityProfile, err := ValidateOpenAIImageCapabilities(normalizedRequest, OpenAIImageProtocolModeNative, mappedModel)
	if err != nil {
		return nil, err
	}
	if c != nil && c.Request != nil {
		ctx := EnsureRequestMetadata(c.Request.Context())
		SetImageCapabilityProfileMetadata(ctx, capabilityProfile.ID)
		c.Request = c.Request.WithContext(ctx)
	}
	SetOpenAIImageNormalizedTracePayload(c, "openai_native_images_normalized_request", normalizedRequest, capabilityProfile.ID)

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	upstreamReq, err := s.buildNativeImagesUpstreamRequest(ctx, c, account, requestBody, rewrittenContentType, token, action)
	if err != nil {
		return nil, fmt.Errorf("build native image request: %w", err)
	}
	if normalizedRequest.Stream {
		upstreamReq.Header.Set("accept", "text/event-stream")
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
			})
			if s.rateLimitService != nil {
				s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				ResponseHeaders:        resp.Header.Clone(),
				RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		return s.handleNativeImagesErrorResponse(resp, c, account)
	}

	if normalizedRequest.Stream {
		return s.forwardNativeImagesStream(
			ctx,
			resp,
			c,
			account,
			startTime,
			originalModel,
			mappedModel,
			firstNonEmptyString(strings.TrimSpace(normalizedRequest.Size), strings.TrimSpace(imageSize)),
		)
	}

	result, err := s.handleNativeImagesNonStreamingResponse(resp, c, originalModel, mappedModel, imageSize)
	if err != nil {
		return nil, err
	}
	result.Duration = time.Since(startTime)
	return result, nil
}

func (s *OpenAIGatewayService) buildNativeImagesUpstreamRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	contentType string,
	token string,
	action string,
) (*http.Request, error) {
	targetURL, err := resolveOpenAIImagesTargetURL(account, s.validateUpstreamBaseURL, action)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("authorization", "Bearer "+token)
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			if !openaiPassthroughAllowedHeaders[strings.ToLower(key)] {
				continue
			}
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	if req.Header.Get("accept") == "" {
		req.Header.Set("accept", "application/json")
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("user-agent", customUA)
	}
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", firstNonEmptyString(contentType, "application/json"))
	}
	return req, nil
}

func (s *OpenAIGatewayService) handleNativeImagesNonStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
	imageSize string,
) (*OpenAIForwardResult, error) {
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		return nil, err
	}
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("parse native image response: invalid json response")
	}
	if mappedModel != "" && mappedModel != originalModel {
		body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
	}

	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := "application/json"
	if s.cfg != nil && !s.cfg.Security.ResponseHeaders.Enabled {
		if upstreamType := resp.Header.Get("Content-Type"); upstreamType != "" {
			contentType = upstreamType
		}
	}
	c.Data(resp.StatusCode, contentType, body)

	return &OpenAIForwardResult{
		RequestID:     resp.Header.Get("x-request-id"),
		Model:         originalModel,
		BillingModel:  mappedModel,
		UpstreamModel: mappedModel,
		ImageCount:    CountOpenAIImageResponse(body),
		ImageSize:     ResolveOpenAIImageSizeTier(imageSize),
		MediaType:     "image",
	}, nil
}

func (s *OpenAIGatewayService) handleNativeImagesErrorResponse(
	resp *http.Response,
	c *gin.Context,
	account *Account,
) (*OpenAIForwardResult, error) {
	return s.handleCompatErrorResponse(resp, c, account, func(c *gin.Context, statusCode int, errType, message string, _ string) {
		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"type":    errType,
				"message": message,
			},
		})
	})
}
