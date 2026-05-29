package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

func (s *GeminiMessagesCompatService) buildGeminiOAuthCompatUpstreamRequest(
	ctx context.Context,
	account *Account,
	mappedModel string,
	action string,
	useUpstreamStream bool,
	geminiReq []byte,
	projectID string,
	shouldMimicGeminiCLI bool,
) (*http.Request, string, error) {
	accessToken, err := s.geminiOAuthAccessToken(ctx, account)
	if err != nil {
		return nil, "", err
	}
	if account.IsGeminiVertexAI() {
		mappedModel = normalizeVertexUpstreamModelID(mappedModel)
		baseURL := account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, "", err
		}
		actionPath, err := account.GeminiVertexModelActionPath(mappedModel, action)
		if err != nil {
			return nil, "", err
		}
		fullURL := strings.TrimRight(normalizedBaseURL, "/") + actionPath
		if useUpstreamStream {
			fullURL += "?alt=sse"
		}
		return newGeminiOAuthJSONRequest(ctx, fullURL, accessToken, geminiReq, shouldMimicGeminiCLI)
	}
	if projectID != "" {
		baseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
		if err != nil {
			return nil, "", err
		}
		fullURL := fmt.Sprintf("%s/v1internal:%s", strings.TrimRight(baseURL, "/"), action)
		if useUpstreamStream {
			fullURL += "?alt=sse"
		}
		wrappedBytes, err := wrapGeminiProjectRequest(geminiReq, mappedModel, projectID)
		if err != nil {
			return nil, "", err
		}
		return newGeminiOAuthJSONRequest(ctx, fullURL, accessToken, wrappedBytes, shouldMimicGeminiCLI)
	}
	baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, "", err
	}
	fullURL := fmt.Sprintf("%s/v1beta/models/%s:%s", strings.TrimRight(normalizedBaseURL, "/"), mappedModel, action)
	if useUpstreamStream {
		fullURL += "?alt=sse"
	}
	return newGeminiOAuthJSONRequest(ctx, fullURL, accessToken, geminiReq, shouldMimicGeminiCLI)
}

func (s *GeminiMessagesCompatService) buildGeminiOAuthNativeUpstreamRequest(
	ctx context.Context,
	account *Account,
	mappedModel string,
	upstreamAction string,
	useUpstreamStream bool,
	body []byte,
	projectID string,
	forceAIStudio bool,
	shouldMimicGeminiCLI bool,
) (*http.Request, string, error) {
	accessToken, err := s.geminiOAuthAccessToken(ctx, account)
	if err != nil {
		return nil, "", err
	}
	if account.IsGeminiVertexAI() {
		baseURL := account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, "", err
		}
		actionPath, err := account.GeminiVertexModelActionPath(mappedModel, upstreamAction)
		if err != nil {
			return nil, "", err
		}
		fullURL := strings.TrimRight(normalizedBaseURL, "/") + actionPath
		if useUpstreamStream {
			fullURL += "?alt=sse"
		}
		restGeminiReq := normalizeGeminiRequestForAIStudio(body)
		return newGeminiOAuthJSONRequest(ctx, fullURL, accessToken, restGeminiReq, shouldMimicGeminiCLI)
	}
	if projectID != "" && !forceAIStudio {
		baseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
		if err != nil {
			return nil, "", err
		}
		fullURL := fmt.Sprintf("%s/v1internal:%s", strings.TrimRight(baseURL, "/"), upstreamAction)
		if useUpstreamStream {
			fullURL += "?alt=sse"
		}
		wrappedBytes, err := wrapGeminiProjectRequest(body, mappedModel, projectID)
		if err != nil {
			return nil, "", err
		}
		return newGeminiOAuthJSONRequest(ctx, fullURL, accessToken, wrappedBytes, shouldMimicGeminiCLI)
	}
	baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, "", err
	}
	fullURL := fmt.Sprintf("%s/v1beta/models/%s:%s", strings.TrimRight(normalizedBaseURL, "/"), mappedModel, upstreamAction)
	if useUpstreamStream {
		fullURL += "?alt=sse"
	}
	restGeminiReq := normalizeGeminiRequestForAIStudio(body)
	return newGeminiOAuthJSONRequest(ctx, fullURL, accessToken, restGeminiReq, shouldMimicGeminiCLI)
}

func (s *GeminiMessagesCompatService) geminiOAuthAccessToken(ctx context.Context, account *Account) (string, error) {
	if s.tokenProvider == nil {
		return "", errors.New("gemini token provider not configured")
	}
	return s.tokenProvider.GetAccessToken(ctx, account)
}

func wrapGeminiProjectRequest(body []byte, mappedModel string, projectID string) ([]byte, error) {
	wrapped := map[string]any{"model": mappedModel, "project": projectID}
	var inner any
	if err := json.Unmarshal(body, &inner); err != nil {
		return nil, fmt.Errorf("failed to parse gemini request: %w", err)
	}
	wrapped["request"] = inner
	wrappedBytes, _ := json.Marshal(wrapped)
	return wrappedBytes, nil
}

func newGeminiOAuthJSONRequest(ctx context.Context, fullURL string, accessToken string, body []byte, shouldMimicGeminiCLI bool) (*http.Request, string, error) {
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
	if shouldMimicGeminiCLI {
		upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
	}
	return upstreamReq, "x-request-id", nil
}
