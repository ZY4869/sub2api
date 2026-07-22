package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
)

func buildGeminiImageBatchPayload(job *ImageBatchJob, items []ImageBatchItem) ([]byte, error) {
	requests := make([]map[string]any, 0, len(items))
	for _, item := range items {
		requests = append(requests, map[string]any{
			"request": map[string]any{
				"contents": []map[string]any{{
					"role": "user",
					"parts": []map[string]string{{
						"text": item.Prompt,
					}},
				}},
				"generationConfig": map[string]any{
					"responseModalities": []string{"TEXT", "IMAGE"},
				},
			},
			"metadata": map[string]any{
				"key":       item.CustomID,
				"custom_id": item.CustomID,
			},
		})
	}
	return json.Marshal(map[string]any{
		"batch": map[string]any{
			"display_name": "sub2api-image-batch-" + strings.ReplaceAll(job.ID, "-", ""),
			"input_config": map[string]any{
				"requests": map[string]any{
					"requests": requests,
				},
			},
		},
	})
}

func buildVertexImageBatchPayload(job *ImageBatchJob, items []ImageBatchItem) ([]byte, error) {
	requests := make([]map[string]any, 0, len(items))
	for _, item := range items {
		requests = append(requests, map[string]any{
			"key": item.CustomID,
			"request": map[string]any{
				"contents": []map[string]any{{
					"role": "user",
					"parts": []map[string]string{{
						"text": item.Prompt,
					}},
				}},
				"generationConfig": map[string]any{
					"responseModalities": []string{"TEXT", "IMAGE"},
				},
			},
		})
	}
	return json.Marshal(map[string]any{
		"display_name": "sub2api-image-batch-" + strings.ReplaceAll(job.ID, "-", ""),
		"model":        strings.TrimPrefix(strings.TrimSpace(job.TargetModelID), "models/"),
		"requests":     requests,
	})
}

func extractImageBatchResultFileName(payload []byte) string {
	for _, path := range []string{
		"dest.fileName",
		"destination.fileName",
		"result.fileName",
		"output.fileName",
		"responseFile.fileName",
		"response_file.file_name",
		"archive.result_file_name",
	} {
		if value := strings.TrimSpace(gjson.GetBytes(payload, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func (s *ImageBatchService) indexResultPayload(ctx context.Context, job *ImageBatchJob, payload []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		customID := imageBatchLineCustomID(line)
		if customID == "" {
			continue
		}
		if msg := imageBatchLineError(line); msg != "" {
			if err := s.repo.MarkItemResult(ctx, job.ID, customID, ImageBatchItemFailed, 0, msg, GenerateSafeRequestID()); err != nil {
				return err
			}
			continue
		}
		outputs := imageBatchLineOutputs(line)
		if len(outputs) == 0 {
			if err := s.repo.MarkItemResult(ctx, job.ID, customID, ImageBatchItemFailed, 0, "未找到图片输出，请重试或联系支持。", GenerateSafeRequestID()); err != nil {
				return err
			}
			continue
		}
		for _, output := range outputs {
			output.JobID = job.ID
			output.CustomID = customID
			stored, err := s.prepareOutputForStorage(ctx, output)
			if err != nil {
				return err
			}
			if err := s.repo.UpsertOutput(ctx, &stored); err != nil {
				return err
			}
		}
		if err := s.repo.MarkItemResult(ctx, job.ID, customID, ImageBatchItemSuccess, len(outputs), "", ""); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return s.repo.RefreshJobCounts(ctx, job.ID)
}

func imageBatchLineCustomID(line []byte) string {
	for _, path := range []string{"key", "metadata.key", "metadata.custom_id", "custom_id", "request.metadata.key"} {
		if value := strings.TrimSpace(gjson.GetBytes(line, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func imageBatchLineError(line []byte) string {
	for _, path := range []string{"error.message", "response.error.message", "status.message"} {
		if value := strings.TrimSpace(gjson.GetBytes(line, path).String()); value != "" {
			return "批量生图条目失败：" + value
		}
	}
	return ""
}

func imageBatchLineOutputs(line []byte) []ImageBatchOutput {
	parts := gjson.GetBytes(line, "response.candidates.0.content.parts")
	if !parts.Exists() {
		parts = gjson.GetBytes(line, "candidates.0.content.parts")
	}
	outputs := make([]ImageBatchOutput, 0, 1)
	parts.ForEach(func(_, part gjson.Result) bool {
		inlineData := part.Get("inlineData")
		if !inlineData.Exists() {
			inlineData = part.Get("inline_data")
		}
		data := strings.TrimSpace(inlineData.Get("data").String())
		if data == "" {
			return true
		}
		content, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return true
		}
		contentType := firstNonEmptyString(inlineData.Get("mimeType").String(), inlineData.Get("mime_type").String(), "image/png")
		sum := sha256.Sum256(content)
		outputs = append(outputs, ImageBatchOutput{
			ContentType:    contentType,
			StorageBackend: "db",
			Content:        content,
			SizeBytes:      int64(len(content)),
			SHA256:         hex.EncodeToString(sum[:]),
		})
		return true
	})
	return outputs
}
