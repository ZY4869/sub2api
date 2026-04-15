package service

import (
	"bytes"
	"strings"

	"github.com/tidwall/gjson"
)

type GeminiRequestClassifier struct{}

func NewGeminiRequestClassifier() *GeminiRequestClassifier {
	return &GeminiRequestClassifier{}
}

func (c *GeminiRequestClassifier) ClassifySimulation(input BillingSimulationInput) *GeminiRequestClassification {
	operationType := normalizeBillingDimension(input.OperationType, "generate_content")
	batchMode := normalizeBillingActualBatchMode(input.BatchMode)
	cachePhase := normalizeBillingDimension(input.CachePhase, "")
	requestedServiceTier := normalizeGeminiRequestedServiceTier(input.ServiceTier)
	serviceTierExplicit := strings.TrimSpace(input.ServiceTier) != ""
	resolvedServiceTier := resolveGeminiResolvedServiceTier(requestedServiceTier, operationType, batchMode, cachePhase)
	classification := &GeminiRequestClassification{
		Surface:              normalizeBillingSurface(input.Surface),
		OperationType:        operationType,
		RequestedMode:        resolveGeminiRequestedMode(serviceTierExplicit, requestedServiceTier, operationType, batchMode, cachePhase),
		ResolvedMode:         resolveGeminiResolvedMode(requestedServiceTier, operationType, batchMode, cachePhase),
		RequestedServiceTier: normalizeBillingDimension(requestedServiceTier, BillingServiceTierStandard),
		ServiceTierExplicit:  serviceTierExplicit,
		ServiceTier:          resolvedServiceTier,
		BatchMode:            batchMode,
		InputModality:        normalizeBillingDimension(input.InputModality, "text"),
		OutputModality:       normalizeBillingDimension(input.OutputModality, inferSimulationOutputModality(input)),
		CachePhase:           cachePhase,
		GroundingKind:        normalizeBillingDimension(input.GroundingKind, ""),
		MediaType:            normalizeBillingDimension(input.OutputModality, ""),
	}
	classification.ChargeSource = inferGeminiChargeSource(classification)
	classification.MediaUnits = inferSimulationMediaUnits(input)
	return classification
}

func (c *GeminiRequestClassifier) ClassifyRequest(input GeminiBillingCalculationInput) *GeminiRequestClassification {
	surface := detectGeminiSurface(input.InboundEndpoint)
	operationType := detectGeminiOperationType(input.InboundEndpoint, input.RequestBody)
	batchMode := detectGeminiBatchMode(input.InboundEndpoint)
	cachePhase := detectGeminiCachePhase(input.InboundEndpoint)
	requestedServiceTier, serviceTierExplicit := parseGeminiRequestedServiceTier(input.RequestedServiceTier, input.RequestBody)
	resolvedServiceTierInput, resolvedServiceTierExplicit := parseGeminiResolvedServiceTier(input.ResolvedServiceTier)
	groundingKind := detectGeminiGroundingKind(input.RequestBody)
	inputModality, outputModality := detectGeminiModalities(input.RequestBody, input.MediaType, input.ImageCount, input.VideoRequests)
	resolvedServiceTierBase := requestedServiceTier
	if resolvedServiceTierExplicit {
		resolvedServiceTierBase = resolvedServiceTierInput
	}
	resolvedServiceTier := resolveGeminiResolvedServiceTier(resolvedServiceTierBase, operationType, batchMode, cachePhase)
	classification := &GeminiRequestClassification{
		Surface:              surface,
		OperationType:        operationType,
		RequestedMode:        resolveGeminiRequestedMode(serviceTierExplicit, requestedServiceTier, operationType, batchMode, cachePhase),
		ResolvedMode:         resolveGeminiResolvedMode(resolvedServiceTierBase, operationType, batchMode, cachePhase),
		RequestedServiceTier: normalizeBillingDimension(requestedServiceTier, BillingServiceTierStandard),
		ServiceTierExplicit:  serviceTierExplicit,
		ServiceTierDowngraded: serviceTierExplicit &&
			resolvedServiceTierExplicit &&
			geminiServiceTierEligible(operationType, batchMode, cachePhase) &&
			normalizeBillingActualServiceTier(requestedServiceTier) != normalizeBillingActualServiceTier(resolvedServiceTierInput),
		ServiceTier:    resolvedServiceTier,
		BatchMode:      batchMode,
		InputModality:  inputModality,
		OutputModality: outputModality,
		CachePhase:     cachePhase,
		GroundingKind:  groundingKind,
		MediaType:      normalizeBillingDimension(input.MediaType, outputModality),
		MediaUnits:     inferRequestMediaUnits(input, outputModality),
	}
	classification.ChargeSource = inferGeminiChargeSource(classification)
	return classification
}

func detectGeminiSurface(inboundEndpoint string) string {
	normalized := NormalizeInboundEndpoint(inboundEndpoint)
	switch {
	case strings.Contains(strings.TrimSpace(inboundEndpoint), "/v1beta/openai/"):
		return BillingSurfaceOpenAICompat
	case strings.Contains(strings.TrimSpace(inboundEndpoint), "/v1alpha/authTokens"),
		strings.Contains(strings.TrimSpace(inboundEndpoint), "/v1beta/live"):
		return BillingSurfaceGeminiLive
	case strings.Contains(strings.TrimSpace(inboundEndpoint), "/v1beta/interactions"):
		return BillingSurfaceInteractions
	case normalized == EndpointVertexSyncModels || normalized == EndpointVertexBatchJobs:
		return BillingSurfaceVertexExisting
	default:
		return BillingSurfaceGeminiNative
	}
}

func detectGeminiOperationType(inboundEndpoint string, body []byte) string {
	path := strings.TrimSpace(strings.ToLower(inboundEndpoint))
	switch {
	case strings.Contains(path, "/filesearchstores"):
		if detectFileSearchOperation(path, body) == "retrieval" {
			return "file_search_retrieval"
		}
		return "file_search_embedding"
	case strings.Contains(path, ":batchgeneratecontent"):
		return "generate_content"
	case strings.Contains(path, ":streamgeneratecontent"),
		strings.Contains(path, ":generatecontent"),
		strings.Contains(path, ":generateanswer"),
		strings.Contains(path, "/chat/completions"),
		strings.Contains(path, "/images/generations"),
		strings.Contains(path, "/videos"):
		return "generate_content"
	case strings.Contains(path, ":counttokens"):
		return "count_tokens"
	case strings.Contains(path, ":embedcontent"), strings.Contains(path, ":batchembedcontents"), strings.Contains(path, ":asyncbatchembedcontent"):
		return "embeddings"
	case strings.Contains(path, "/cachedcontents"):
		if strings.Contains(path, "/cachedcontents/") {
			return "cached_content_read"
		}
		return "cached_content_create"
	case strings.Contains(path, "/documents"):
		return "document_operation"
	case strings.Contains(path, "/operations"):
		return "operation_status"
	case strings.Contains(path, "/embeddings"):
		return "embeddings"
	case strings.Contains(path, "/files"):
		return "file_operation"
	case strings.Contains(path, "/batches"):
		return "batch_operation"
	case strings.Contains(path, "/v1alpha/authtokens"),
		strings.Contains(path, "/v1beta/live/auth-token"),
		strings.Contains(path, "/v1beta/live/auth-tokens"),
		strings.Contains(path, "/v1beta/live/authtokens"):
		return "auth_tokens"
	case strings.Contains(path, "/v1beta/live"):
		return "live_session"
	case strings.Contains(path, "/v1beta/interactions"):
		return "interaction"
	case strings.Contains(path, "/models"):
		return "models"
	default:
		return "generate_content"
	}
}

func detectFileSearchOperation(path string, body []byte) string {
	path = strings.TrimSpace(strings.ToLower(path))
	switch {
	case strings.Contains(path, ":search"), strings.Contains(path, ":query"), strings.Contains(path, "/search"), strings.Contains(path, "/retrieve"):
		return "retrieval"
	case strings.Contains(path, "/documents"), strings.Contains(path, ":import"), strings.Contains(path, ":upload"), strings.Contains(path, ":create"):
		return "embedding"
	}
	if bytes.Contains(bytes.ToLower(body), []byte(`"filesearch"`)) {
		return "retrieval"
	}
	return "embedding"
}

func detectGeminiCachePhase(inboundEndpoint string) string {
	path := strings.TrimSpace(strings.ToLower(inboundEndpoint))
	if !strings.Contains(path, "/cachedcontents") {
		return ""
	}
	if strings.Contains(path, "/cachedcontents/") {
		return "read"
	}
	return "create"
}

func detectGeminiGroundingKind(body []byte) string {
	lowerBody := bytes.ToLower(body)
	switch {
	case bytes.Contains(lowerBody, []byte(`"googlesearch"`)) || bytes.Contains(lowerBody, []byte(`"googlesearchretrieval"`)):
		return "search"
	case bytes.Contains(lowerBody, []byte(`"googlemaps"`)):
		return "maps"
	case bytes.Contains(lowerBody, []byte(`"urlcontext"`)):
		return "url_context"
	case bytes.Contains(lowerBody, []byte(`"filesearch"`)):
		return "file_search"
	default:
		return ""
	}
}

func detectGeminiModalities(body []byte, mediaType string, imageCount int, videoRequests int) (string, string) {
	lowerBody := bytes.ToLower(body)
	switch {
	case imageCount > 0:
		return "text", "image"
	case videoRequests > 0:
		return "text", "video"
	case strings.Contains(strings.ToLower(mediaType), "audio"):
		return "audio", "audio"
	case bytes.Contains(lowerBody, []byte(`"inlinedata"`)):
		switch {
		case bytes.Contains(lowerBody, []byte(`audio/`)):
			return "audio", inferBodyOutputModality(body)
		case bytes.Contains(lowerBody, []byte(`image/`)):
			return "image", inferBodyOutputModality(body)
		default:
			return "binary", inferBodyOutputModality(body)
		}
	default:
		return "text", inferBodyOutputModality(body)
	}
}

func inferBodyOutputModality(body []byte) string {
	model := strings.ToLower(gjson.GetBytes(body, "model").String())
	switch {
	case strings.Contains(model, "tts"):
		return "audio"
	case strings.Contains(model, "image"):
		return "image"
	case strings.Contains(model, "video"):
		return "video"
	default:
		for _, item := range gjson.GetBytes(body, "generationConfig.responseModalities").Array() {
			switch strings.ToLower(strings.TrimSpace(item.String())) {
			case "audio":
				return "audio"
			case "image":
				return "image"
			case "video":
				return "video"
			}
		}
		return "text"
	}
}

func detectGeminiBatchMode(inboundEndpoint string) string {
	path := strings.TrimSpace(strings.ToLower(inboundEndpoint))
	if strings.Contains(path, ":batchgeneratecontent") ||
		strings.Contains(path, ":batchembedcontents") ||
		strings.Contains(path, ":asyncbatchembedcontent") {
		return BillingBatchModeBatch
	}
	return BillingBatchModeRealtime
}

func inferSimulationOutputModality(input BillingSimulationInput) string {
	switch {
	case input.Charges.AudioOutputTokens > 0:
		return "audio"
	case input.Charges.ImageOutputs > 0 || input.ImageCount > 0:
		return "image"
	case input.Charges.VideoRequests > 0 || input.VideoRequests > 0:
		return "video"
	default:
		return "text"
	}
}

func inferSimulationMediaUnits(input BillingSimulationInput) int {
	switch {
	case input.Charges.ImageOutputs > 0:
		return int(input.Charges.ImageOutputs)
	case input.Charges.VideoRequests > 0:
		return int(input.Charges.VideoRequests)
	case input.MediaUnits > 0:
		return int(input.MediaUnits)
	case input.ImageCount > 0:
		return int(input.ImageCount)
	case input.VideoRequests > 0:
		return int(input.VideoRequests)
	default:
		return 0
	}
}

func inferRequestMediaUnits(input GeminiBillingCalculationInput, outputModality string) int {
	switch outputModality {
	case "image":
		return input.ImageCount
	case "video":
		return input.VideoRequests
	default:
		return 0
	}
}

func inferGeminiChargeSource(classification *GeminiRequestClassification) string {
	if classification == nil {
		return ""
	}
	switch classification.OperationType {
	case "file_search_embedding":
		return "file_search_embedding"
	case "file_search_retrieval":
		return "file_search_retrieval"
	}
	if classification.BatchMode == BillingBatchModeBatch {
		return "model_batch"
	}
	if classification.CachePhase != "" {
		return "cache"
	}
	if classification.GroundingKind != "" {
		return "grounding"
	}
	return "billing_rule"
}

func normalizeBillingSurface(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", BillingSurfaceGeminiNative:
		return BillingSurfaceGeminiNative
	case "compat", BillingSurfaceOpenAICompat:
		return BillingSurfaceOpenAICompat
	case BillingSurfaceGeminiLive:
		return BillingSurfaceGeminiLive
	case BillingSurfaceInteractions:
		return BillingSurfaceInteractions
	case "vertex", BillingSurfaceVertexExisting:
		return BillingSurfaceVertexExisting
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}

func normalizeBillingBatchMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", BillingBatchModeAny:
		return BillingBatchModeAny
	case "false", BillingBatchModeRealtime:
		return BillingBatchModeRealtime
	case "true", BillingBatchModeBatch:
		return BillingBatchModeBatch
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}

func normalizeBillingDimension(value string, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return strings.TrimSpace(strings.ToLower(fallback))
	}
	return value
}

func parseGeminiRequestedServiceTier(value string, body []byte) (string, bool) {
	if strings.TrimSpace(value) != "" {
		return normalizeGeminiRequestedServiceTier(value), true
	}
	if extracted, explicit := extractGeminiRequestedServiceTierValue(body); explicit {
		return extracted, true
	}
	return BillingServiceTierStandard, false
}

func parseGeminiResolvedServiceTier(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", false
	}
	return normalizeGeminiRequestedServiceTier(value), true
}

func extractGeminiRequestedServiceTierFromBody(body []byte) *string {
	normalized, explicit := extractGeminiRequestedServiceTierValue(body)
	if !explicit {
		return nil
	}
	return &normalized
}

func extractGeminiRequestedServiceTierValue(body []byte) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	for _, path := range []string{"service_tier", "serviceTier"} {
		raw := gjson.GetBytes(body, path)
		if !raw.Exists() {
			continue
		}
		return normalizeGeminiRequestedServiceTier(raw.String()), true
	}
	return "", false
}

func normalizeGeminiRequestedServiceTier(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if normalized := normalizeOpenAIServiceTier(value); normalized != nil {
		return normalizeBillingDimension(*normalized, BillingServiceTierStandard)
	}
	switch value {
	case "", "default":
		return BillingServiceTierStandard
	default:
		return value
	}
}

func resolveGeminiResolvedServiceTier(requestedServiceTier string, operationType string, batchMode string, cachePhase string) string {
	requestedServiceTier = normalizeGeminiRequestedServiceTier(requestedServiceTier)
	if !geminiServiceTierEligible(operationType, batchMode, cachePhase) {
		return BillingServiceTierStandard
	}
	return normalizeBillingActualServiceTier(requestedServiceTier)
}

func resolveGeminiRequestedMode(serviceTierExplicit bool, requestedServiceTier string, operationType string, batchMode string, cachePhase string) string {
	requestedServiceTier = normalizeGeminiRequestedServiceTier(requestedServiceTier)
	if serviceTierExplicit {
		return normalizeBillingActualServiceTier(requestedServiceTier)
	}
	return resolveGeminiResolvedMode(requestedServiceTier, operationType, batchMode, cachePhase)
}

func resolveGeminiResolvedMode(requestedServiceTier string, operationType string, batchMode string, cachePhase string) string {
	if normalizeBillingActualBatchMode(batchMode) == BillingBatchModeBatch {
		return BillingBatchModeBatch
	}
	if normalizeBillingDimension(cachePhase, "") != "" {
		return "cache"
	}
	return resolveGeminiResolvedServiceTier(requestedServiceTier, operationType, batchMode, cachePhase)
}

func resolveGeminiModeFallbackReason(classification *GeminiRequestClassification) string {
	if classification == nil || !classification.ServiceTierExplicit {
		return ""
	}
	if normalizeBillingActualBatchMode(classification.BatchMode) == BillingBatchModeBatch {
		return "service_tier_ignored_for_batch"
	}
	if normalizeBillingDimension(classification.CachePhase, "") != "" {
		return "service_tier_ignored_for_cache"
	}
	if !geminiServiceTierOperationEligible(classification.OperationType) {
		return "service_tier_ignored_for_non_realtime_operation"
	}
	if classification.ServiceTierDowngraded {
		return "service_tier_downgraded_by_upstream"
	}
	return ""
}

func geminiServiceTierEligible(operationType string, batchMode string, cachePhase string) bool {
	return geminiServiceTierOperationEligible(operationType) &&
		normalizeBillingActualBatchMode(batchMode) == BillingBatchModeRealtime &&
		normalizeBillingDimension(cachePhase, "") == ""
}

func geminiServiceTierOperationEligible(operationType string) bool {
	switch normalizeBillingDimension(operationType, "generate_content") {
	case "generate_content", "interaction":
		return true
	default:
		return false
	}
}

func detectGroundingQueryCount(body []byte, groundingKind string) int {
	groundingKind = normalizeBillingDimension(groundingKind, "")
	if groundingKind == "" || len(body) == 0 {
		return 0
	}
	lower := bytes.ToLower(body)
	switch groundingKind {
	case "search":
		if bytes.Contains(lower, []byte(`"googlesearch"`)) || bytes.Contains(lower, []byte(`"googlesearchretrieval"`)) {
			return 1
		}
	case "maps":
		if bytes.Contains(lower, []byte(`"googlemaps"`)) {
			return 1
		}
	}
	return 0
}
