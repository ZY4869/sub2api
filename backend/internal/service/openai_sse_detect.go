package service

import "bytes"

func looksLikeEventStreamBody(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	if bytes.HasPrefix(trimmed, []byte("data:")) || bytes.HasPrefix(trimmed, []byte("event:")) {
		return true
	}
	// SSE frames typically appear at the start of a line.
	return bytes.Contains(trimmed, []byte("\ndata:")) ||
		bytes.Contains(trimmed, []byte("\nevent:")) ||
		bytes.Contains(trimmed, []byte("\r\ndata:")) ||
		bytes.Contains(trimmed, []byte("\r\nevent:"))
}
