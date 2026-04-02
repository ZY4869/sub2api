package service

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func collectKiroResponse(body io.Reader) (*kiroCollectedResponse, error) {
	reader := bufio.NewReaderSize(body, 1024*1024)
	collected := &kiroCollectedResponse{}
	processedIDs := make(map[string]bool)
	var currentTool *kiroToolUseState
	var content strings.Builder

	for {
		msg, err := readKiroEventStreamMessage(reader)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			break
		}
		if len(msg.Payload) == 0 {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal(msg.Payload, &event); err != nil {
			continue
		}
		if eventErr := detectKiroEventError(event); eventErr != nil {
			return nil, eventErr
		}

		collected.StopReason = firstKiroStopReason(collected.StopReason, msg.EventType, event)
		updateKiroUsageFromEvent(msg.EventType, event, &collected.Usage)

		switch msg.EventType {
		case "assistantResponseEvent":
			if delta := extractKiroAssistantContent(event); delta != "" {
				_, _ = content.WriteString(delta)
			}
			collected.ToolUses = append(collected.ToolUses, extractKiroAssistantToolUses(event, processedIDs)...)
		case "reasoningContentEvent":
			if thinking := extractKiroReasoningContent(event); thinking != "" {
				_, _ = content.WriteString(kiroThinkingStartTag)
				_, _ = content.WriteString(thinking)
				_, _ = content.WriteString(kiroThinkingEndTag)
			}
		case "toolUseEvent":
			toolUses, next := processKiroToolUseEvent(event, currentTool, processedIDs)
			currentTool = next
			collected.ToolUses = append(collected.ToolUses, toolUses...)
		}
	}

	collected.Content = content.String()
	collected.StopReason = normalizeKiroStopReason(collected.StopReason, len(collected.ToolUses) > 0)
	return collected, nil
}

func streamKiroToClaude(body io.Reader, writer io.Writer, modelID string) {
	reader := bufio.NewReaderSize(body, 1024*1024)
	writeSSE(writer, "message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            "msg_" + randomHex(12),
			"type":          "message",
			"role":          "assistant",
			"model":         modelID,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	})

	processedIDs := make(map[string]bool)
	var currentTool *kiroToolUseState
	var usage ClaudeUsage
	var hasToolUse bool
	stopReason := ""
	state := &kiroEmitState{openTextIndex: -1, openThinkingIndex: -1}

	for {
		msg, err := readKiroEventStreamMessage(reader)
		if err != nil {
			writeSSE(writer, "error", map[string]any{
				"type": "error",
				"error": map[string]any{
					"message": err.Error(),
				},
			})
			return
		}
		if msg == nil {
			break
		}
		if len(msg.Payload) == 0 {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal(msg.Payload, &event); err != nil {
			continue
		}
		if eventErr := detectKiroEventError(event); eventErr != nil {
			writeSSE(writer, "error", map[string]any{
				"type": "error",
				"error": map[string]any{
					"message": eventErr.Error(),
				},
			})
			return
		}

		stopReason = firstKiroStopReason(stopReason, msg.EventType, event)
		updateKiroUsageFromEvent(msg.EventType, event, &usage)

		switch msg.EventType {
		case "assistantResponseEvent":
			state.consumeContent(writer, extractKiroAssistantContent(event))
			for _, toolUse := range extractKiroAssistantToolUses(event, processedIDs) {
				hasToolUse = true
				state.emitToolUse(writer, toolUse)
			}
		case "reasoningContentEvent":
			state.emitThinking(writer, extractKiroReasoningContent(event))
		case "toolUseEvent":
			toolUses, next := processKiroToolUseEvent(event, currentTool, processedIDs)
			currentTool = next
			for _, toolUse := range toolUses {
				hasToolUse = true
				state.emitToolUse(writer, toolUse)
			}
		}
	}

	state.flushPending(writer)
	state.closeText(writer)
	state.closeThinking(writer)

	writeSSE(writer, "message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   normalizeKiroStopReason(stopReason, hasToolUse),
			"stop_sequence": nil,
		},
		"usage": map[string]any{
			"input_tokens":  usage.InputTokens,
			"output_tokens": usage.OutputTokens,
		},
	})
	writeSSE(writer, "message_stop", map[string]any{"type": "message_stop"})
}

func buildClaudeResponseFromKiro(collected *kiroCollectedResponse, modelID string) []byte {
	blocks := extractKiroContentBlocks(collected.Content)
	for _, toolUse := range collected.ToolUses {
		blocks = append(blocks, map[string]any{
			"type":  "tool_use",
			"id":    toolUse.ID,
			"name":  toolUse.Name,
			"input": toolUse.Input,
		})
	}
	if len(blocks) == 0 {
		blocks = []map[string]any{{"type": "text", "text": ""}}
	}
	data, _ := json.Marshal(map[string]any{
		"id":            "msg_" + randomHex(12),
		"type":          "message",
		"role":          "assistant",
		"model":         modelID,
		"content":       blocks,
		"stop_reason":   normalizeKiroStopReason(collected.StopReason, len(collected.ToolUses) > 0),
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  collected.Usage.InputTokens,
			"output_tokens": collected.Usage.OutputTokens,
		},
	})
	return data
}

func extractKiroContentBlocks(content string) []map[string]any {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	var blocks []map[string]any
	for len(content) > 0 {
		start := strings.Index(content, kiroThinkingStartTag)
		if start < 0 {
			blocks = append(blocks, map[string]any{"type": "text", "text": content})
			break
		}
		if start > 0 {
			blocks = append(blocks, map[string]any{"type": "text", "text": content[:start]})
		}
		content = content[start+len(kiroThinkingStartTag):]
		end := strings.Index(content, kiroThinkingEndTag)
		if end < 0 {
			blocks = append(blocks, map[string]any{"type": "thinking", "thinking": content, "signature": "kiro-runtime"})
			break
		}
		thinking := content[:end]
		if strings.TrimSpace(thinking) != "" {
			blocks = append(blocks, map[string]any{"type": "thinking", "thinking": thinking, "signature": "kiro-runtime"})
		}
		content = content[end+len(kiroThinkingEndTag):]
	}
	return blocks
}

func readKiroEventStreamMessage(reader *bufio.Reader) (*kiroEventStreamMessage, error) {
	prelude := make([]byte, 12)
	if _, err := io.ReadFull(reader, prelude); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("kiro event stream prelude: %w", err)
	}

	totalLength := binary.BigEndian.Uint32(prelude[0:4])
	headersLength := binary.BigEndian.Uint32(prelude[4:8])
	if totalLength < kiroEventStreamMinFrameSize {
		return nil, fmt.Errorf("kiro event stream frame too small: %d", totalLength)
	}
	if totalLength > kiroEventStreamMaxMessageSize {
		return nil, fmt.Errorf("kiro event stream frame too large: %d", totalLength)
	}
	if headersLength > totalLength-16 {
		return nil, fmt.Errorf("kiro event stream header length invalid: %d", headersLength)
	}

	remaining := make([]byte, totalLength-12)
	if _, err := io.ReadFull(reader, remaining); err != nil {
		return nil, fmt.Errorf("kiro event stream body: %w", err)
	}

	eventType := extractKiroEventType(remaining[:headersLength])
	payloadStart := headersLength
	payloadEnd := uint32(len(remaining)) - 4
	if payloadStart >= payloadEnd {
		return &kiroEventStreamMessage{EventType: eventType}, nil
	}

	return &kiroEventStreamMessage{
		EventType: eventType,
		Payload:   remaining[payloadStart:payloadEnd],
	}, nil
}

func extractKiroEventType(headers []byte) string {
	offset := 0
	for offset < len(headers) {
		nameLen := int(headers[offset])
		offset++
		if offset+nameLen > len(headers) {
			return ""
		}
		name := string(headers[offset : offset+nameLen])
		offset += nameLen
		if offset >= len(headers) {
			return ""
		}
		valueType := headers[offset]
		offset++
		if valueType == 7 {
			if offset+2 > len(headers) {
				return ""
			}
			valueLen := int(binary.BigEndian.Uint16(headers[offset : offset+2]))
			offset += 2
			if offset+valueLen > len(headers) {
				return ""
			}
			value := string(headers[offset : offset+valueLen])
			offset += valueLen
			if name == ":event-type" {
				return value
			}
			continue
		}
		next, ok := skipKiroHeaderValue(headers, offset, valueType)
		if !ok {
			return ""
		}
		offset = next
	}
	return ""
}

func skipKiroHeaderValue(headers []byte, offset int, valueType byte) (int, bool) {
	switch valueType {
	case 0, 1:
		return offset, true
	case 2:
		if offset+1 > len(headers) {
			return offset, false
		}
		return offset + 1, true
	case 3:
		if offset+2 > len(headers) {
			return offset, false
		}
		return offset + 2, true
	case 4:
		if offset+4 > len(headers) {
			return offset, false
		}
		return offset + 4, true
	case 5, 8:
		if offset+8 > len(headers) {
			return offset, false
		}
		return offset + 8, true
	case 6:
		if offset+2 > len(headers) {
			return offset, false
		}
		valueLen := int(binary.BigEndian.Uint16(headers[offset : offset+2]))
		offset += 2
		if offset+valueLen > len(headers) {
			return offset, false
		}
		return offset + valueLen, true
	case 9:
		if offset+16 > len(headers) {
			return offset, false
		}
		return offset + 16, true
	default:
		return offset, false
	}
}

func detectKiroEventError(event map[string]any) error {
	if event == nil {
		return nil
	}
	if errType := strings.TrimSpace(stringValue(event["_type"])); errType != "" {
		return fmt.Errorf("%s: %s", errType, strings.TrimSpace(stringValue(event["message"])))
	}
	if eventType := strings.TrimSpace(stringValue(event["type"])); eventType == "error" || eventType == "exception" {
		return fmt.Errorf("%s", strings.TrimSpace(stringValue(event["message"])))
	}
	return nil
}

func firstKiroStopReason(current, eventType string, event map[string]any) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	if nested, ok := event[eventType].(map[string]any); ok {
		if stop := strings.TrimSpace(stringValue(nested["stop_reason"])); stop != "" {
			return stop
		}
		if stop := strings.TrimSpace(stringValue(nested["stopReason"])); stop != "" {
			return stop
		}
	}
	if stop := strings.TrimSpace(stringValue(event["stop_reason"])); stop != "" {
		return stop
	}
	return strings.TrimSpace(stringValue(event["stopReason"]))
}

func updateKiroUsageFromEvent(eventType string, event map[string]any, usage *ClaudeUsage) {
	if usage == nil {
		return
	}
	readUsage := func(container map[string]any) {
		if container == nil {
			return
		}
		if tokenUsage, ok := container["tokenUsage"].(map[string]any); ok {
			if input, ok := intValue(tokenUsage["uncachedInputTokens"]); ok && input > 0 {
				usage.InputTokens = input
			}
			if cacheRead, ok := intValue(tokenUsage["cacheReadInputTokens"]); ok && cacheRead > 0 {
				usage.CacheReadInputTokens = cacheRead
				if usage.InputTokens > 0 {
					usage.InputTokens += cacheRead
				}
			}
			if output, ok := intValue(tokenUsage["outputTokens"]); ok && output > 0 {
				usage.OutputTokens = output
			}
		}
		if input, ok := intValue(container["inputTokens"]); ok && input > 0 && usage.InputTokens == 0 {
			usage.InputTokens = input
		}
		if output, ok := intValue(container["outputTokens"]); ok && output > 0 {
			usage.OutputTokens = output
		}
	}

	readUsage(event)
	if nested, ok := event[eventType].(map[string]any); ok {
		readUsage(nested)
	}
	if usageObj, ok := event["usage"].(map[string]any); ok {
		if input, ok := intValue(usageObj["input_tokens"]); ok && input > 0 {
			usage.InputTokens = input
		}
		if output, ok := intValue(usageObj["output_tokens"]); ok && output > 0 {
			usage.OutputTokens = output
		}
	}
}

func extractKiroAssistantContent(event map[string]any) string {
	if nested, ok := event["assistantResponseEvent"].(map[string]any); ok {
		if content := strings.TrimSpace(stringValue(nested["content"])); content != "" {
			return content
		}
	}
	return strings.TrimSpace(stringValue(event["content"]))
}

func extractKiroReasoningContent(event map[string]any) string {
	if nested, ok := event["reasoningContentEvent"].(map[string]any); ok {
		return strings.TrimSpace(stringValue(nested["text"]))
	}
	return strings.TrimSpace(stringValue(event["text"]))
}

func extractKiroAssistantToolUses(event map[string]any, processed map[string]bool) []kiroToolUse {
	var rawUses []any
	if nested, ok := event["assistantResponseEvent"].(map[string]any); ok {
		if uses, ok := nested["toolUses"].([]any); ok {
			rawUses = append(rawUses, uses...)
		}
	}
	if uses, ok := event["toolUses"].([]any); ok {
		rawUses = append(rawUses, uses...)
	}
	var toolUses []kiroToolUse
	for _, raw := range rawUses {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := strings.TrimSpace(stringValue(item["toolUseId"]))
		name := strings.TrimSpace(stringValue(item["name"]))
		if id == "" || name == "" || processed[id] {
			continue
		}
		processed[id] = true
		toolUses = append(toolUses, kiroToolUse{
			ID:    id,
			Name:  name,
			Input: mapValue(item["input"]),
		})
	}
	return toolUses
}

func processKiroToolUseEvent(event map[string]any, current *kiroToolUseState, processed map[string]bool) ([]kiroToolUse, *kiroToolUseState) {
	payload := event
	if nested, ok := event["toolUseEvent"].(map[string]any); ok {
		payload = nested
	}

	id := strings.TrimSpace(stringValue(payload["toolUseId"]))
	name := strings.TrimSpace(stringValue(payload["name"]))
	stop := boolValue(payload["stop"])

	if current == nil && id != "" && name != "" {
		current = &kiroToolUseState{ID: id, Name: name}
	}
	if current == nil {
		return nil, nil
	}
	if current.ID != "" && id != "" && current.ID != id {
		current = &kiroToolUseState{ID: id, Name: name}
	}

	switch input := payload["input"].(type) {
	case string:
		_, _ = current.Parts.WriteString(input)
	case map[string]any:
		bytes, _ := json.Marshal(input)
		current.Parts.Reset()
		_, _ = current.Parts.Write(bytes)
		stop = true
	}

	if !stop || processed[current.ID] || current.ID == "" || current.Name == "" {
		return nil, current
	}

	input := map[string]any{}
	if raw := strings.TrimSpace(current.Parts.String()); raw != "" {
		_ = json.Unmarshal([]byte(repairKiroJSON(raw)), &input)
	}
	processed[current.ID] = true
	return []kiroToolUse{{
		ID:    current.ID,
		Name:  current.Name,
		Input: input,
	}}, nil
}

func repairKiroJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	open := strings.Count(raw, "{")
	closeCount := strings.Count(raw, "}")
	if closeCount < open {
		raw += strings.Repeat("}", open-closeCount)
	}
	return raw
}

func (s *kiroEmitState) consumeContent(writer io.Writer, delta string) {
	if strings.TrimSpace(delta) == "" {
		return
	}
	s.pendingContent += delta
	for len(s.pendingContent) > 0 {
		if s.inThinking {
			idx := strings.Index(s.pendingContent, kiroThinkingEndTag)
			if idx >= 0 {
				s.emitThinking(writer, s.pendingContent[:idx])
				s.pendingContent = s.pendingContent[idx+len(kiroThinkingEndTag):]
				s.inThinking = false
				s.closeThinking(writer)
				continue
			}
			emit, pending := splitKiroPotentialTagSuffix(s.pendingContent, kiroThinkingEndTag)
			s.emitThinking(writer, emit)
			s.pendingContent = pending
			return
		}

		idx := strings.Index(s.pendingContent, kiroThinkingStartTag)
		if idx >= 0 {
			s.emitText(writer, s.pendingContent[:idx])
			s.pendingContent = s.pendingContent[idx+len(kiroThinkingStartTag):]
			s.closeText(writer)
			s.inThinking = true
			continue
		}
		emit, pending := splitKiroPotentialTagSuffix(s.pendingContent, kiroThinkingStartTag)
		s.emitText(writer, emit)
		s.pendingContent = pending
		return
	}
}

func (s *kiroEmitState) flushPending(writer io.Writer) {
	if s.pendingContent == "" {
		return
	}
	if s.inThinking {
		s.emitThinking(writer, s.pendingContent)
	} else {
		s.emitText(writer, s.pendingContent)
	}
	s.pendingContent = ""
}

func splitKiroPotentialTagSuffix(content, tag string) (string, string) {
	maxCheck := len(tag) - 1
	if len(content) < maxCheck {
		maxCheck = len(content)
	}
	for i := maxCheck; i > 0; i-- {
		if strings.HasSuffix(content, tag[:i]) {
			return content[:len(content)-i], content[len(content)-i:]
		}
	}
	return content, ""
}

func (s *kiroEmitState) emitText(writer io.Writer, text string) {
	if text == "" {
		return
	}
	s.closeThinking(writer)
	if s.openTextIndex < 0 {
		s.openTextIndex = s.nextIndex
		s.nextIndex++
		writeSSE(writer, "content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": s.openTextIndex,
			"content_block": map[string]any{
				"type": "text",
				"text": "",
			},
		})
	}
	writeSSE(writer, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": s.openTextIndex,
		"delta": map[string]any{
			"type": "text_delta",
			"text": text,
		},
	})
}

func (s *kiroEmitState) emitThinking(writer io.Writer, thinking string) {
	if thinking == "" {
		return
	}
	s.closeText(writer)
	if s.openThinkingIndex < 0 {
		s.openThinkingIndex = s.nextIndex
		s.nextIndex++
		writeSSE(writer, "content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": s.openThinkingIndex,
			"content_block": map[string]any{
				"type":      "thinking",
				"thinking":  "",
				"signature": "kiro-runtime",
			},
		})
	}
	writeSSE(writer, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": s.openThinkingIndex,
		"delta": map[string]any{
			"type":     "thinking_delta",
			"thinking": thinking,
		},
	})
}

func (s *kiroEmitState) emitToolUse(writer io.Writer, toolUse kiroToolUse) {
	s.flushPending(writer)
	s.closeText(writer)
	s.closeThinking(writer)
	index := s.nextIndex
	s.nextIndex++
	writeSSE(writer, "content_block_start", map[string]any{
		"type":  "content_block_start",
		"index": index,
		"content_block": map[string]any{
			"type":  "tool_use",
			"id":    toolUse.ID,
			"name":  toolUse.Name,
			"input": map[string]any{},
		},
	})
	if len(toolUse.Input) > 0 {
		input, _ := json.Marshal(toolUse.Input)
		writeSSE(writer, "content_block_delta", map[string]any{
			"type":  "content_block_delta",
			"index": index,
			"delta": map[string]any{
				"type":         "input_json_delta",
				"partial_json": string(input),
			},
		})
	}
	writeSSE(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": index})
}

func (s *kiroEmitState) closeText(writer io.Writer) {
	if s.openTextIndex < 0 {
		return
	}
	writeSSE(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": s.openTextIndex})
	s.openTextIndex = -1
}

func (s *kiroEmitState) closeThinking(writer io.Writer) {
	if s.openThinkingIndex < 0 {
		return
	}
	writeSSE(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": s.openThinkingIndex})
	s.openThinkingIndex = -1
}
