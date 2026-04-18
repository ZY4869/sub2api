package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestForwardSimplifiedVertexBatchPredictionJobs_CreateInjectsManagedOutput(t *testing.T) {
	groupID := int64(7)
	account := newGoogleBatchVertexOAuthTestAccount(701, "vertex-project", "us-central1")
	accountRepo := &googleBatchSelectionAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{accountsByID: map[int64]*Account{account.ID: account}},
		schedulableByGroup:         map[int64][]Account{groupID: []Account{*account}},
	}
	settingService := NewSettingService(&staticSettingRepoStub{
		values: map[string]string{
			SettingKeyGoogleBatchGCSProfiles: buildGoogleBatchGCSProfilesStoreJSON(t),
		},
	}, nil)

	svc := &GeminiMessagesCompatService{
		accountRepo:    accountRepo,
		settingService: settingService,
		tokenProvider:  NewGeminiTokenProvider(nil, nil, nil),
		httpUpstream: googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
			require.Equal(t, account.ID, accountID)
			require.Equal(t, "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `"model":"publishers/google/models/gemini-2.5-pro"`)
			require.Contains(t, string(body), `"outputUriPrefix":"gs://managed-bucket/managed-prefix/`)
			return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1"}`), nil
		}),
	}

	result, resolvedAccount, err := svc.ForwardSimplifiedVertexBatchPredictionJobs(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodPost,
		Path:    "/v1/vertex/batchPredictionJobs",
		Body:    []byte(`{"displayName":"job-1","model":"gemini-2.5-pro","inputConfig":{"instancesFormat":"jsonl","gcsSource":{"uris":["gs://custom/input.jsonl"]}}}`),
	})
	require.NoError(t, err)
	require.NotNil(t, resolvedAccount)
	require.Equal(t, account.ID, resolvedAccount.ID)

	httpResult, ok := result.(*UpstreamHTTPResult)
	require.True(t, ok)
	require.Contains(t, string(httpResult.Body), `"name":"batchPredictionJobs/job-1"`)
}

func TestForwardSimplifiedVertexBatchPredictionJobs_UsesBindingForFollowUp(t *testing.T) {
	groupID := int64(8)
	accountA := newGoogleBatchVertexOAuthTestAccount(801, "vertex-project", "us-central1")
	accountB := newGoogleBatchVertexOAuthTestAccount(802, "vertex-project-b", "europe-west4")
	accountRepo := &googleBatchSelectionAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{accountsByID: map[int64]*Account{accountA.ID: accountA, accountB.ID: accountB}},
		schedulableByGroup:         map[int64][]Account{groupID: []Account{*accountA, *accountB}},
	}
	bindingRepo := &googleBatchBindingRepoStub{items: map[string]*UpstreamResourceBinding{
		strings.ToLower(buildVertexBatchJobResourceName("vertex-project-b", "europe-west4", "job-42")): {
			ResourceKind: UpstreamResourceKindVertexBatchJob,
			ResourceName: buildVertexBatchJobResourceName("vertex-project-b", "europe-west4", "job-42"),
			AccountID:    accountB.ID,
		},
	}}

	svc := &GeminiMessagesCompatService{
		accountRepo:         accountRepo,
		resourceBindingRepo: bindingRepo,
		tokenProvider:       NewGeminiTokenProvider(nil, nil, nil),
		httpUpstream: googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
			require.Equal(t, accountB.ID, accountID)
			require.Equal(t, "/v1/projects/vertex-project-b/locations/europe-west4/batchPredictionJobs/job-42", req.URL.Path)
			return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"projects/vertex-project-b/locations/europe-west4/batchPredictionJobs/job-42","state":"JOB_STATE_SUCCEEDED"}`), nil
		}),
	}

	result, resolvedAccount, err := svc.ForwardSimplifiedVertexBatchPredictionJobs(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodGet,
		Path:    "/vertex-batch/jobs/job-42",
	})
	require.NoError(t, err)
	require.NotNil(t, resolvedAccount)
	require.Equal(t, accountB.ID, resolvedAccount.ID)

	httpResult, ok := result.(*UpstreamHTTPResult)
	require.True(t, ok)
	require.Contains(t, string(httpResult.Body), `"name":"batchPredictionJobs/job-42"`)
}

func TestForwardSimplifiedVertexBatchPredictionJobs_MissingManagedProfileFails(t *testing.T) {
	groupID := int64(9)
	account := newGoogleBatchVertexOAuthTestAccount(901, "vertex-project", "us-central1")
	accountRepo := &googleBatchSelectionAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{accountsByID: map[int64]*Account{account.ID: account}},
		schedulableByGroup:         map[int64][]Account{groupID: []Account{*account}},
	}
	svc := &GeminiMessagesCompatService{
		accountRepo: accountRepo,
	}

	_, _, err := svc.ForwardSimplifiedVertexBatchPredictionJobs(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodPost,
		Path:    "/v1/vertex/batchPredictionJobs",
		Body:    []byte(`{"model":"gemini-2.5-pro","inputConfig":{"instancesFormat":"jsonl","gcsSource":{"uris":["gs://custom/input.jsonl"]}}}`),
	})
	appErr := infraerrors.FromError(err)
	require.NotNil(t, appErr)
	require.Equal(t, "VERTEX_SIMPLIFIED_GCS_PROFILE_UNAVAILABLE", appErr.Reason)
}

func TestForwardSimplifiedVertexBatchPredictionJobs_CreateUsesUnifiedVertexSelection(t *testing.T) {
	groupID := int64(10)
	accountA := newGoogleBatchVertexOAuthTestAccount(1001, "vertex-project-a", "us-central1")
	accountA.Priority = 20
	accountB := newGoogleBatchVertexOAuthTestAccount(1002, "vertex-project-b", "europe-west4")
	accountB.Priority = 1

	accountRepo := &googleBatchSelectionAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{
			accountsByID: map[int64]*Account{
				accountA.ID: accountA,
				accountB.ID: accountB,
			},
		},
		schedulableByGroup: map[int64][]Account{
			groupID: []Account{*accountA, *accountB},
		},
	}

	svc := &GeminiMessagesCompatService{
		accountRepo:   accountRepo,
		tokenProvider: NewGeminiTokenProvider(nil, nil, nil),
		httpUpstream: googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
			require.Equal(t, accountB.ID, accountID)
			require.Equal(t, "/v1/projects/vertex-project-b/locations/europe-west4/batchPredictionJobs", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `"model":"publishers/google/models/gemini-2.5-pro"`)
			require.Contains(t, string(body), `"gs://custom/input.jsonl"`)
			require.Contains(t, string(body), `"gs://custom/output"`)
			return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"projects/vertex-project-b/locations/europe-west4/batchPredictionJobs/job-priority"}`), nil
		}),
	}

	result, resolvedAccount, err := svc.ForwardSimplifiedVertexBatchPredictionJobs(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodPost,
		Path:    "/vertex-batch/jobs",
		Body:    []byte(`{"model":"gemini-2.5-pro","display_name":"job-priority","input_uri":"gs://custom/input.jsonl","output_uri_prefix":"gs://custom/output"}`),
	})
	require.NoError(t, err)
	require.NotNil(t, resolvedAccount)
	require.Equal(t, accountB.ID, resolvedAccount.ID)

	httpResult, ok := result.(*UpstreamHTTPResult)
	require.True(t, ok)
	require.Contains(t, string(httpResult.Body), `"name":"batchPredictionJobs/job-priority"`)
}

func buildGoogleBatchGCSProfilesStoreJSON(t *testing.T) string {
	t.Helper()
	store := googleBatchGCSProfilesStore{
		ActiveProfileID: "managed-profile",
		Items: []googleBatchGCSProfileStoreItem{
			{
				ProfileID:          "managed-profile",
				Name:               "Managed",
				Enabled:            true,
				Bucket:             "managed-bucket",
				Prefix:             "managed-prefix",
				ProjectID:          "managed-project",
				ServiceAccountJSON: `{"client_email":"managed@example.com","private_key_id":"kid","private_key":"-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n","token_uri":"https://oauth2.googleapis.com/token"}`,
				UpdatedAt:          "2026-04-18T00:00:00Z",
			},
		},
	}
	data, err := json.Marshal(store)
	require.NoError(t, err)
	return string(data)
}
