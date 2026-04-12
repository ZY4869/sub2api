package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/gin-gonic/gin"
)

type AccountTestMode string

const (
	AccountTestModeRealForward AccountTestMode = "real_forward"
	AccountTestModeHealthCheck AccountTestMode = "health_check"
)

const (
	accountTestRuntimeModeKey           = "account_test_runtime_mode"
	accountTestRuntimePlatformKey       = "account_test_runtime_platform"
	accountTestRuntimeSourceProtocolKey = "account_test_runtime_source_protocol"
	accountTestRuntimeClientKey         = "account_test_runtime_client"
	accountTestRuntimeInboundKey        = "account_test_runtime_inbound_endpoint"
	accountTestRuntimeCompatPathKey     = "account_test_runtime_compat_path"
	accountTestRuntimeTargetProviderKey = "account_test_runtime_target_provider"
	accountTestRuntimeTargetModelIDKey  = "account_test_runtime_target_model_id"
	accountTestRuntimeResolvedModelKey  = "account_test_runtime_resolved_model_id"
	accountTestRuntimeMetaSource        = "account_test"
)

func normalizeAccountTestMode(value string) AccountTestMode {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case string(AccountTestModeHealthCheck):
		return AccountTestModeHealthCheck
	case string(AccountTestModeRealForward), "":
		return AccountTestModeRealForward
	default:
		return AccountTestModeRealForward
	}
}

func accountTestModeLabel(mode AccountTestMode) string {
	switch mode {
	case AccountTestModeHealthCheck:
		return "Health check"
	default:
		return "Real forward"
	}
}

func accountTestPlatformLabel(platform string) string {
	switch strings.TrimSpace(strings.ToLower(platform)) {
	case PlatformOpenAI:
		return "OpenAI"
	case PlatformAnthropic:
		return "Anthropic"
	case PlatformGemini:
		return "Gemini"
	case PlatformAntigravity:
		return "Antigravity"
	case PlatformKiro:
		return "Kiro"
	case PlatformCopilot:
		return "GitHub Copilot"
	case PlatformProtocolGateway:
		return "Protocol Gateway"
	default:
		return strings.TrimSpace(platform)
	}
}

func (s *AccountTestService) setResolvedTestRuntimeMeta(c *gin.Context, meta accountTestRuntimeMeta) {
	if c == nil {
		return
	}
	c.Set(accountTestRuntimeModeKey, string(meta.Mode))
	c.Set(accountTestRuntimePlatformKey, strings.TrimSpace(meta.RuntimePlatform))
	c.Set(accountTestRuntimeSourceProtocolKey, normalizeTestSourceProtocol(meta.SourceProtocol))
	c.Set(accountTestRuntimeClientKey, strings.TrimSpace(meta.SimulatedClient))
	c.Set(accountTestRuntimeInboundKey, strings.TrimSpace(meta.InboundEndpoint))
	c.Set(accountTestRuntimeCompatPathKey, strings.TrimSpace(meta.CompatPath))
	c.Set(accountTestRuntimeTargetProviderKey, NormalizeModelProvider(meta.TargetProvider))
	c.Set(accountTestRuntimeTargetModelIDKey, strings.TrimSpace(meta.TargetModelID))
	c.Set(accountTestRuntimeResolvedModelKey, strings.TrimSpace(meta.ResolvedModelID))
}

func (s *AccountTestService) sendResolvedTestRuntimeMetaEvents(c *gin.Context) {
	if c == nil {
		return
	}
	if mode, ok := c.Get(accountTestRuntimeModeKey); ok {
		modeValue := normalizeAccountTestMode(toTrimmedString(mode))
		s.sendTestRuntimeMetaEvent(
			c,
			"test_mode",
			string(modeValue),
			accountTestModeLabel(modeValue),
			accountTestRuntimeMetaSource,
			"Test mode: "+accountTestModeLabel(modeValue),
		)
	}
	if platform, ok := c.Get(accountTestRuntimePlatformKey); ok {
		platformValue := toTrimmedString(platform)
		if platformValue != "" {
			s.sendTestRuntimeMetaEvent(
				c,
				"resolved_platform",
				platformValue,
				accountTestPlatformLabel(platformValue),
				accountTestRuntimeMetaSource,
				"Effective platform: "+accountTestPlatformLabel(platformValue),
			)
		}
	}
	if sourceProtocol, ok := c.Get(accountTestRuntimeSourceProtocolKey); ok {
		protocolValue := normalizeTestSourceProtocol(toTrimmedString(sourceProtocol))
		if protocolValue != "" {
			s.sendTestRuntimeMetaEvent(
				c,
				"resolved_protocol",
				protocolValue,
				gatewayTestSourceProtocolLabel(protocolValue),
				accountTestRuntimeMetaSource,
				"Gateway source protocol: "+gatewayTestSourceProtocolLabel(protocolValue),
			)
		}
	}
	if simulatedClient, ok := c.Get(accountTestRuntimeClientKey); ok {
		clientValue := toTrimmedString(simulatedClient)
		if clientLabel := gatewayTestSimulatedClientLabel(clientValue); clientLabel != "" {
			s.sendTestRuntimeMetaEvent(
				c,
				"simulated_client",
				clientValue,
				clientLabel,
				accountTestRuntimeMetaSource,
				"Gateway simulated client: "+clientLabel,
			)
		}
	}
	if inboundEndpoint, ok := c.Get(accountTestRuntimeInboundKey); ok {
		inboundValue := strings.TrimSpace(toTrimmedString(inboundEndpoint))
		if inboundValue != "" {
			s.sendTestRuntimeMetaEvent(
				c,
				"inbound_endpoint",
				inboundValue,
				inboundValue,
				accountTestRuntimeMetaSource,
				"Inbound endpoint: "+inboundValue,
			)
		}
	}
	if compatPath, ok := c.Get(accountTestRuntimeCompatPathKey); ok {
		compatValue := strings.TrimSpace(toTrimmedString(compatPath))
		if compatValue != "" {
			s.sendTestRuntimeMetaEvent(
				c,
				"compat_path",
				compatValue,
				compatValue,
				accountTestRuntimeMetaSource,
				"Compatibility path: "+compatValue,
			)
		}
	}
	if targetProvider, ok := c.Get(accountTestRuntimeTargetProviderKey); ok {
		providerValue := NormalizeModelProvider(toTrimmedString(targetProvider))
		if providerValue != "" {
			s.sendTestRuntimeMetaEvent(
				c,
				"target_provider",
				providerValue,
				FormatProviderLabel(providerValue),
				accountTestRuntimeMetaSource,
				"Target provider: "+FormatProviderLabel(providerValue),
			)
		}
	}
	if targetModelID, ok := c.Get(accountTestRuntimeTargetModelIDKey); ok {
		modelValue := strings.TrimSpace(toTrimmedString(targetModelID))
		if modelValue != "" {
			s.sendTestRuntimeMetaEvent(
				c,
				"target_model_id",
				modelValue,
				modelValue,
				accountTestRuntimeMetaSource,
				"Target model: "+modelValue,
			)
		}
	}
	if resolvedModelID, ok := c.Get(accountTestRuntimeResolvedModelKey); ok {
		modelValue := strings.TrimSpace(toTrimmedString(resolvedModelID))
		if modelValue != "" {
			s.sendTestRuntimeMetaEvent(
				c,
				"resolved_model_id",
				modelValue,
				modelValue,
				accountTestRuntimeMetaSource,
				"Resolved model: "+modelValue,
			)
		}
	}
}

func (s *AccountTestService) sendTestRuntimeMetaEvent(c *gin.Context, key string, value string, label string, source string, text string) {
	if c == nil || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
		return
	}
	s.sendEvent(c, TestEvent{
		Type: "content",
		Text: text,
		Data: map[string]any{
			"kind":   "runtime_meta",
			"key":    key,
			"value":  value,
			"label":  label,
			"source": source,
		},
	})
}

func toTrimmedString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func (s *AccountTestService) testAccountConnectionRealForward(c *gin.Context, account *Account, modelID string, prompt string, sourceProtocol string, simulatedClient string) error {
	if account == nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	switch RoutingPlatformForAccount(account) {
	case PlatformAntigravity:
		return s.testAntigravityRealForwardConnection(c, account, modelID, prompt)
	}

	if account.IsOpenAI() {
		return s.testOpenAIRealForwardConnection(c, account, modelID, simulatedClient)
	}

	if account.IsGemini() {
		return s.testGeminiRealForwardConnection(c, account, modelID, prompt)
	}

	return s.testClaudeRealForwardConnection(c, account, modelID, sourceProtocol)
}

func (s *AccountTestService) testClaudeRealForwardConnection(c *gin.Context, account *Account, modelID string, sourceProtocol string) error {
	if s.gatewayService == nil {
		return s.testClaudeAccountConnection(c, account, modelID, sourceProtocol, "")
	}

	testModelID := strings.TrimSpace(modelID)
	if testModelID == "" {
		testModelID = claude.DefaultTestModel
	}

	var payload map[string]any
	var err error
	if account.IsOAuth() || IsClaudeClientMimicEnabled(account, sourceProtocol) {
		payload, err = createTestPayload(testModelID)
		if err != nil {
			return s.sendErrorAndEnd(c, "Failed to create test payload")
		}
	} else {
		payload = createAnthropicStandardTestPayload(testModelID)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to encode test payload")
	}
	parsed, err := ParseGatewayRequest(body, PlatformAnthropic)
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to parse real forward test payload")
	}

	s.prepareTestStream(c)
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	child, recorder := s.prepareForwardTestContext(c, http.MethodPost, "/v1/messages", body)
	_, forwardErr := s.gatewayService.Forward(c.Request.Context(), child, account, parsed)
	return s.relayForwardRecorderStream(c, account, recorder, forwardErr, s.processClaudeStream)
}

func (s *AccountTestService) testOpenAIRealForwardConnection(c *gin.Context, account *Account, modelID string, simulatedClient string) error {
	if s.openAIGatewayService == nil {
		return s.testOpenAIAccountConnection(c, account, modelID, "", simulatedClient)
	}

	requestFormat := ResolveOpenAITextRequestFormatForAccount(account, "")
	testModelID := strings.TrimSpace(modelID)
	if testModelID == "" {
		testModelID = defaultOpenAIOAuthTestModelID(c.Request.Context(), account, s.modelRegistryService)
	}
	body, err := json.Marshal(createOpenAITestPayloadForRequestFormat(testModelID, requestFormat, isChatGPTOpenAIOAuthAccount(account)))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to encode OpenAI test payload")
	}

	s.prepareTestStream(c)
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	targetPath := "/v1/responses"
	if requestFormat == GatewayOpenAIRequestFormatChatCompletions {
		targetPath = "/v1/chat/completions"
	}
	child, recorder := s.prepareForwardTestContext(c, http.MethodPost, targetPath, body)
	if requestFormat == GatewayOpenAIRequestFormatResponses {
		SetOpenAIClientTransport(child, OpenAIClientTransportHTTP)
	}
	if strings.TrimSpace(simulatedClient) == GatewayClientProfileCodex {
		child.Request.Header.Set("User-Agent", codexCLIUserAgent)
		child.Request.Header.Set("Originator", "codex_cli_rs")
	}

	var forwardErr error
	if requestFormat == GatewayOpenAIRequestFormatChatCompletions {
		_, forwardErr = s.openAIGatewayService.ForwardAsChatCompletions(c.Request.Context(), child, account, body, "", "")
		return s.relayForwardRecorderStream(c, account, recorder, forwardErr, s.processOpenAIChatCompletionsStream)
	}

	_, forwardErr = s.openAIGatewayService.Forward(c.Request.Context(), child, account, body)
	return s.relayForwardRecorderStream(c, account, recorder, forwardErr, s.processOpenAIStream)
}

func (s *AccountTestService) testGeminiRealForwardConnection(c *gin.Context, account *Account, modelID string, prompt string) error {
	if s.geminiCompatService == nil {
		return s.testGeminiAccountConnection(c, account, modelID, prompt, "", "")
	}

	testModelID := strings.TrimSpace(modelID)
	if testModelID == "" {
		testModelID = defaultGeminiTestModelID(account)
	}
	testModelID = s.resolveTestModelID(c.Request.Context(), account, testModelID)
	body := createGeminiTestPayload(testModelID, prompt)

	s.prepareTestStream(c)
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	targetPath := "/v1beta/models/" + testModelID + ":streamGenerateContent"
	child, recorder := s.prepareForwardTestContext(c, http.MethodPost, targetPath, body)
	_, forwardErr := s.geminiCompatService.ForwardNative(
		c.Request.Context(),
		child,
		account,
		testModelID,
		"streamGenerateContent",
		true,
		body,
	)
	return s.relayForwardRecorderStream(c, account, recorder, forwardErr, s.processGeminiStream)
}

func (s *AccountTestService) testAntigravityRealForwardConnection(c *gin.Context, account *Account, modelID string, prompt string) error {
	if s.antigravityGatewayService == nil {
		return s.routeAntigravityTest(c, account, modelID, prompt)
	}

	testModelID := strings.TrimSpace(modelID)
	if testModelID == "" {
		testModelID = "claude-sonnet-4-5"
	}

	s.prepareTestStream(c)
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	if strings.HasPrefix(strings.ToLower(testModelID), "gemini-") {
		body := createGeminiTestPayload(testModelID, prompt)
		targetPath := "/v1beta/models/" + testModelID + ":streamGenerateContent"
		child, recorder := s.prepareForwardTestContext(c, http.MethodPost, targetPath, body)
		_, forwardErr := s.antigravityGatewayService.ForwardGemini(
			c.Request.Context(),
			child,
			account,
			testModelID,
			"streamGenerateContent",
			true,
			body,
			false,
		)
		return s.relayForwardRecorderStream(c, account, recorder, forwardErr, s.processGeminiStream)
	}

	body, err := json.Marshal(createAnthropicStandardTestPayload(testModelID))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to encode Antigravity Claude test payload")
	}
	child, recorder := s.prepareForwardTestContext(c, http.MethodPost, "/v1/messages", body)
	_, forwardErr := s.antigravityGatewayService.Forward(c.Request.Context(), child, account, body, false)
	return s.relayForwardRecorderStream(c, account, recorder, forwardErr, s.processClaudeStream)
}

func (s *AccountTestService) prepareForwardTestContext(parent *gin.Context, method string, targetPath string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	child, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, targetPath, bytes.NewReader(body))
	if parent != nil && parent.Request != nil {
		req = req.WithContext(parent.Request.Context())
		req.RemoteAddr = parent.Request.RemoteAddr
		copyHeaderIfPresent(req.Header, parent.Request.Header, "Accept-Language")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "sub2api-account-test/real-forward")
	child.Request = req
	return child, recorder
}

func copyHeaderIfPresent(dst http.Header, src http.Header, key string) {
	if dst == nil || src == nil {
		return
	}
	if value := strings.TrimSpace(src.Get(key)); value != "" {
		dst.Set(key, value)
	}
}

func (s *AccountTestService) prepareTestStream(c *gin.Context) {
	if c == nil {
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()
}

func (s *AccountTestService) relayForwardRecorderStream(
	c *gin.Context,
	account *Account,
	recorder *httptest.ResponseRecorder,
	forwardErr error,
	parser func(*gin.Context, io.Reader) error,
) error {
	if recorder == nil {
		if forwardErr != nil {
			return s.sendErrorAndEnd(c, forwardErr.Error())
		}
		return s.sendErrorAndEnd(c, "Real forward test did not return a response")
	}

	statusCode := recorder.Code
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	body := recorder.Body.Bytes()
	if statusCode >= http.StatusBadRequest {
		if len(body) == 0 && forwardErr != nil {
			return s.sendErrorAndEnd(c, forwardErr.Error())
		}
		ctx := context.Background()
		if c != nil && c.Request != nil {
			ctx = c.Request.Context()
		}
		return s.sendFailedTestResponse(c, ctx, account, statusCode, body, "API returned")
	}
	if len(body) == 0 {
		if forwardErr != nil {
			return s.sendErrorAndEnd(c, forwardErr.Error())
		}
		return s.sendErrorAndEnd(c, "Real forward test returned an empty response")
	}
	if parser == nil {
		s.sendEvent(c, TestEvent{Type: "content", Text: strings.TrimSpace(string(body))})
		s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
		return nil
	}
	return parser(c, bytes.NewReader(body))
}
