//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type kiroTestStreamEvent struct {
	EventType string
	Payload   string
}

func newKiroEventStreamHTTPResponse(status int, events ...kiroTestStreamEvent) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/vnd.amazon.eventstream"},
		},
		Body: io.NopCloser(bytes.NewReader(buildKiroEventStream(events...))),
	}
}

func buildKiroEventStream(events ...kiroTestStreamEvent) []byte {
	var buf bytes.Buffer
	for _, event := range events {
		buf.Write(buildKiroEventStreamFrame(event.EventType, []byte(event.Payload)))
	}
	return buf.Bytes()
}

func buildKiroEventStreamFrame(eventType string, payload []byte) []byte {
	headers := buildKiroEventStreamHeaders(eventType)
	totalLength := uint32(12 + len(headers) + len(payload) + 4)
	frame := make([]byte, 0, totalLength)
	prelude := make([]byte, 4)

	binary.BigEndian.PutUint32(prelude, totalLength)
	frame = append(frame, prelude...)
	binary.BigEndian.PutUint32(prelude, uint32(len(headers)))
	frame = append(frame, prelude...)
	frame = append(frame, 0, 0, 0, 0)
	frame = append(frame, headers...)
	frame = append(frame, payload...)
	frame = append(frame, 0, 0, 0, 0)
	return frame
}

func buildKiroEventStreamHeaders(eventType string) []byte {
	var buf bytes.Buffer
	const headerName = ":event-type"
	buf.WriteByte(byte(len(headerName)))
	buf.WriteString(headerName)
	buf.WriteByte(7)
	_ = binary.Write(&buf, binary.BigEndian, uint16(len(eventType)))
	buf.WriteString(eventType)
	return buf.Bytes()
}

func TestResolveKiroAPIRegion_PrioritizesRuntimeRegionSources(t *testing.T) {
	t.Run("api_region wins over profile arn", func(t *testing.T) {
		account := &Account{
			Credentials: map[string]any{
				"api_region":  "us-west-2",
				"profile_arn": "arn:aws:codewhisperer:eu-west-1:123456789012:profile/test",
				"region":      "ap-southeast-1",
			},
		}

		require.Equal(t, "us-west-2", ResolveKiroAPIRegion(account))
	})

	t.Run("profile arn wins over oidc region", func(t *testing.T) {
		account := &Account{
			Credentials: map[string]any{
				"profile_arn": "arn:aws:codewhisperer:eu-west-1:123456789012:profile/test",
				"region":      "ap-southeast-1",
			},
		}

		require.Equal(t, "eu-west-1", ResolveKiroAPIRegion(account))
	})

	t.Run("oidc region alone does not become api region", func(t *testing.T) {
		account := &Account{
			Credentials: map[string]any{
				"region": "ap-southeast-1",
			},
		}

		require.Equal(t, kiroDefaultAPIRegion, ResolveKiroAPIRegion(account))
	})
}

func TestBuildKiroEndpointConfigs_PrefersAmazonQThenCodeWhisperer(t *testing.T) {
	endpoints := buildKiroEndpointConfigs("eu-west-1")
	require.Len(t, endpoints, 2)
	require.Equal(t, "https://q.eu-west-1.amazonaws.com/generateAssistantResponse", endpoints[0].URL)
	require.Equal(t, "AmazonQ", endpoints[0].Name)
	require.Empty(t, endpoints[0].AmzTarget)
	require.Equal(t, "https://codewhisperer.eu-west-1.amazonaws.com/generateAssistantResponse", endpoints[1].URL)
	require.Equal(t, "CodeWhisperer", endpoints[1].Name)
	require.Equal(t, "AmazonCodeWhispererStreamingService.GenerateAssistantResponse", endpoints[1].AmzTarget)
}

func TestKiroRuntimeService_ExecuteClaude_UsesKiroHeadersAndOAuthToken(t *testing.T) {
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusUnauthorized, `{"message":"token expired"}`),
		},
	}
	tokenCache := newClaudeTokenCacheStub()
	account := &Account{
		ID:          801,
		Platform:    PlatformKiro,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "stale-token",
			"api_region":   "us-west-2",
			"profile_arn":  "arn:aws:codewhisperer:us-west-2:123456789012:profile/test",
		},
	}
	tokenCache.tokens[KiroTokenCacheKey(account)] = "refreshed-token"
	runtime := NewKiroRuntimeService(&mockAccountRepoForPlatform{}, upstream, NewClaudeTokenProvider(nil, tokenCache, nil))

	result, err := runtime.ExecuteClaude(context.Background(), account, KiroRuntimeExecuteInput{
		Body:    []byte(`{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":"hi"}],"max_tokens":32}`),
		ModelID: "claude-sonnet-4.5",
		Stream:  false,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusUnauthorized, result.Response.StatusCode)
	require.Equal(t, "us-west-2", result.Region)
	require.Equal(t, "AmazonQ", result.Endpoint.Name)
	require.Len(t, upstream.requests, 1)

	req := upstream.requests[0]
	require.Equal(t, "https://q.us-west-2.amazonaws.com/generateAssistantResponse", req.URL.String())
	require.Equal(t, "Bearer refreshed-token", req.Header.Get("Authorization"))
	require.Equal(t, kiroAgentMode, req.Header.Get("x-amzn-kiro-agent-mode"))
	require.Equal(t, "true", req.Header.Get("x-amzn-codewhisperer-optout"))
	require.Equal(t, "attempt=1; max=3", req.Header.Get("Amz-Sdk-Request"))
	require.NotEmpty(t, req.Header.Get("Amz-Sdk-Invocation-Id"))
	require.Contains(t, req.Header.Get("User-Agent"), "KiroIDE-sub2api-")
	require.Contains(t, req.Header.Get("X-Amz-User-Agent"), "aws-sdk-js/")
	require.Empty(t, req.Header.Get("anthropic-beta"))
	require.Empty(t, req.Header.Get("X-Amz-Target"))

	body, readErr := io.ReadAll(req.Body)
	require.NoError(t, readErr)
	require.Contains(t, string(body), `"profileArn":"arn:aws:codewhisperer:us-west-2:123456789012:profile/test"`)
}

func TestKiroRuntimeService_ExecuteClaude_BackfillsProfileARNAndRegion(t *testing.T) {
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusOK, `{"profiles":[{"arn":"arn:aws:codewhisperer:eu-west-1:123456789012:profile/test"}]}`),
			newJSONResponse(http.StatusUnauthorized, `{"message":"expired"}`),
		},
	}
	account := &Account{
		ID:          802,
		Platform:    PlatformKiro,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "live-token",
			"region":       "ap-southeast-1",
		},
	}
	runtime := NewKiroRuntimeService(&mockAccountRepoForPlatform{}, upstream, nil)

	result, err := runtime.ExecuteClaude(context.Background(), account, KiroRuntimeExecuteInput{
		Body:    []byte(`{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":"hi"}]}`),
		ModelID: "claude-sonnet-4.5",
		Stream:  false,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "https://q.us-east-1.amazonaws.com/ListAvailableProfiles", upstream.requests[0].URL.String())
	require.Equal(t, "https://q.eu-west-1.amazonaws.com/generateAssistantResponse", upstream.requests[1].URL.String())
	require.Equal(t, "eu-west-1", result.Region)
	require.Equal(t, "arn:aws:codewhisperer:eu-west-1:123456789012:profile/test", result.ProfileARN)
	require.Equal(t, result.ProfileARN, account.GetCredential("profile_arn"))

	body, readErr := io.ReadAll(upstream.requests[1].Body)
	require.NoError(t, readErr)
	require.Contains(t, string(body), `"profileArn":"arn:aws:codewhisperer:eu-west-1:123456789012:profile/test"`)
}

func TestGatewayService_Forward_KiroUsesRuntimeEndpoint(t *testing.T) {
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusBadRequest, `{"message":"bad request"}`),
		},
	}
	account := &Account{
		ID:          803,
		Name:        "kiro-runtime",
		Platform:    PlatformKiro,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "live-token",
			"profile_arn":  "arn:aws:codewhisperer:us-west-2:123456789012:profile/test",
		},
	}
	svc := &GatewayService{
		httpUpstream: upstream,
	}
	body := []byte(`{"model":"claude-haiku-4.5","messages":[{"role":"user","content":"hi"}],"max_tokens":32}`)
	c, recorder := newSoraTestContext()

	result, err := svc.Forward(c.Request.Context(), c, account, &ParsedRequest{
		Body:     body,
		Model:    "claude-haiku-4.5",
		RawModel: "claude-haiku-4.5",
	})
	require.Nil(t, result)
	require.Error(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://q.us-west-2.amazonaws.com/generateAssistantResponse", upstream.requests[0].URL.String())
	require.NotContains(t, upstream.requests[0].URL.String(), "api.anthropic.com")
	require.Contains(t, recorder.Body.String(), "bad request")
}
