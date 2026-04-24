package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	defaultOpenAIImageTestPrompt       = "生成一张可爱的橘猫宇航员贴纸，插画风格，干净的纯色背景。"
	defaultOpenAIImageTestSize         = "1024x1024"
	defaultOpenAIImageTestOutputFormat = "png"
)

func (s *AccountTestService) testOpenAIImageAccountConnection(
	c *gin.Context,
	account *Account,
	modelID string,
	prompt string,
	_ string,
	_ string,
) error {
	if account == nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	testModelID := strings.TrimSpace(modelID)
	if testModelID == "" {
		return s.sendErrorAndEnd(c, "Model is required for image test")
	}

	s.prepareTestStream(c)
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)
	return s.forwardOpenAIImageTestAndSendEvents(c, account, testModelID, prompt)
}

func (s *AccountTestService) forwardOpenAIImageTestAndSendEvents(c *gin.Context, account *Account, modelID string, prompt string) error {
	if c == nil {
		return fmt.Errorf("gin context is required")
	}
	if account == nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	requestBody := buildOpenAIImageTestRequestBody(modelID, prompt)

	statusCode := 0
	var responseBody []byte
	var forwardErr error

	if s.openAIGatewayService != nil {
		child, recorder := s.prepareForwardTestContext(c, http.MethodPost, EndpointImagesGen, requestBody)
		child.Request.Header.Set("Accept", "application/json")

		mode := ResolveAccountImageProtocolMode(account)
		switch mode {
		case OpenAIImageProtocolModeCompat:
			_, forwardErr = s.openAIGatewayService.ForwardCompatImages(
				c.Request.Context(),
				child,
				account,
				requestBody,
				"application/json",
				"generation",
				modelID,
			)
		default:
			_, forwardErr = s.openAIGatewayService.ForwardNativeImagesGeneration(c.Request.Context(), child, account, requestBody)
		}

		if recorder != nil {
			statusCode = recorder.Code
			responseBody = recorder.Body.Bytes()
		}
	} else {
		statusCode, responseBody, forwardErr = s.forwardOpenAIImageTestDirect(c.Request.Context(), account, requestBody)
	}

	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	if statusCode >= http.StatusBadRequest {
		if len(responseBody) == 0 && forwardErr != nil {
			return s.sendErrorAndEnd(c, forwardErr.Error())
		}
		return s.sendFailedTestResponse(c, c.Request.Context(), account, statusCode, responseBody, "API returned")
	}
	if len(responseBody) == 0 {
		if forwardErr != nil {
			return s.sendErrorAndEnd(c, forwardErr.Error())
		}
		return s.sendErrorAndEnd(c, "Image test returned an empty response")
	}

	imageURL, mimeType, ok := extractOpenAIImagesResponsePreview(responseBody, defaultOpenAIImageTestOutputFormat)
	if !ok || strings.TrimSpace(imageURL) == "" {
		return s.sendErrorAndEnd(c, "Image test did not return a previewable image")
	}

	s.sendEvent(c, TestEvent{Type: "image", ImageURL: imageURL, MimeType: mimeType})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

func (s *AccountTestService) forwardOpenAIImageTestDirect(ctx context.Context, account *Account, body []byte) (int, []byte, error) {
	if account == nil {
		return 0, nil, fmt.Errorf("account is required")
	}

	var authToken string
	if account.IsOAuth() {
		if s.openAITokenProvider != nil {
			token, err := s.openAITokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return 0, nil, fmt.Errorf("failed to get access token: %w", err)
			}
			authToken = token
		} else {
			authToken = account.GetOpenAIAccessToken()
		}
		if strings.TrimSpace(authToken) == "" {
			return 0, nil, fmt.Errorf("no access token available")
		}
	} else if account.Type == AccountTypeAPIKey {
		authToken = account.GetOpenAIApiKey()
		if strings.TrimSpace(authToken) == "" {
			return 0, nil, fmt.Errorf("no api key available")
		}
	} else {
		return 0, nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}

	targetURL, err := resolveOpenAIImagesTargetURL(account, s.validateUpstreamBaseURL, "generations")
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return resp.StatusCode, respBody, nil
}

func buildOpenAIImageTestRequestBody(modelID string, prompt string) []byte {
	payload := map[string]any{
		"model":         strings.TrimSpace(modelID),
		"prompt":        normalizeOpenAIImageTestPrompt(prompt),
		"size":          defaultOpenAIImageTestSize,
		"output_format": defaultOpenAIImageTestOutputFormat,
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func normalizeOpenAIImageTestPrompt(prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	if trimmed != "" {
		return trimmed
	}
	return defaultOpenAIImageTestPrompt
}

func extractOpenAIImagesResponsePreview(body []byte, defaultOutputFormat string) (string, string, bool) {
	if len(body) == 0 {
		return "", "", false
	}
	outputFormat := strings.TrimSpace(gjson.GetBytes(body, "output_format").String())
	if outputFormat == "" {
		outputFormat = strings.TrimSpace(defaultOutputFormat)
	}
	mimeType := openAIImageOutputFormatToMimeType(outputFormat)

	if b64 := strings.TrimSpace(gjson.GetBytes(body, "data.0.b64_json").String()); b64 != "" {
		return "data:" + mimeType + ";base64," + b64, mimeType, true
	}
	if url := strings.TrimSpace(gjson.GetBytes(body, "data.0.url").String()); url != "" {
		return url, "", true
	}
	return "", "", false
}

func openAIImageOutputFormatToMimeType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
	}
}
