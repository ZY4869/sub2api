//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateGeminiTestPayload_ImageModel(t *testing.T) {
	t.Parallel()

	payload := createGeminiTestPayload("gemini-2.5-flash-image", "draw a tiny robot")

	var parsed struct {
		Contents []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
		GenerationConfig struct {
			ResponseModalities []string `json:"responseModalities"`
			ImageConfig        struct {
				AspectRatio string `json:"aspectRatio"`
			} `json:"imageConfig"`
		} `json:"generationConfig"`
	}

	require.NoError(t, json.Unmarshal(payload, &parsed))
	require.Len(t, parsed.Contents, 1)
	require.Len(t, parsed.Contents[0].Parts, 1)
	require.Equal(t, "draw a tiny robot", parsed.Contents[0].Parts[0].Text)
	require.Equal(t, []string{"TEXT", "IMAGE"}, parsed.GenerationConfig.ResponseModalities)
	require.Equal(t, "1:1", parsed.GenerationConfig.ImageConfig.AspectRatio)
}

func TestProcessGeminiStream_EmitsImageEvent(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	ctx, recorder := newSoraTestContext()
	svc := &AccountTestService{}

	stream := strings.NewReader("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"},{\"inlineData\":{\"mimeType\":\"image/png\",\"data\":\"QUJD\"}}]}}]}\n\ndata: [DONE]\n\n")

	err := svc.processGeminiStream(ctx, stream)
	require.NoError(t, err)

	body := recorder.Body.String()
	require.Contains(t, body, "\"type\":\"content\"")
	require.Contains(t, body, "\"text\":\"ok\"")
	require.Contains(t, body, "\"type\":\"image\"")
	require.Contains(t, body, "\"image_url\":\"data:image/png;base64,QUJD\"")
	require.Contains(t, body, "\"mime_type\":\"image/png\"")
}

func TestBuildGeminiOAuthRequest_VertexAIUsesPublisherModelsURL(t *testing.T) {
	t.Parallel()

	svc := &AccountTestService{
		cfg: &config.Config{
			Security: config.Security{
				URLAllowlist: config.URLAllowlist{
					AllowInsecureHTTP: true,
				},
			},
		},
		geminiTokenProvider: &GeminiTokenProvider{
			tokenCache: &accountModelImportGeminiTokenCacheStub{token: "vertex-access-token"},
		},
	}
	account := &Account{
		ID:       301,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"oauth_type":        "vertex_ai",
			"vertex_project_id": "vertex-project",
			"vertex_location":   "us-central1",
		},
	}

	req, err := svc.buildGeminiOAuthRequest(
		context.Background(),
		account,
		"gemini-2.5-pro",
		[]byte(`{"contents":[]}`),
		true,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		geminicli.VertexAIBaseURL+"/v1/projects/vertex-project/locations/us-central1/publishers/google/models/gemini-2.5-pro:streamGenerateContent?alt=sse",
		req.URL.String(),
	)
	require.Equal(t, "Bearer vertex-access-token", req.Header.Get("Authorization"))
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
	require.Equal(t, geminicli.GeminiCLIUserAgent, req.Header.Get("User-Agent"))
}

func TestBuildGeminiOAuthRequest_VertexAIRequiresProjectAndLocation(t *testing.T) {
	t.Parallel()

	svc := &AccountTestService{
		cfg: &config.Config{
			Security: config.Security{
				URLAllowlist: config.URLAllowlist{
					AllowInsecureHTTP: true,
				},
			},
		},
		geminiTokenProvider: &GeminiTokenProvider{
			tokenCache: &accountModelImportGeminiTokenCacheStub{token: "vertex-access-token"},
		},
	}

	_, err := svc.buildGeminiOAuthRequest(
		context.Background(),
		&Account{
			ID:       302,
			Platform: PlatformGemini,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"oauth_type":      "vertex_ai",
				"vertex_location": "us-central1",
			},
		},
		"gemini-2.5-pro",
		[]byte(`{"contents":[]}`),
		false,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing vertex_project_id")

	_, err = svc.buildGeminiOAuthRequest(
		context.Background(),
		&Account{
			ID:       303,
			Platform: PlatformGemini,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"oauth_type":        "vertex_ai",
				"vertex_project_id": "vertex-project",
			},
		},
		"gemini-2.5-pro",
		[]byte(`{"contents":[]}`),
		false,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing vertex_location")
}

func TestBuildGeminiOAuthRequest_VertexAIRequiresAccessToken(t *testing.T) {
	t.Parallel()

	svc := &AccountTestService{
		cfg: &config.Config{
			Security: config.Security{
				URLAllowlist: config.URLAllowlist{
					AllowInsecureHTTP: true,
				},
			},
		},
		geminiTokenProvider: &GeminiTokenProvider{
			tokenCache: &accountModelImportGeminiTokenCacheStub{},
		},
	}
	account := &Account{
		ID:       304,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"oauth_type":        "vertex_ai",
			"vertex_project_id": "vertex-project",
			"vertex_location":   "us-central1",
		},
	}

	_, err := svc.buildGeminiOAuthRequest(
		context.Background(),
		account,
		"gemini-2.5-pro",
		[]byte(`{"contents":[]}`),
		false,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "access_token not found in credentials")
}
