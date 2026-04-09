package service

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

const (
	compatRuntimePolicyAnthropicMessagesToResponses        = "anthropic_messages_to_responses"
	compatRuntimePolicyChatCompletionsToResponses          = "chat_completions_to_responses"
	compatRuntimePolicyAnthropicMessagesToGeminiGenContent = "anthropic_messages_to_gemini_generate_content"
)

type CompatRuntimeRegistryEntry struct {
	PolicyID         string                                 `json:"policy_id"`
	SourceProtocol   string                                 `json:"source_protocol"`
	TargetProtocol   string                                 `json:"target_protocol"`
	PolicyDefinition apicompat.CompatPolicy                 `json:"policy_definition"`
	Converter        func(input any) (any, error)           `json:"-"`
	ErrorMapper      func(err error) *apicompat.CompatError `json:"-"`
}

var compatRuntimeRegistry = []CompatRuntimeRegistryEntry{
	{
		PolicyID:         compatRuntimePolicyAnthropicMessagesToResponses,
		SourceProtocol:   "anthropic",
		TargetProtocol:   "responses",
		PolicyDefinition: apicompat.AnthropicMessagesToResponsesCompatPolicy(),
		Converter: func(input any) (any, error) {
			req, ok := input.(*apicompat.AnthropicRequest)
			if !ok || req == nil {
				return nil, compatRuntimeRegistryMissingError(compatRuntimePolicyAnthropicMessagesToResponses)
			}
			return apicompat.AnthropicToResponses(req)
		},
		ErrorMapper: defaultCompatRuntimeErrorMapper,
	},
	{
		PolicyID:         compatRuntimePolicyChatCompletionsToResponses,
		SourceProtocol:   "openai_chat_completions",
		TargetProtocol:   "responses",
		PolicyDefinition: apicompat.ChatCompletionsToResponsesCompatPolicy(),
		Converter: func(input any) (any, error) {
			req, ok := input.(*apicompat.ChatCompletionsRequest)
			if !ok || req == nil {
				return nil, compatRuntimeRegistryMissingError(compatRuntimePolicyChatCompletionsToResponses)
			}
			return apicompat.ChatCompletionsToResponses(req)
		},
		ErrorMapper: defaultCompatRuntimeErrorMapper,
	},
	{
		PolicyID:         compatRuntimePolicyAnthropicMessagesToGeminiGenContent,
		SourceProtocol:   "anthropic",
		TargetProtocol:   "gemini_generate_content",
		PolicyDefinition: apicompat.AnthropicMessagesToGeminiGenerateContentCompatPolicy(),
		Converter: func(input any) (any, error) {
			request, ok := input.(*compatRuntimeGeminiRequest)
			if !ok || request == nil {
				return nil, compatRuntimeRegistryMissingError(compatRuntimePolicyAnthropicMessagesToGeminiGenContent)
			}
			return convertClaudeMessagesToGeminiGenerateContent(request.Body, request.Options...)
		},
		ErrorMapper: defaultCompatRuntimeErrorMapper,
	},
}

type compatRuntimeGeminiRequest struct {
	Body    []byte
	Options []geminiTransformOptions
}

func CompatRuntimeRegistryEntries() []CompatRuntimeRegistryEntry {
	return append([]CompatRuntimeRegistryEntry(nil), compatRuntimeRegistry...)
}

func LookupCompatRuntimeRegistryEntry(policyID string) (CompatRuntimeRegistryEntry, bool) {
	normalized := strings.TrimSpace(policyID)
	for _, entry := range compatRuntimeRegistry {
		if entry.PolicyID == normalized {
			return entry, true
		}
	}
	return CompatRuntimeRegistryEntry{}, false
}

func defaultCompatRuntimeErrorMapper(err error) *apicompat.CompatError {
	if compatErr, ok := apicompat.AsCompatError(err); ok {
		return compatErr
	}
	return nil
}

func compatRuntimeRegistryMissingError(policyID string) error {
	policyID = strings.TrimSpace(policyID)
	return apicompat.NewCompatError(
		apicompat.CompatReasonRuntimeRegistryMissing,
		"compat.runtime.registry_missing",
		fmt.Sprintf("compat conversion is not configured for policy %s", policyID),
	)
}

func requireCompatRuntimeRegistryEntry(policyID string) (CompatRuntimeRegistryEntry, error) {
	entry, ok := LookupCompatRuntimeRegistryEntry(policyID)
	if !ok {
		return CompatRuntimeRegistryEntry{}, compatRuntimeRegistryMissingError(policyID)
	}
	return entry, nil
}

func ConvertAnthropicMessagesToResponsesRuntime(req *apicompat.AnthropicRequest) (*apicompat.ResponsesRequest, CompatRuntimeRegistryEntry, error) {
	entry, err := requireCompatRuntimeRegistryEntry(compatRuntimePolicyAnthropicMessagesToResponses)
	if err != nil {
		return nil, CompatRuntimeRegistryEntry{}, err
	}
	output, err := entry.Converter(req)
	if err != nil {
		if compatErr := entry.ErrorMapper(err); compatErr != nil {
			return nil, entry, compatErr
		}
		return nil, entry, err
	}
	responsesReq, ok := output.(*apicompat.ResponsesRequest)
	if !ok || responsesReq == nil {
		return nil, entry, compatRuntimeRegistryMissingError(entry.PolicyID)
	}
	return responsesReq, entry, nil
}

func ConvertChatCompletionsToResponsesRuntime(req *apicompat.ChatCompletionsRequest) (*apicompat.ResponsesRequest, CompatRuntimeRegistryEntry, error) {
	entry, err := requireCompatRuntimeRegistryEntry(compatRuntimePolicyChatCompletionsToResponses)
	if err != nil {
		return nil, CompatRuntimeRegistryEntry{}, err
	}
	output, err := entry.Converter(req)
	if err != nil {
		if compatErr := entry.ErrorMapper(err); compatErr != nil {
			return nil, entry, compatErr
		}
		return nil, entry, err
	}
	responsesReq, ok := output.(*apicompat.ResponsesRequest)
	if !ok || responsesReq == nil {
		return nil, entry, compatRuntimeRegistryMissingError(entry.PolicyID)
	}
	return responsesReq, entry, nil
}

func ConvertAnthropicMessagesToGeminiGenerateContentRuntime(body []byte, options ...geminiTransformOptions) ([]byte, CompatRuntimeRegistryEntry, error) {
	entry, err := requireCompatRuntimeRegistryEntry(compatRuntimePolicyAnthropicMessagesToGeminiGenContent)
	if err != nil {
		return nil, CompatRuntimeRegistryEntry{}, err
	}
	output, err := entry.Converter(&compatRuntimeGeminiRequest{
		Body:    body,
		Options: options,
	})
	if err != nil {
		if compatErr := entry.ErrorMapper(err); compatErr != nil {
			return nil, entry, compatErr
		}
		return nil, entry, err
	}
	geminiBody, ok := output.([]byte)
	if !ok {
		return nil, entry, compatRuntimeRegistryMissingError(entry.PolicyID)
	}
	return geminiBody, entry, nil
}
