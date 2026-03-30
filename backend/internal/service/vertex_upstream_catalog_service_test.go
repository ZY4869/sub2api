package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestVertexUpstreamCatalogService_GetCatalog_PaginatesAndFormatsOfficialModels(t *testing.T) {
	upstream := &testVertexCatalogHTTPUpstream{}
	listAuthHeader := ""
	upstream.doFunc = func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/v1beta1/publishers/google/models"):
			listAuthHeader = req.Header.Get("Authorization")
			if req.URL.Query().Get("pageToken") == "" {
				return newTestVertexCatalogHTTPResponse(http.StatusOK, `{
					"publisherModels":[
						{"name":"publishers/google/models/gemini-2.5-flash","displayName":"Ignore This","launchStage":"GA"}
					],
					"nextPageToken":"page-2"
				}`), nil
			}
			return newTestVertexCatalogHTTPResponse(http.StatusOK, `{
				"publisherModels":[
					{"name":"publishers/google/models/gemini-3.1-pro-preview","launchStage":"PREVIEW"}
				]
			}`), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, ":countTokens"):
			return newTestVertexCatalogHTTPResponse(http.StatusOK, `{"totalTokens":1}`), nil
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}
	}

	svc := newTestVertexUpstreamCatalogService(upstream, "vertex-access-token")
	result, err := svc.GetCatalog(context.Background(), newTestVertexServiceAccountAccount("us-central1"), true)
	require.NoError(t, err)

	require.Equal(t, "Bearer vertex-access-token", listAuthHeader)
	require.Len(t, result.OfficialModels, 2)
	require.Contains(t, upstream.requests[0], "pageSize=200")
	require.Contains(t, upstream.requests[1], "pageToken=page-2")
	require.Contains(t, strings.Join(upstream.requests, "\n"), "/v1/projects/vertex-project/locations/us-central1/publishers/google/models/gemini-2.5-flash:countTokens")

	flashModel, ok := findVertexCatalogModelByID(result.OfficialModels, "gemini-2.5-flash")
	require.True(t, ok)
	require.Equal(t, FormatModelCatalogDisplayName("gemini-2.5-flash"), flashModel.DisplayName)
	require.Equal(t, "GA", flashModel.LaunchStage)
	require.Equal(t, "publishers/google/models/gemini-2.5-flash", flashModel.OfficialResource)
	require.Equal(t, vertexCatalogOfficialSource, flashModel.UpstreamSource)
}

func TestVertexUpstreamCatalogService_GetCatalog_AddsVerifiedExtraWhenOfficialListMissesCallableModel(t *testing.T) {
	upstream := &testVertexCatalogHTTPUpstream{}
	upstream.doFunc = func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/v1beta1/publishers/google/models"):
			return newTestVertexCatalogHTTPResponse(http.StatusOK, `{
				"publisherModels":[
					{"name":"publishers/google/models/gemini-2.5-flash","launchStage":"GA"}
				]
			}`), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/models/gemini-2.5-flash:countTokens"):
			return newTestVertexCatalogHTTPResponse(http.StatusOK, `{"totalTokens":1}`), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/models/gemini-3.1-pro-preview:countTokens"):
			return newTestVertexCatalogHTTPResponse(http.StatusOK, `{"totalTokens":1}`), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, ":countTokens"):
			return newTestVertexCatalogHTTPResponse(http.StatusNotFound, `{"error":{"code":404,"status":"NOT_FOUND","message":"not found"}}`), nil
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}
	}

	svc := newTestVertexUpstreamCatalogService(upstream, "vertex-access-token")
	result, err := svc.GetCatalog(context.Background(), newTestVertexServiceAccountAccount("global"), true)
	require.NoError(t, err)

	officialModel, ok := findVertexCatalogModelByID(result.OfficialModels, "gemini-2.5-flash")
	require.True(t, ok)
	require.Equal(t, vertexCatalogCallableAvailability, officialModel.Availability)

	extraModel, ok := findVertexCatalogModelByID(result.VerifiedExtras, "gemini-3.1-pro-preview")
	require.True(t, ok)
	require.Equal(t, vertexCatalogVerifiedExtraSource, extraModel.UpstreamSource)
	require.Equal(t, vertexCatalogCallableAvailability, extraModel.Availability)

	_, ok = findVertexCatalogModelByID(result.CallableUnion, "gemini-2.5-flash")
	require.True(t, ok)
	_, ok = findVertexCatalogModelByID(result.CallableUnion, "gemini-3.1-pro-preview")
	require.True(t, ok)
}

func TestVertexUpstreamCatalogService_GetCatalog_ReturnsErrorWhenOfficialListFails(t *testing.T) {
	upstream := &testVertexCatalogHTTPUpstream{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/v1beta1/publishers/google/models") {
				return newTestVertexCatalogHTTPResponse(http.StatusServiceUnavailable, `{"error":{"code":503,"message":"backend unavailable","status":"UNAVAILABLE"}}`), nil
			}
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		},
	}

	svc := newTestVertexUpstreamCatalogService(upstream, "vertex-access-token")
	_, err := svc.GetCatalog(context.Background(), newTestVertexServiceAccountAccount("global"), true)
	require.Error(t, err)

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusServiceUnavailable), appErr.Code)
	require.Contains(t, appErr.Message, "vertex official model listing failed with status 503")
}

func TestVertexUpstreamCatalogService_GetCatalog_VertexExpressUsesAPIKeyAndPropagatesListRejection(t *testing.T) {
	upstream := &testVertexCatalogHTTPUpstream{}
	listKey := ""
	upstream.doFunc = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/v1beta1/publishers/google/models") {
			listKey = req.URL.Query().Get("key")
			return newTestVertexCatalogHTTPResponse(http.StatusForbidden, `{"error":{"code":403,"message":"Permission denied","status":"PERMISSION_DENIED"}}`), nil
		}
		t.Fatalf("unexpected request: %s", req.URL.String())
		return nil, nil
	}

	svc := newTestVertexUpstreamCatalogService(upstream, "")
	_, err := svc.GetCatalog(context.Background(), newTestVertexExpressAccount(), true)
	require.Error(t, err)

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, accountModelImportReasonKindPermissionDenied, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyPermissionDenied, appErr.Metadata["hint_key"])
	require.Equal(t, "vertex-express-key", listKey)
	require.Contains(t, appErr.Message, "status 403")
}

func TestNormalizeVertexUpstreamModelID_PreservesLegacyAliasCompatibility(t *testing.T) {
	require.Equal(t, "gemini-3.1-flash-image-preview", normalizeVertexUpstreamModelID("gemini-3.1-flash-image"))
	require.Equal(t, "gemini-3.1-flash-image-preview", normalizeVertexUpstreamModelID("models/gemini-3.1-flash-image-preview"))
}

func newTestVertexServiceAccountAccount(location string) *Account {
	return &Account{
		ID:       9001,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type":                  "vertex_ai",
			"vertex_project_id":           "vertex-project",
			"vertex_location":             location,
			"vertex_service_account_json": `{"type":"service_account","client_email":"svc@example.com","private_key":"test","token_uri":"https://oauth2.googleapis.com/token"}`,
		},
	}
}

func newTestVertexExpressAccount() *Account {
	return &Account{
		ID:       9002,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":            "vertex-express-key",
			"gemini_api_variant": GeminiAPIKeyVariantVertexExpress,
		},
	}
}
