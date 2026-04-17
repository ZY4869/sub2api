package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

const baiduDocumentAIHealthCheckPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO6C2i8AAAAASUVORK5CYII="

func (s *AccountTestService) testBaiduDocumentAIAccountConnection(c *gin.Context, account *Account) error {
	if account == nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	payload, err := base64.StdEncoding.DecodeString(baiduDocumentAIHealthCheckPNGBase64)
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to prepare Document AI health-check sample")
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	client := newBaiduDocumentAIClient(s.httpUpstream, s.tlsFingerprintProfileService)
	if account.GetBaiduDocumentAIAsyncBearerToken() != "" {
		modelID := DocumentAIModelPPOCRV5Server
		s.sendEvent(c, TestEvent{Type: "test_start", Model: modelID})
		s.sendEvent(c, TestEvent{
			Type: "content",
			Text: fmt.Sprintf("Document AI mode: async (%s)", modelID),
			Data: map[string]any{"kind": "runtime_meta", "key": "resolved_platform", "value": PlatformBaiduDocumentAI},
		})
		s.sendEvent(c, TestEvent{
			Type: "content",
			Text: fmt.Sprintf("Async base URL: %s", account.GetBaiduDocumentAIAsyncBaseURL()),
			Data: map[string]any{"kind": "runtime_meta", "key": "endpoint", "value": account.GetBaiduDocumentAIAsyncBaseURL()},
		})
		result, err := client.submitAsyncJob(c.Request.Context(), account, DocumentAISubmitJobInput{
			Model:       modelID,
			SourceType:  DocumentAISourceTypeFile,
			FileName:    "health-check.png",
			ContentType: "image/png",
			FileSize:    int64(len(payload)),
			FileBytes:   payload,
		})
		if err != nil {
			return s.sendErrorAndEnd(c, err.Error())
		}
		s.sendEvent(c, TestEvent{
			Type: "content",
			Text: fmt.Sprintf("Async job submitted: %s", result.ProviderJobID),
			Data: map[string]any{"kind": "runtime_meta", "key": "provider_job_id", "value": result.ProviderJobID},
		})
		statusResult, statusErr := client.getAsyncJobStatus(context.Background(), account, result.ProviderJobID)
		if statusErr != nil {
			return s.sendErrorAndEnd(c, statusErr.Error())
		}
		s.sendEvent(c, TestEvent{
			Type: "content",
			Text: fmt.Sprintf("Async poll status: %s", statusResult.Status),
			Data: map[string]any{"kind": "runtime_meta", "key": "job_status", "value": statusResult.Status},
		})
		s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
		return nil
	}

	apiURL := account.GetBaiduDocumentAIDirectAPIURL(DocumentAIModelPPOCRV5Server)
	if account.GetBaiduDocumentAIDirectToken() == "" || apiURL == "" {
		return s.sendErrorAndEnd(c, "Baidu Document AI account is missing async token or direct PP-OCRv5 API_URL")
	}

	modelID := DocumentAIModelPPOCRV5Server
	s.sendEvent(c, TestEvent{Type: "test_start", Model: modelID})
	s.sendEvent(c, TestEvent{
		Type: "content",
		Text: fmt.Sprintf("Document AI mode: direct (%s)", modelID),
		Data: map[string]any{"kind": "runtime_meta", "key": "resolved_platform", "value": PlatformBaiduDocumentAI},
	})
	s.sendEvent(c, TestEvent{
		Type: "content",
		Text: fmt.Sprintf("Direct API URL: %s", apiURL),
		Data: map[string]any{"kind": "runtime_meta", "key": "endpoint", "value": apiURL},
	})
	result, err := client.parseDirect(c.Request.Context(), account, DocumentAIParseDirectInput{
		Model:       modelID,
		SourceType:  DocumentAISourceTypeFile,
		FileType:    DocumentAIFileTypeImage,
		FileName:    "health-check.png",
		ContentType: "image/png",
		FileSize:    int64(len(payload)),
		FileBytes:   payload,
	})
	if err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}
	s.sendEvent(c, TestEvent{
		Type: "content",
		Text: fmt.Sprintf("Direct parse completed: pages=%d text=%t", result.Envelope.PageCount, strings.TrimSpace(result.Envelope.Text) != ""),
		Data: map[string]any{"kind": "runtime_meta", "key": "job_status", "value": result.Envelope.Status},
	})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}
