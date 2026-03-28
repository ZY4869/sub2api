package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type accountModelImportMixedProbeUpstreamStub struct {
	lastRequests []string
}

func (s *accountModelImportMixedProbeUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return s.DoWithTLS(req, proxyURL, accountID, accountConcurrency, nil)
}

func (s *accountModelImportMixedProbeUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, tlsProfile *TLSFingerprintProfile) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	s.lastRequests = append(s.lastRequests, req.URL.String())

	body := ""
	switch {
	case strings.HasSuffix(req.URL.Path, "/v1/models"):
		body = `{"data":[{"id":"gpt-4.1"},{"id":"shared-model"}]}`
	case strings.HasSuffix(req.URL.Path, "/v1beta/models"):
		body = `{"models":[{"name":"models/gemini-2.5-pro"},{"name":"models/shared-model"}]}`
	default:
		return nil, errors.New("unexpected path: " + req.URL.Path)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestProbeAccountModels_MixedProtocolGatewayMergesModelsAndKeepsSourceProtocol(t *testing.T) {
	upstream := &accountModelImportMixedProbeUpstreamStub{}
	geminiCompatService := newTestGeminiCompatService(upstream)
	svc := NewAccountModelImportService(nil, geminiCompatService, upstream, nil)

	account := &Account{
		ID:       3001,
		Platform: PlatformProtocolGateway,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "gateway-key",
			"base_url": "http://gateway.local.test",
		},
		Extra: map[string]any{
			"gateway_protocol":           GatewayProtocolMixed,
			"gateway_accepted_protocols": []any{"openai", "gemini"},
		},
	}

	result, err := svc.ProbeAccountModels(context.Background(), account)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, accountModelProbeSourceUpstream, result.ProbeSource)
	require.Equal(t, []string{"gemini-2.5-pro", "gpt-4.1", "shared-model"}, result.DetectedModels)
	require.Len(t, result.Models, 3)
	require.ElementsMatch(t, []string{
		"http://gateway.local.test/v1/models",
		"http://gateway.local.test/v1beta/models",
	}, upstream.lastRequests)

	detailsByID := make(map[string]AccountModelProbeModel, len(result.Models))
	for _, model := range result.Models {
		detailsByID[model.ID] = model
	}

	require.Equal(t, PlatformOpenAI, detailsByID["gpt-4.1"].SourceProtocol)
	require.Equal(t, PlatformGemini, detailsByID["gemini-2.5-pro"].SourceProtocol)
	require.Equal(t, PlatformOpenAI, detailsByID["shared-model"].SourceProtocol)
}
