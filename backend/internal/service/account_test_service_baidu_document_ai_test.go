package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newAccountTestServiceContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/admin/accounts/test", nil)
	return ctx, rec
}

func TestAccountTestServiceBaiduForcesHealthCheckAndPrefersAsync(t *testing.T) {
	ctx, recorder := newAccountTestServiceContext()

	account := &Account{
		ID:          101,
		Platform:    PlatformBaiduDocumentAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"async_bearer_token": "async-token",
			"async_base_url":     DefaultBaiduDocumentAIAsyncBaseURL(),
			"direct_token":       "direct-token",
			"direct_api_urls": map[string]any{
				DocumentAIModelPPOCRV5Server: "https://direct.example.com/ocr",
			},
		},
	}

	accountRepo := &googleBatchAccountRepoStub{
		accountsByID: map[int64]*Account{account.ID: account},
	}

	var requestURLs []string
	upstream := googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		requestURLs = append(requestURLs, req.URL.String())
		switch len(requestURLs) {
		case 1:
			require.Equal(t, DefaultBaiduDocumentAIAsyncBaseURL()+"/jobs", req.URL.String())
			return documentAIServiceResponse(http.StatusOK, `{"jobId":"provider-job-1","status":"RUNNING"}`, nil), nil
		case 2:
			require.Equal(t, DefaultBaiduDocumentAIAsyncBaseURL()+"/jobs/provider-job-1", req.URL.String())
			return documentAIServiceResponse(http.StatusOK, `{"status":"RUNNING"}`, nil), nil
		default:
			t.Fatalf("unexpected upstream request %d", len(requestURLs))
			return nil, nil
		}
	})

	svc := &AccountTestService{
		accountRepo:  accountRepo,
		httpUpstream: upstream,
	}

	err := svc.TestAccountConnection(ctx, account.ID, "", "", "", "", "", string(AccountTestModeRealForward))
	require.NoError(t, err)
	require.Len(t, requestURLs, 2)
	require.Contains(t, recorder.Body.String(), "Document AI mode: async")
	require.Contains(t, recorder.Body.String(), "Async job submitted")
	require.Contains(t, recorder.Body.String(), `"type":"test_complete"`)
	require.NotContains(t, recorder.Body.String(), "Direct API URL")
}

func TestAccountTestServiceBaiduFallsBackToDirectWithoutAsyncToken(t *testing.T) {
	ctx, recorder := newAccountTestServiceContext()

	account := &Account{
		ID:          102,
		Platform:    PlatformBaiduDocumentAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"direct_token": "direct-token",
			"direct_api_urls": map[string]any{
				DocumentAIModelPPOCRV5Server: "https://direct.example.com/ocr",
			},
		},
	}

	accountRepo := &googleBatchAccountRepoStub{
		accountsByID: map[int64]*Account{account.ID: account},
	}

	upstream := googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		require.Equal(t, "https://direct.example.com/ocr", req.URL.String())
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `"fileType":1`)
		return documentAIServiceResponse(http.StatusOK, `{"result":{"text":"ok"}}`, nil), nil
	})

	svc := &AccountTestService{
		accountRepo:  accountRepo,
		httpUpstream: upstream,
	}

	err := svc.TestAccountConnection(ctx, account.ID, "", "", "", "", "", string(AccountTestModeRealForward))
	require.NoError(t, err)
	require.Contains(t, recorder.Body.String(), "Document AI mode: direct")
	require.Contains(t, recorder.Body.String(), "Direct parse completed")
	require.Contains(t, recorder.Body.String(), `"type":"test_complete"`)
}
