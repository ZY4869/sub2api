package service

import (
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"strings"
	"time"
)

const (
	openAIWSBetaV1Value                          = "responses_websockets=2026-02-04"
	openAIWSBetaV2Value                          = "responses_websockets=2026-02-06"
	openAIWSTurnStateHeader                      = "x-codex-turn-state"
	openAIWSTurnMetadataHeader                   = "x-codex-turn-metadata"
	openAIWSLogValueMaxLen                       = 160
	openAIWSHeaderValueMaxLen                    = 120
	openAIWSIDValueMaxLen                        = 64
	openAIWSEventLogHeadLimit                    = 20
	openAIWSEventLogEveryN                       = 50
	openAIWSBufferLogHeadLimit                   = 8
	openAIWSBufferLogEveryN                      = 20
	openAIWSPrewarmEventLogHead                  = 10
	openAIWSPayloadKeySizeTopN                   = 6
	openAIWSPayloadSizeEstimateDepth             = 3
	openAIWSPayloadSizeEstimateMaxBytes          = 64 * 1024
	openAIWSPayloadSizeEstimateMaxItems          = 16
	openAIWSEventFlushBatchSizeDefault           = 4
	openAIWSEventFlushIntervalDefault            = 25 * time.Millisecond
	openAIWSPayloadLogSampleDefault              = 0.2
	openAIWSPassthroughIdleTimeoutDefault        = time.Hour
	openAIWSStoreDisabledConnModeStrict          = "strict"
	openAIWSStoreDisabledConnModeAdaptive        = "adaptive"
	openAIWSStoreDisabledConnModeOff             = "off"
	openAIWSIngressStagePreviousResponseNotFound = "previous_response_not_found"
	openAIWSMaxPrevResponseIDDeletePasses        = 8
)

var openAIWSLogValueReplacer = strings.NewReplacer("error", "err", "fallback", "fb", "warning", "warnx", "failed", "fail")
var openAIWSIngressPreflightPingIdle = 20 * time.Second

func shouldLogOpenAIWSEvent(idx int, eventType string) bool {
	if idx <= openAIWSEventLogHeadLimit {
		return true
	}
	if openAIWSEventLogEveryN > 0 && idx%openAIWSEventLogEveryN == 0 {
		return true
	}
	if eventType == "error" || isOpenAIWSTerminalEvent(eventType) {
		return true
	}
	return false
}
func shouldLogOpenAIWSBufferedEvent(idx int) bool {
	if idx <= openAIWSBufferLogHeadLimit {
		return true
	}
	if openAIWSBufferLogEveryN > 0 && idx%openAIWSBufferLogEveryN == 0 {
		return true
	}
	return false
}
func populateOpenAIUsageFromResponseJSON(body []byte, usage *OpenAIUsage) {
	if usage == nil || len(body) == 0 {
		return
	}
	values := gjson.GetManyBytes(body, "usage.input_tokens", "usage.output_tokens", "usage.input_tokens_details.cached_tokens")
	usage.InputTokens = int(values[0].Int())
	usage.OutputTokens = int(values[1].Int())
	usage.CacheReadInputTokens = int(values[2].Int())
}
func getOpenAIGroupIDFromContext(c *gin.Context) int64 {
	if c == nil {
		return 0
	}
	value, exists := c.Get("api_key")
	if !exists {
		return 0
	}
	apiKey, ok := value.(*APIKey)
	if !ok || apiKey == nil || apiKey.GroupID == nil {
		return 0
	}
	return *apiKey.GroupID
}
