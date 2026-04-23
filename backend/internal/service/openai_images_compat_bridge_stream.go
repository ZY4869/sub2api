package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) forwardCompatImagesStream(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	normalizedRequest *NormalizedImageRequest,
	responsesBody []byte,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	if c != nil && c.Request != nil {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	upstreamReq, err := s.buildUpstreamRequest(ctx, c, account, responsesBody, token, true, "", false)
	if err != nil {
		return nil, err
	}
	upstreamReq.Header.Set("accept", "text/event-stream")

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		return s.handleCompatErrorResponse(resp, c, account, func(c *gin.Context, statusCode int, errType, message, _ string) {
			c.JSON(statusCode, gin.H{"error": gin.H{"type": errType, "message": message}})
		})
	}

	if c == nil {
		return nil, fmt.Errorf("streaming compat images requires gin context")
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	accumulator := apicompat.NewBufferedResponseAccumulator()
	usage := OpenAIUsage{}
	outputCount := 0
	partialIndex := 0
	eventPrefix := compatImageStreamEventPrefix(normalizedRequest.Operation)

	scanner := bufio.NewScanner(resp.Body)
	scanBuf := getSSEScannerBuf64K()
	defer putSSEScannerBuf64K(scanBuf)
	scanner.Buffer(scanBuf[:0], defaultMaxLineSize)
	for scanner.Scan() {
		line := scanner.Text()
		data, ok := extractOpenAISSEDataLine(line)
		if !ok || data == "" || data == "[DONE]" {
			continue
		}

		var event apicompat.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		accumulator.ProcessEvent(&event)

		switch event.Type {
		case "response.image_generation_call.partial_image":
			if strings.TrimSpace(event.PartialImageB64) == "" {
				continue
			}
			if err := writeCompatImagesSSEEvent(c.Writer, flusher, eventPrefix+".partial_image", map[string]any{
				"type":                eventPrefix + ".partial_image",
				"b64_json":            strings.TrimSpace(event.PartialImageB64),
				"partial_image_index": partialIndex,
			}); err != nil {
				return nil, err
			}
			partialIndex++
		case "response.completed", "response.done":
			if event.Response != nil {
				accumulator.SupplementResponseOutput(event.Response)
				if event.Response.Usage != nil {
					usage = OpenAIUsage{
						InputTokens:          event.Response.Usage.InputTokens,
						OutputTokens:         event.Response.Usage.OutputTokens,
						CacheReadInputTokens: 0,
					}
					if event.Response.Usage.InputTokensDetails != nil {
						usage.CacheReadInputTokens = event.Response.Usage.InputTokensDetails.CachedTokens
					}
				}
			}
			usagePayload := compatImagesUsagePayload(event.Response)
			finalImages := compatImagesCompletedPayloads(event.Response)
			for index, payload := range finalImages {
				if usagePayload != nil && index == len(finalImages)-1 {
					payload["usage"] = usagePayload
				}
				payload["type"] = eventPrefix + ".completed"
				if err := writeCompatImagesSSEEvent(c.Writer, flusher, eventPrefix+".completed", payload); err != nil {
					return nil, err
				}
				outputCount++
			}
		case "response.failed":
			message := extractOpenAISSEErrorMessage([]byte(data))
			if message == "" {
				message = "Upstream image stream failed"
			}
			return nil, fmt.Errorf("compat image stream failed: %s", message)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if c != nil && c.Request != nil {
		SetImageOutputCountMetadata(c.Request.Context(), outputCount)
	}
	return &OpenAIForwardResult{
		RequestID:     resp.Header.Get("x-request-id"),
		Usage:         usage,
		Model:         normalizedRequest.DisplayModelID,
		BillingModel:  normalizedRequest.TargetModelID,
		UpstreamModel: normalizedRequest.TargetModelID,
		ImageCount:    outputCount,
		ImageSize:     ResolveOpenAIImageSizeTier(normalizedRequest.Size),
		MediaType:     "image",
		Duration:      time.Since(startTime),
	}, nil
}

func compatImageStreamEventPrefix(operation string) string {
	if normalizeOpenAIImageOperation(operation) == "edit" {
		return "image_edit"
	}
	return "image_generation"
}

func compatImagesCompletedPayloads(response *apicompat.ResponsesResponse) []map[string]any {
	if response == nil {
		return nil
	}
	payloads := make([]map[string]any, 0, 2)
	for _, item := range response.Output {
		if item.Type != "message" {
			continue
		}
		for _, content := range item.Content {
			if content.Type != "output_image" || strings.TrimSpace(content.ImageURL) == "" {
				continue
			}
			payload := map[string]any{}
			if b64 := stripDataURLBase64(content.ImageURL); b64 != "" {
				payload["b64_json"] = b64
			} else {
				payload["url"] = strings.TrimSpace(content.ImageURL)
			}
			if len(payload) > 0 {
				payloads = append(payloads, payload)
			}
		}
	}
	return payloads
}

func compatImagesUsagePayload(response *apicompat.ResponsesResponse) map[string]any {
	if response == nil || response.Usage == nil {
		return nil
	}
	raw, err := json.Marshal(response.Usage)
	if err != nil {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	return payload
}

func writeCompatImagesSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventName string, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", strings.TrimSpace(eventName), raw); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
