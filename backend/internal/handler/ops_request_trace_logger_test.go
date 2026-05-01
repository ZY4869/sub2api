package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildOpsTraceNormalizeResult_UsesAccountAwareOpenAIEndpointPair(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(ctxKeyInboundEndpoint, EndpointResponses)
	setOpsEndpointContext(c, "gpt-5.4", service.RequestTypeSync)
	setOpsSelectedAccountDetails(c, &service.Account{
		ID:       42,
		Platform: service.PlatformProtocolGateway,
		Type:     service.AccountTypeAPIKey,
		Extra: map[string]any{
			"gateway_protocol":              service.GatewayProtocolOpenAI,
			"gateway_openai_request_format": service.GatewayOpenAIRequestFormatChatCompletions,
		},
	})

	normalize, usage := buildOpsTraceNormalizeResult(
		c,
		nil,
		[]byte(`{"model":"gpt-5.4"}`),
		[]byte(`{"id":"chatcmpl_1","model":"gpt-5.4","usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}}`),
	)

	require.Equal(t, service.PlatformProtocolGateway, normalize.Platform)
	require.Equal(t, EndpointResponses, normalize.ProtocolIn)
	require.Equal(t, EndpointChatCompletions, normalize.ProtocolOut)
	require.Equal(t, "openai_compat", normalize.Channel)
	require.Equal(t, "/v1/responses", normalize.RoutePath)
	require.Equal(t, "gpt-5.4", normalize.ActualUpstreamModel)
	require.Equal(t, "chatcmpl_1", normalize.UpstreamRequestID)
	require.Equal(t, 7, usage.inputTokens)
	require.Equal(t, 3, usage.outputTokens)
	require.Equal(t, 10, usage.totalTokens)
}

func TestEnrichOpsTraceResponseMetadata_GeminiCanonicalEndpointUsesGoogleParser(t *testing.T) {
	result := &service.ProtocolNormalizeResult{}
	usage := &opsTraceUsage{}

	enrichOpsTraceResponseMetadata(map[string]any{
		"responseId":   "resp-gemini-1",
		"modelVersion": "gemini-2.5-pro",
		"usageMetadata": map[string]any{
			"promptTokenCount":     12,
			"candidatesTokenCount": 5,
			"totalTokenCount":      17,
		},
	}, EndpointGeminiModels, result, usage)

	require.Equal(t, "resp-gemini-1", result.UpstreamRequestID)
	require.Equal(t, "gemini-2.5-pro", result.ActualUpstreamModel)
	require.Equal(t, 12, usage.inputTokens)
	require.Equal(t, 5, usage.outputTokens)
	require.Equal(t, 17, usage.totalTokens)
}

func TestBuildOpsTraceNormalizeResult_OpsRequestTraceResponsesImageToolMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(ctxKeyInboundEndpoint, EndpointResponses)
	setOpsEndpointContext(c, "gpt-5.4-mini", service.RequestTypeSync)
	applyResponsesImageToolTraceMetadata(
		c,
		service.PlatformOpenAI,
		"gpt-5.4-mini",
		"gpt-image-2",
		service.PublicImageToolRouteReason,
	)
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	service.SetImageProtocolModeMetadata(ctx, service.OpenAIImageProtocolModeCompat)
	service.SetImageRequestSurfaceMetadata(ctx, "responses_tool")
	service.SetImageSizeTierMetadata(ctx, service.OpenAIImageSizeTier2K)
	service.SetImageCapabilityProfileMetadata(ctx, "openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on")
	service.SetOpenAIResponsesImageGenCompatMetadata(ctx, service.OpenAIResponsesCompatMetadata{
		Enabled:                   true,
		Source:                    service.OpenAIResponsesImagegenCompatSourceMultipart,
		ReferenceImageCount:       2,
		ReferenceImageBytesBefore: 4096,
		ReferenceImageBytesAfter:  2048,
		ReferenceImagesNormalized: true,
		ImageGenerationSize:       "1536x1024",
	})
	c.Request = c.Request.WithContext(ctx)

	normalize, _ := buildOpsTraceNormalizeResult(
		c,
		nil,
		[]byte(`{"model":"gpt-5.4-mini","tools":[{"type":"image_generation","model":"gpt-image-2"}]}`),
		[]byte(`{"id":"resp_123","model":"gpt-5.4-mini","usage":{"input_tokens":8,"output_tokens":2,"total_tokens":10}}`),
	)

	require.Equal(t, service.PublicImageToolRouteFamily, normalize.ImageRouteFamily)
	require.Equal(t, "generations", normalize.ImageAction)
	require.Equal(t, service.PlatformOpenAI, normalize.ImageResolvedProvider)
	require.Equal(t, "gpt-5.4-mini", normalize.ImageDisplayModelID)
	require.Equal(t, "gpt-image-2", normalize.ImageTargetModelID)
	require.Equal(t, service.EndpointResponses, normalize.ImageUpstreamEndpoint)
	require.Equal(t, service.EndpointResponses, normalize.ImageRequestFormat)
	require.Equal(t, service.PublicImageToolRouteReason, normalize.ImageRouteReason)
	require.Equal(t, service.OpenAIImageProtocolModeCompat, normalize.ImageProtocolMode)
	require.Equal(t, "responses_tool", normalize.ImageRequestSurface)
	require.Equal(t, service.OpenAIImageSizeTier2K, normalize.ImageSizeTier)
	require.Equal(t, "openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on", normalize.ImageCapabilityProfile)
	require.True(t, normalize.ImagegenCompat)
	require.False(t, normalize.ImagegenCompatRejected)
	require.Equal(t, service.OpenAIResponsesImagegenCompatSourceMultipart, normalize.ImagegenCompatSource)
	require.Equal(t, 2, normalize.ImagegenCompatReferenceImageCount)
	require.Equal(t, int64(4096), normalize.ImagegenCompatReferenceImageBytesBefore)
	require.Equal(t, int64(2048), normalize.ImagegenCompatReferenceImageBytesAfter)
	require.True(t, normalize.ImagegenCompatNormalized)
	require.Equal(t, "1536x1024", normalize.ImagegenCompatImageGenerationSize)
	require.Equal(t, "1536x1024", normalize.MediaResolution)
}

func TestBuildOpsRequestTraceInput_OpsRequestTraceImageRouteMetricsCoexist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	responsesInput := buildOpsTraceInputForTest(
		t,
		http.MethodPost,
		"/v1/responses",
		http.StatusOK,
		45*time.Millisecond,
		[]byte(`{"model":"gpt-5.4-mini","tools":[{"type":"image_generation","model":"gpt-image-2"}]}`),
		[]byte(`{"id":"resp_123","model":"gpt-5.4-mini","usage":{"input_tokens":8,"output_tokens":2,"total_tokens":10}}`),
		func(c *gin.Context) {
			setOpsEndpointContext(c, "gpt-5.4-mini", service.RequestTypeSync)
			applyResponsesImageToolTraceMetadata(
				c,
				service.PlatformCopilot,
				"gpt-5.4-mini",
				"gpt-image-2",
				service.PublicImageToolRouteReason,
			)
		},
	)

	require.Equal(t, service.PublicImageToolRouteFamily, responsesInput.Trace.Normalize.ImageRouteFamily)
	require.Equal(t, service.PlatformOpenAI, responsesInput.Trace.Normalize.ImageResolvedProvider)
	require.Equal(t, service.EndpointResponses, responsesInput.Trace.Normalize.ImageUpstreamEndpoint)

	publicInput := buildOpsTraceInputForTest(
		t,
		http.MethodPost,
		"/v1/images/generations",
		http.StatusBadRequest,
		20*time.Millisecond,
		[]byte(`{"model":"gemini-2.5-flash-image","prompt":"hello"}`),
		[]byte(`{"error":{"message":"unsupported"}}`),
		func(c *gin.Context) {
			setOpsEndpointContext(c, "gemini-2.5-flash-image", service.RequestTypeSync)
			ctx := service.EnsureRequestMetadata(c.Request.Context())
			service.SetImageRouteFamilyMetadata(ctx, service.PublicImageRouteFamily)
			service.SetImageActionMetadata(ctx, "generations")
			service.SetImageResolvedProviderMetadata(ctx, service.PlatformGemini)
			service.SetImageDisplayModelIDMetadata(ctx, "gemini-2.5-flash-image")
			service.SetImageTargetModelIDMetadata(ctx, "gemini-2.5-flash-image")
			service.SetImageUpstreamEndpointMetadata(ctx, "/v1beta/openai/images/generations")
			service.SetImageRequestFormatMetadata(ctx, service.EndpointImagesGen)
			service.SetImageRouteReasonMetadata(ctx, service.PublicImageRouteReasonUnsupported)
			c.Request = c.Request.WithContext(ctx)
		},
	)

	require.Equal(t, service.PublicImageRouteFamily, publicInput.Trace.Normalize.ImageRouteFamily)
	require.Equal(t, service.PlatformGemini, publicInput.Trace.Normalize.ImageResolvedProvider)
	require.Equal(t, "/v1beta/openai/images/generations", publicInput.Trace.Normalize.ImageUpstreamEndpoint)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(2), snapshot.ImageRouteTotal)
	require.Equal(t, int64(1), snapshot.ImageRouteSuccessTotal)
	require.Equal(t, int64(1), snapshot.ImageRouteFailureTotal)
	require.Equal(t, int64(1), snapshot.ImageRouteByFamily[service.PublicImageToolRouteFamily])
	require.Equal(t, int64(1), snapshot.ImageRouteByFamily[service.PublicImageRouteFamily])
	require.Equal(t, int64(1), snapshot.ImageRouteByProvider[service.PlatformOpenAI])
	require.Equal(t, int64(1), snapshot.ImageRouteByProvider[service.PlatformGemini])
	require.Equal(t, int64(1), snapshot.ImageRouteSuccessByFamily[service.PublicImageToolRouteFamily])
	require.Equal(t, int64(1), snapshot.ImageRouteFailureByFamily[service.PublicImageRouteFamily])
	require.Equal(t, int64(1), snapshot.ImageRouteSuccessByProvider[service.PlatformOpenAI])
	require.Equal(t, int64(1), snapshot.ImageRouteFailureByProvider[service.PlatformGemini])
	require.GreaterOrEqual(t, snapshot.ImageRouteLatencyMsTotal, int64(65))
	require.GreaterOrEqual(t, snapshot.ImageRouteLatencyMsByFamily[service.PublicImageToolRouteFamily], int64(45))
	require.GreaterOrEqual(t, snapshot.ImageRouteLatencyMsByFamily[service.PublicImageRouteFamily], int64(20))
}

func TestBuildOpsTraceNormalizeResult_OpsRequestTraceResponsesImageToolRejectMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(ctxKeyInboundEndpoint, EndpointResponses)
	setOpsEndpointContext(c, "gpt-5.4-mini", service.RequestTypeSync)
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	service.SetOpenAIResponsesImageGenCompatMetadata(ctx, service.OpenAIResponsesCompatMetadata{
		Rejected:            true,
		RejectCode:          "multipart_stream_unsupported",
		SourceGuess:         service.OpenAIResponsesImagegenCompatSourceMultipart,
		ReferenceImageCount: 1,
	})
	c.Request = c.Request.WithContext(ctx)

	normalize, _ := buildOpsTraceNormalizeResult(
		c,
		nil,
		[]byte(`{"model":"gpt-5.4-mini","input":"$imagegen hidden prompt","reference_images":[{"image_url":"data:image/png;base64,AAAA"}]}`),
		[]byte(`{"error":{"code":"multipart_stream_unsupported"}}`),
	)

	require.False(t, normalize.ImagegenCompat)
	require.True(t, normalize.ImagegenCompatRejected)
	require.Equal(t, "multipart_stream_unsupported", normalize.ImagegenCompatRejectCode)
	require.Equal(t, service.OpenAIResponsesImagegenCompatSourceMultipart, normalize.ImagegenCompatSourceGuess)
	require.Equal(t, 1, normalize.ImagegenCompatReferenceImageCount)
	require.False(t, normalize.ImagegenCompatNormalized)
}

func TestRecordImageRouteRuntimeMetrics_RecordsResponsesImageToolFailure(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	status := http.StatusBadGateway
	recordImageRouteRuntimeMetrics(
		service.ProtocolNormalizeResult{
			ImageRouteFamily:       service.PublicImageToolRouteFamily,
			ImageResolvedProvider:  service.PlatformOpenAI,
			ImageProtocolMode:      service.OpenAIImageProtocolModeCompat,
			ImageAction:            "edit",
			ImageSizeTier:          service.OpenAIImageSizeTier4K,
			ImageCapabilityProfile: "openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on",
		},
		status,
		&status,
		120,
	)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.ResponsesImageToolFailureTotal)
	require.Equal(t, int64(1), snapshot.ResponsesImageToolFailureByProvider[service.PlatformOpenAI])
	require.Equal(t, int64(1), snapshot.ImageRouteByProtocolMode[service.OpenAIImageProtocolModeCompat])
	require.Equal(t, int64(1), snapshot.ImageRouteFailureByProtocolMode[service.OpenAIImageProtocolModeCompat])
	require.Equal(t, int64(1), snapshot.ImageRouteByAction["edit"])
	require.Equal(t, int64(1), snapshot.ImageRouteFailureByAction["edit"])
	require.Equal(t, int64(1), snapshot.ImageRouteBySizeTier[service.OpenAIImageSizeTier4K])
	require.Equal(t, int64(1), snapshot.ImageRouteFailureBySizeTier[service.OpenAIImageSizeTier4K])
	require.Equal(t, int64(1), snapshot.ImageRouteByCapabilityProfile["openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on"])
	require.Equal(t, int64(1), snapshot.ImageRouteFailureByCapabilityProfile["openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on"])
	require.Equal(t, int64(1), snapshot.ImageRouteFailureByUpstreamStatus["502"])
}

func buildOpsTraceInputForTest(
	t *testing.T,
	method string,
	path string,
	statusCode int,
	duration time.Duration,
	requestBody []byte,
	responseBody []byte,
	configure func(c *gin.Context),
) *service.OpsRecordRequestTraceInput {
	t.Helper()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(ctxKeyInboundEndpoint, NormalizeInboundEndpoint(path))
	setOpsRequestContext(c, "", false, requestBody)
	if configure != nil {
		configure(c)
	}

	writer := acquireOpsRequestTraceCaptureWriter(c.Writer)
	defer releaseOpsRequestTraceCaptureWriter(writer)
	c.Writer = writer
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Status(statusCode)
	if len(responseBody) > 0 {
		_, err := c.Writer.Write(responseBody)
		require.NoError(t, err)
	}

	input := buildOpsRequestTraceInput(nil, c, writer, time.Now().Add(-duration))
	require.NotNil(t, input)
	return input
}
