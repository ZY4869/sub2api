package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type codexImportEntry struct {
	Index int
	Value any
}

func parseCodexSessionImportEntries(req CodexSessionImportRequest) ([]codexImportEntry, error) {
	contents := make([]string, 0, 1+len(req.Contents))
	if strings.TrimSpace(req.Content) != "" {
		contents = append(contents, req.Content)
	}
	for _, content := range req.Contents {
		if strings.TrimSpace(content) != "" {
			contents = append(contents, content)
		}
	}

	entries := make([]codexImportEntry, 0)
	for _, content := range contents {
		values, err := parseCodexSessionImportContent(content)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			entries = append(entries, codexImportEntry{Index: len(entries) + 1, Value: value})
		}
	}
	return entries, nil
}

func parseCodexSessionImportContent(content string) ([]any, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, nil
	}
	if looksLikeJSON(trimmed) {
		values, err := decodeCodexJSONStream(trimmed)
		if err != nil {
			if strings.Contains(trimmed, "\n") {
				if lineValues, lineErr := parseCodexSessionImportLines(trimmed); lineErr == nil {
					return lineValues, nil
				}
			}
			return nil, fmt.Errorf("JSON 解析失败: %w", err)
		}
		return flattenCodexImportValues(values), nil
	}
	return parseCodexSessionImportLines(trimmed)
}

func parseCodexSessionImportLines(content string) ([]any, error) {
	values := make([]any, 0)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if looksLikeJSON(line) {
			lineValues, err := decodeCodexJSONStream(line)
			if err != nil {
				return nil, fmt.Errorf("第 %d 行 JSON 解析失败: %w", len(values)+1, err)
			}
			values = append(values, flattenCodexImportValues(lineValues)...)
			continue
		}
		values = append(values, line)
	}
	return values, nil
}

func decodeCodexJSONStream(content string) ([]any, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()
	values := make([]any, 0, 1)
	for {
		var value any
		err := decoder.Decode(&value)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, errors.New("空 JSON 内容")
	}
	return values, nil
}

func flattenCodexImportValues(values []any) []any {
	out := make([]any, 0, len(values))
	var appendValue func(any)
	appendValue = func(value any) {
		if arr, ok := value.([]any); ok {
			for _, item := range arr {
				appendValue(item)
			}
			return
		}
		out = append(out, value)
	}
	for _, value := range values {
		appendValue(value)
	}
	return out
}

func looksLikeJSON(content string) bool {
	if content == "" {
		return false
	}
	return content[0] == '{' || content[0] == '['
}

func firstCodexString(obj map[string]any, paths ...[]string) string {
	for _, path := range paths {
		value, ok := codexPathValue(obj, path)
		if !ok {
			continue
		}
		if text := codexStringValue(value); text != "" {
			return text
		}
	}
	return ""
}

func codexPathValue(obj map[string]any, path []string) (any, bool) {
	var current any = obj
	for _, key := range path {
		currentObj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok := currentObj[key]
		if !ok {
			return nil, false
		}
		current = value
	}
	return current, true
}

func codexStringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(v, 'f', -1, 64))
	case float32:
		return strings.TrimSpace(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	default:
		return ""
	}
}
