package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type geminiStreamChunk struct {
	response map[string]any
	raw      []byte
}

type geminiClaudeStreamEmitter struct {
	writer  io.Writer
	flusher interface{ Flush() }

	model     string
	startTime time.Time
	messageID string

	started bool

	nextIndex int

	openTextIndex     int
	seenText          string
	openThinkingIndex int
	openThinkingSig   string
	seenThinking      string
	openToolIndex     int
	openToolID        string
	openToolName      string
	seenToolJSON      string
	toolSequence      int

	usage               ClaudeUsage
	finishReason        string
	sawToolUse          bool
	firstTokenMs        *int
	responseID          string
	resolvedServiceTier *string
}

func newGeminiClaudeStreamEmitter(writer io.Writer, flusher interface{ Flush() }, startTime time.Time, model string) *geminiClaudeStreamEmitter {
	return &geminiClaudeStreamEmitter{
		writer:            writer,
		flusher:           flusher,
		model:             model,
		startTime:         startTime,
		messageID:         "msg_" + randomHex(12),
		openTextIndex:     -1,
		openThinkingIndex: -1,
		openToolIndex:     -1,
	}
}

func (e *geminiClaudeStreamEmitter) consumeResponse(geminiResp map[string]any, raw []byte) {
	analysis := analyzeGeminiResponse(geminiResp, raw)
	if analysis.Usage != nil {
		e.usage = *analysis.Usage
	}
	if resolvedServiceTier := extractGeminiResolvedServiceTierFromResponse(raw, nil); resolvedServiceTier != nil {
		e.resolvedServiceTier = resolvedServiceTier
	}
	if strings.TrimSpace(analysis.ResponseID) != "" {
		e.responseID = analysis.ResponseID
	}
	if strings.TrimSpace(analysis.FinishReason) != "" {
		e.finishReason = analysis.FinishReason
	}
	for _, part := range analysis.Parts {
		e.consumePart(part)
	}
}

func (e *geminiClaudeStreamEmitter) consumePart(part map[string]any) {
	if part == nil {
		return
	}
	if functionCall, ok := part["functionCall"].(map[string]any); ok && functionCall != nil {
		e.emitToolUse(functionCall)
		return
	}
	text := stringValueFromAny(part["text"])
	if strings.TrimSpace(text) != "" {
		if thought, _ := part["thought"].(bool); thought {
			e.emitThinking(text, stringValueFromAny(part["thoughtSignature"]))
			return
		}
		e.emitText(text)
		return
	}
	if inlineData, ok := part["inlineData"].(map[string]any); ok && inlineData != nil {
		mimeType := firstNonEmptyString(
			stringValueFromAny(inlineData["mimeType"]),
			stringValueFromAny(inlineData["mime_type"]),
		)
		if mimeType == "" {
			mimeType = "image/*"
		}
		e.emitText(fmt.Sprintf("[Gemini returned inline %s data]", mimeType))
	}
}

func (e *geminiClaudeStreamEmitter) ensureStarted() {
	if e.started {
		return
	}
	messageStart := map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            e.messageID,
			"type":          "message",
			"role":          "assistant",
			"model":         e.model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	}
	writeSSE(e.writer, "message_start", messageStart)
	e.flush()
	e.started = true
}

func (e *geminiClaudeStreamEmitter) noteFirstToken() {
	if e.firstTokenMs != nil {
		return
	}
	ms := int(time.Since(e.startTime).Milliseconds())
	e.firstTokenMs = &ms
}

func (e *geminiClaudeStreamEmitter) emitText(text string) {
	delta, newSeen := computeGeminiTextDelta(e.seenText, text)
	e.seenText = newSeen
	if delta == "" {
		return
	}
	e.ensureStarted()
	e.closeThinking()
	e.closeTool()
	if e.openTextIndex < 0 {
		e.openTextIndex = e.nextIndex
		e.nextIndex++
		writeSSE(e.writer, "content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": e.openTextIndex,
			"content_block": map[string]any{
				"type": "text",
				"text": "",
			},
		})
	}
	e.noteFirstToken()
	writeSSE(e.writer, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": e.openTextIndex,
		"delta": map[string]any{
			"type": "text_delta",
			"text": delta,
		},
	})
	e.flush()
}

func (e *geminiClaudeStreamEmitter) emitThinking(thinking string, signature string) {
	delta, newSeen := computeGeminiTextDelta(e.seenThinking, thinking)
	e.seenThinking = newSeen
	if delta == "" {
		return
	}
	e.ensureStarted()
	e.closeText()
	e.closeTool()
	if e.openThinkingIndex < 0 || (signature != "" && signature != e.openThinkingSig) {
		e.closeThinking()
		e.openThinkingIndex = e.nextIndex
		e.nextIndex++
		e.openThinkingSig = signature
		contentBlock := map[string]any{
			"type":     "thinking",
			"thinking": "",
		}
		if strings.TrimSpace(signature) != "" {
			contentBlock["signature"] = signature
		}
		writeSSE(e.writer, "content_block_start", map[string]any{
			"type":          "content_block_start",
			"index":         e.openThinkingIndex,
			"content_block": contentBlock,
		})
	}
	e.noteFirstToken()
	writeSSE(e.writer, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": e.openThinkingIndex,
		"delta": map[string]any{
			"type":     "thinking_delta",
			"thinking": delta,
		},
	})
	e.flush()
}

func (e *geminiClaudeStreamEmitter) emitToolUse(functionCall map[string]any) {
	name := strings.TrimSpace(stringValueFromAny(functionCall["name"]))
	if name == "" {
		name = "tool"
	}
	e.ensureStarted()
	e.closeText()
	e.closeThinking()
	toolID := buildGeminiToolUseID(functionCall, e.toolSequence+1)
	if e.openToolIndex >= 0 && (e.openToolID != toolID || e.openToolName != name) {
		e.closeTool()
	}
	if e.openToolIndex < 0 {
		e.toolSequence++
		e.openToolIndex = e.nextIndex
		e.nextIndex++
		e.openToolID = toolID
		e.openToolName = name
		e.seenToolJSON = ""
		e.sawToolUse = true
		writeSSE(e.writer, "content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": e.openToolIndex,
			"content_block": map[string]any{
				"type":  "tool_use",
				"id":    toolID,
				"name":  name,
				"input": map[string]any{},
			},
		})
	}
	argsJSON := "{}"
	switch args := functionCall["args"].(type) {
	case nil:
	case string:
		if strings.TrimSpace(args) != "" {
			argsJSON = args
		}
	default:
		if raw, err := json.Marshal(args); err == nil && len(raw) > 0 {
			argsJSON = string(raw)
		}
	}
	delta, newSeen := computeGeminiTextDelta(e.seenToolJSON, argsJSON)
	e.seenToolJSON = newSeen
	if delta == "" {
		return
	}
	e.noteFirstToken()
	writeSSE(e.writer, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": e.openToolIndex,
		"delta": map[string]any{
			"type":         "input_json_delta",
			"partial_json": delta,
		},
	})
	e.flush()
}

func (e *geminiClaudeStreamEmitter) finalize() *geminiStreamResult {
	if e.openTextIndex >= 0 {
		e.closeText()
	}
	if e.openThinkingIndex >= 0 {
		e.closeThinking()
	}
	if e.openToolIndex >= 0 {
		e.closeTool()
	}
	if !e.started {
		return &geminiStreamResult{usage: &e.usage, firstTokenMs: e.firstTokenMs, responseID: e.responseID, resolvedServiceTier: e.resolvedServiceTier}
	}
	stopReason := mapGeminiFinishReasonToClaudeStopReason(e.finishReason)
	if e.sawToolUse {
		stopReason = "tool_use"
	}
	usage := map[string]any{"output_tokens": e.usage.OutputTokens}
	if e.usage.InputTokens > 0 {
		usage["input_tokens"] = e.usage.InputTokens
	}
	writeSSE(e.writer, "message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": usage,
	})
	writeSSE(e.writer, "message_stop", map[string]any{"type": "message_stop"})
	e.flush()
	return &geminiStreamResult{usage: &e.usage, firstTokenMs: e.firstTokenMs, responseID: e.responseID, resolvedServiceTier: e.resolvedServiceTier}
}

func (e *geminiClaudeStreamEmitter) closeText() {
	if e.openTextIndex < 0 {
		return
	}
	writeSSE(e.writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": e.openTextIndex})
	e.openTextIndex = -1
	e.seenText = ""
}

func (e *geminiClaudeStreamEmitter) closeThinking() {
	if e.openThinkingIndex < 0 {
		return
	}
	writeSSE(e.writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": e.openThinkingIndex})
	e.openThinkingIndex = -1
	e.openThinkingSig = ""
	e.seenThinking = ""
}

func (e *geminiClaudeStreamEmitter) closeTool() {
	if e.openToolIndex < 0 {
		return
	}
	writeSSE(e.writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": e.openToolIndex})
	e.openToolIndex = -1
	e.openToolID = ""
	e.openToolName = ""
	e.seenToolJSON = ""
}

func (e *geminiClaudeStreamEmitter) flush() {
	if e.flusher != nil {
		e.flusher.Flush()
	}
}

func readNextGeminiStreamChunk(reader *bufio.Reader, isOAuth bool) (*geminiStreamChunk, bool, error) {
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(trimmed, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				switch payload {
				case "":
				case "[DONE]":
					return nil, true, nil
				default:
					raw := []byte(payload)
					if isOAuth {
						if inner, unwrapErr := unwrapGeminiResponse(raw); unwrapErr == nil {
							raw = inner
						}
					}
					var parsed map[string]any
					if json.Unmarshal(raw, &parsed) == nil && parsed != nil {
						return &geminiStreamChunk{response: parsed, raw: raw}, false, nil
					}
				}
			}
		}
		if errors.Is(err, io.EOF) {
			return nil, true, nil
		}
		if err != nil {
			return nil, false, err
		}
	}
}
