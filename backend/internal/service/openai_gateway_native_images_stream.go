package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *OpenAIGatewayService) forwardNativeImagesStream(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	startTime time.Time,
	originalModel string,
	mappedModel string,
	imageSize string,
) (*OpenAIForwardResult, error) {
	if c == nil {
		return nil, fmt.Errorf("streaming native images requires gin context")
	}

	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	if v := resp.Header.Get("x-request-id"); v != "" {
		c.Header("x-request-id", v)
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	bufferedWriter := bufio.NewWriterSize(c.Writer, 4*1024)
	flushBuffered := func() error {
		if err := bufferedWriter.Flush(); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	defer putSSEScannerBuf64K(scanBuf)
	scanner.Buffer(scanBuf[:0], maxLineSize)

	imageCount := 0
	needModelReplace := originalModel != mappedModel
	clientDisconnected := false

	processLine := func(line string) {
		if data, ok := extractOpenAISSEDataLine(line); ok {
			imageCount += countOpenAINativeImagesStreamOutputImages(data)
			if needModelReplace && mappedModel != "" && strings.Contains(data, mappedModel) {
				line = s.replaceModelInSSELine(line, mappedModel, originalModel)
			}
		}
		if clientDisconnected {
			return
		}
		if _, err := bufferedWriter.WriteString(line); err != nil {
			clientDisconnected = true
			logger.LegacyPrintf("service.openai_gateway", "Client disconnected during native images streaming, continuing to drain upstream for billing")
			return
		}
		if _, err := bufferedWriter.WriteString("\n"); err != nil {
			clientDisconnected = true
			logger.LegacyPrintf("service.openai_gateway", "Client disconnected during native images streaming, continuing to drain upstream for billing")
			return
		}
		if err := flushBuffered(); err != nil {
			clientDisconnected = true
			logger.LegacyPrintf("service.openai_gateway", "Client disconnected during native images streaming flush, continuing to drain upstream for billing")
		}
	}

	for scanner.Scan() {
		processLine(scanner.Text())
	}

	if !clientDisconnected {
		if err := flushBuffered(); err != nil {
			clientDisconnected = true
			logger.LegacyPrintf("service.openai_gateway", "Client disconnected during native images final flush, returning collected image count")
		}
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.LegacyPrintf("service.openai_gateway", "Context canceled during native images streaming, returning collected image count")
			return buildNativeImagesStreamingResult(resp, startTime, originalModel, mappedModel, imageSize, imageCount), nil
		}
		if clientDisconnected {
			logger.LegacyPrintf("service.openai_gateway", "Upstream read error after client disconnect (native images): %v, returning collected image count", err)
			return buildNativeImagesStreamingResult(resp, startTime, originalModel, mappedModel, imageSize, imageCount), nil
		}
		if errors.Is(err, bufio.ErrTooLong) {
			return buildNativeImagesStreamingResult(resp, startTime, originalModel, mappedModel, imageSize, imageCount), fmt.Errorf("native images stream line too long: %w", err)
		}
		return buildNativeImagesStreamingResult(resp, startTime, originalModel, mappedModel, imageSize, imageCount), fmt.Errorf("native images stream read error: %w", err)
	}

	return buildNativeImagesStreamingResult(resp, startTime, originalModel, mappedModel, imageSize, imageCount), nil
}

func buildNativeImagesStreamingResult(
	resp *http.Response,
	startTime time.Time,
	originalModel string,
	mappedModel string,
	imageSize string,
	imageCount int,
) *OpenAIForwardResult {
	requestID := ""
	if resp != nil {
		requestID = resp.Header.Get("x-request-id")
	}
	return &OpenAIForwardResult{
		RequestID:     requestID,
		Model:         originalModel,
		BillingModel:  mappedModel,
		UpstreamModel: mappedModel,
		Stream:        true,
		ImageCount:    imageCount,
		ImageSize:     ResolveOpenAIImageSizeTier(imageSize),
		MediaType:     "image",
		Duration:      time.Since(startTime),
	}
}

func countOpenAINativeImagesStreamOutputImages(data string) int {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" || trimmed == "[DONE]" {
		return 0
	}
	if !gjson.Valid(trimmed) {
		return 0
	}

	eventType := strings.TrimSpace(gjson.Get(trimmed, "type").String())
	eventTypeLower := strings.ToLower(eventType)

	if strings.Contains(eventTypeLower, "partial_image") || gjson.Get(trimmed, "partial_image_index").Exists() {
		return 0
	}

	if eventType != "" {
		if !strings.Contains(eventTypeLower, "completed") {
			return 0
		}
		countArray := countOpenAIImageLikeItemsInArray(trimmed, "data")
		if countArray > 0 {
			return countArray
		}
		if gjson.Get(trimmed, "b64_json").Exists() || gjson.Get(trimmed, "url").Exists() {
			return 1
		}
		return 0
	}

	countArray := countOpenAIImageLikeItemsInArray(trimmed, "data")
	if countArray > 0 {
		return countArray
	}
	if gjson.Get(trimmed, "b64_json").Exists() || gjson.Get(trimmed, "url").Exists() {
		return 1
	}
	return 0
}

func countOpenAIImageLikeItemsInArray(data string, path string) int {
	arr := gjson.Get(data, path)
	if !arr.Exists() || !arr.IsArray() {
		return 0
	}
	count := 0
	for _, item := range arr.Array() {
		if item.Get("b64_json").Exists() || item.Get("url").Exists() {
			count++
		}
	}
	return count
}
