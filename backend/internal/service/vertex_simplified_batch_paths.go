package service

import (
	"encoding/json"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type simplifiedVertexBatchPath struct {
	jobName string
	action  string
}

func parseSimplifiedVertexBatchPath(path string) (simplifiedVertexBatchPath, error) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return simplifiedVertexBatchPath{}, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BATCH_PATH_INVALID", "invalid simplified Vertex batch path")
	}

	segments := strings.Split(trimmed, "/")
	switch {
	case len(segments) >= 3 && segments[0] == "v1" && segments[1] == "vertex" && segments[2] == "batchPredictionJobs":
		return parseSimplifiedVertexBatchSegments(segments[3:])
	case len(segments) >= 2 && segments[0] == "vertex-batch" && segments[1] == "jobs":
		return parseSimplifiedVertexBatchSegments(segments[2:])
	default:
		return simplifiedVertexBatchPath{}, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BATCH_PATH_INVALID", "invalid simplified Vertex batch path")
	}
}

func parseSimplifiedVertexBatchSegments(segments []string) (simplifiedVertexBatchPath, error) {
	if len(segments) == 0 {
		return simplifiedVertexBatchPath{}, nil
	}
	jobName := strings.TrimSpace(segments[0])
	if jobName == "" {
		return simplifiedVertexBatchPath{}, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BATCH_PATH_INVALID", "invalid simplified Vertex batch path")
	}
	if idx := strings.Index(jobName, ":"); idx >= 0 {
		action := strings.TrimSpace(jobName[idx+1:])
		jobName = strings.TrimSpace(jobName[:idx])
		return simplifiedVertexBatchPath{jobName: jobName, action: action}, nil
	}
	if len(segments) > 1 {
		return simplifiedVertexBatchPath{}, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BATCH_PATH_INVALID", "invalid simplified Vertex batch path")
	}
	return simplifiedVertexBatchPath{jobName: jobName}, nil
}

func buildSimplifiedVertexBatchAccountPath(account *Account, jobName string, action string) string {
	base := strings.TrimRight(strings.TrimSpace(buildVertexBatchPredictionJobsPath(account)), "/")
	if base == "" {
		return ""
	}
	if strings.TrimSpace(jobName) == "" {
		return base
	}
	path := base + "/" + strings.TrimSpace(jobName)
	if strings.TrimSpace(action) != "" {
		path += ":" + strings.TrimSpace(action)
	}
	return path
}

func simplifiedVertexBatchPublicName(jobName string) string {
	jobName = strings.TrimSpace(strings.TrimPrefix(jobName, "batchPredictionJobs/"))
	if jobName == "" {
		return ""
	}
	return "batchPredictionJobs/" + jobName
}

func extractVertexBatchJobName(resourceName string) string {
	trimmed := strings.Trim(strings.TrimSpace(resourceName), "/")
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "batchPredictionJobs/") {
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "batchPredictionJobs/"))
	}
	parts := strings.Split(trimmed, "/")
	for index := 0; index < len(parts)-1; index++ {
		if parts[index] == "batchPredictionJobs" {
			name := strings.TrimSpace(parts[index+1])
			if cut := strings.Index(name, ":"); cut >= 0 {
				name = strings.TrimSpace(name[:cut])
			}
			return name
		}
	}
	return ""
}

func rewriteSimplifiedVertexBatchBody(body []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		return body
	}
	rewriteSimplifiedVertexBatchNameField(payload)
	if items, ok := payload["batchPredictionJobs"].([]any); ok {
		for _, rawItem := range items {
			item, _ := rawItem.(map[string]any)
			rewriteSimplifiedVertexBatchNameField(item)
		}
	}
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return rewritten
}

func rewriteSimplifiedVertexBatchNameField(item map[string]any) {
	if item == nil {
		return
	}
	name := extractVertexBatchJobName(stringMapValue(item, "name"))
	if name == "" {
		return
	}
	item["name"] = simplifiedVertexBatchPublicName(name)
}
