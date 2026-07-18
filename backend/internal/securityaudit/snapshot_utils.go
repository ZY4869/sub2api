package securityaudit

import "strings"

func normalizeSegmentsLatestUserFirst(values []promptSegment) []string {
	segments := make([]promptSegment, 0, len(values))
	for _, segment := range values {
		segment.text = strings.TrimSpace(segment.text)
		if segment.text != "" {
			segments = append(segments, segment)
		}
	}
	if len(segments) == 0 {
		return nil
	}
	priority := latestUserSegmentIndex(segments)
	out := []string{segments[priority].text}
	for i, segment := range segments {
		if i != priority {
			out = append(out, segment.text)
		}
	}
	return out
}

func latestUserSegmentIndex(segments []promptSegment) int {
	priority := len(segments) - 1
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i].user {
			return i
		}
	}
	return priority
}

func firstPresent(object map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := object[key]; ok {
			return value
		}
	}
	return nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
