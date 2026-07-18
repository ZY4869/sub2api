package securityaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrNoPromptText = errors.New("prompt audit request contains no user text")

type promptSegment struct {
	text string
	user bool
}

func ExtractPromptSnapshot(req Request) (PromptSnapshot, error) {
	var document any
	if err := json.Unmarshal(req.Body, &document); err != nil {
		return PromptSnapshot{}, errors.New("prompt audit request JSON is invalid")
	}
	segments := normalizeSegmentsLatestUserFirst(extractProtocolSegments(req.Protocol, document))
	if len(segments) == 0 {
		return PromptSnapshot{}, ErrNoPromptText
	}
	fullPrompt := strings.Join(segments, "\n\n")
	digest := sha256.Sum256([]byte(fullPrompt))
	stage := strings.TrimSpace(req.Stage)
	if stage == "" {
		stage = "http"
	}
	return PromptSnapshot{
		RequestID: req.RequestID, UserID: cloneInt64(req.UserID),
		UsernameSnapshot: req.Username, UserEmailSnapshot: req.UserEmail,
		APIKeyID: cloneInt64(req.APIKeyID), APIKeyNameSnapshot: req.APIKeyName,
		GroupID: cloneInt64(req.GroupID), GroupName: req.GroupName,
		Provider: req.Provider, Endpoint: req.Endpoint, Protocol: req.Protocol,
		Model: req.Model, PromptHash: hex.EncodeToString(digest[:]),
		RedactedPreview: BuildPromptPreview(fullPrompt, DefaultPromptPreviewRunes),
		FullPrompt:      BuildFullPrompt(fullPrompt, DefaultFullPromptRunes),
		PromptLength:    utf8.RuneCountInString(fullPrompt),
		MessageCount:    len(segments), Stage: stage, ScanText: fullPrompt,
	}, nil
}

func extractProtocolSegments(protocol string, document any) []promptSegment {
	root, _ := document.(map[string]any)
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case ProtocolOpenAIChat:
		return extractMessages(root["messages"], true)
	case ProtocolOpenAICompletions:
		return extractOpenAICompletions(root)
	case ProtocolOpenAIEmbeddings:
		return extractOpenAIEmbeddings(root)
	case ProtocolOpenAIAlphaSearch:
		return extractOpenAIAlphaSearch(root)
	case ProtocolAnthropic:
		return append(extractSystem(root["system"]), extractMessages(root["messages"], true)...)
	case ProtocolGemini:
		return extractGeminiRoot(root)
	case ProtocolOpenAIResponses, ProtocolResponsesWS:
		return append(extractSystem(root["instructions"]), extractResponses(root)...)
	case ProtocolOpenAIImages, ProtocolGrokMedia:
		return userSegments(extractMediaPrompts(root))
	default:
		if segments := extractMessages(root["messages"], true); len(segments) > 0 {
			return segments
		}
		if segments := extractOpenAIAlphaSearch(root); len(segments) > 0 {
			return segments
		}
		if segments := extractOpenAIEmbeddings(root); len(segments) > 0 {
			return segments
		}
		if segments := extractOpenAICompletions(root); len(segments) > 0 {
			return segments
		}
		if segments := extractGeminiRoot(root); len(segments) > 0 {
			return segments
		}
		return userSegments(extractMediaPrompts(root))
	}
}
