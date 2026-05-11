//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type accountRepoStubForBulkUpdate struct {
	accountRepoStub
	bulkUpdateErr    error
	bulkUpdateIDs    []int64
	lastBulkUpdate   AccountBulkUpdate
	bindGroupErrByID map[int64]error
	bindGroupsCalls  []int64
	getByIDsAccounts []*Account
	getByIDsErr      error
	getByIDsCalled   bool
	getByIDsIDs      []int64
	getByIDAccounts  map[int64]*Account
	getByIDErrByID   map[int64]error
	getByIDCalled    []int64
	listByGroupData  map[int64][]Account
	listByGroupErr   map[int64]error
}

func (s *accountRepoStubForBulkUpdate) BulkUpdate(_ context.Context, ids []int64, update AccountBulkUpdate) (int64, error) {
	s.bulkUpdateIDs = append([]int64{}, ids...)
	s.lastBulkUpdate = update
	if s.bulkUpdateErr != nil {
		return 0, s.bulkUpdateErr
	}
	return int64(len(ids)), nil
}

func (s *accountRepoStubForBulkUpdate) BindGroups(_ context.Context, accountID int64, _ []int64) error {
	s.bindGroupsCalls = append(s.bindGroupsCalls, accountID)
	if err, ok := s.bindGroupErrByID[accountID]; ok {
		return err
	}
	return nil
}

func (s *accountRepoStubForBulkUpdate) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	s.getByIDsCalled = true
	s.getByIDsIDs = append([]int64{}, ids...)
	if s.getByIDsErr != nil {
		return nil, s.getByIDsErr
	}
	return s.getByIDsAccounts, nil
}

func (s *accountRepoStubForBulkUpdate) GetByID(_ context.Context, id int64) (*Account, error) {
	s.getByIDCalled = append(s.getByIDCalled, id)
	if err, ok := s.getByIDErrByID[id]; ok {
		return nil, err
	}
	if account, ok := s.getByIDAccounts[id]; ok {
		return account, nil
	}
	return nil, errors.New("account not found")
}

func (s *accountRepoStubForBulkUpdate) ListByGroup(_ context.Context, groupID int64) ([]Account, error) {
	if err, ok := s.listByGroupErr[groupID]; ok {
		return nil, err
	}
	if rows, ok := s.listByGroupData[groupID]; ok {
		return rows, nil
	}
	return nil, nil
}

// TestAdminService_BulkUpdateAccounts_AllSuccessIDs 验证批量更新成功时返回 success_ids/failed_ids。
func TestAdminService_BulkUpdateAccounts_AllSuccessIDs(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo}

	schedulable := true
	input := &BulkUpdateAccountsInput{
		AccountIDs:  []int64{1, 2, 3},
		Schedulable: &schedulable,
	}

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, 3, result.Success)
	require.Equal(t, 0, result.Failed)
	require.ElementsMatch(t, []int64{1, 2, 3}, result.SuccessIDs)
	require.Empty(t, result.FailedIDs)
	require.Len(t, result.Results, 3)
}

// TestAdminService_BulkUpdateAccounts_PartialFailureIDs 验证部分失败时 success_ids/failed_ids 正确。
func TestAdminService_BulkUpdateAccounts_PartialFailureIDs(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		bindGroupErrByID: map[int64]error{
			2: errors.New("bind failed"),
		},
	}
	svc := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &groupRepoStubForAdmin{getByID: &Group{ID: 10, Name: "g10"}},
	}

	groupIDs := []int64{10}
	schedulable := false
	input := &BulkUpdateAccountsInput{
		AccountIDs:            []int64{1, 2, 3},
		GroupIDs:              &groupIDs,
		Schedulable:           &schedulable,
		SkipMixedChannelCheck: true,
	}

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, 2, result.Success)
	require.Equal(t, 1, result.Failed)
	require.ElementsMatch(t, []int64{1, 3}, result.SuccessIDs)
	require.ElementsMatch(t, []int64{2}, result.FailedIDs)
	require.Len(t, result.Results, 3)
}

func TestAdminService_BulkUpdateAccounts_NilGroupRepoReturnsError(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo}

	groupIDs := []int64{10}
	input := &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		GroupIDs:   &groupIDs,
	}

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.Nil(t, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "group repository not configured")
}

func TestAdminService_BulkUpdateAccounts_InvalidBaseURLReturnsStructuredError(t *testing.T) {
	cases := []string{
		"file:///etc/passwd",
		"gopher://127.0.0.1:70",
		"http://127.0.0.1",
		"https:///missing-host",
		"https://api.openai.com:99999",
	}

	for _, rawBaseURL := range cases {
		t.Run(rawBaseURL, func(t *testing.T) {
			repo := &accountRepoStubForBulkUpdate{}
			cfg := &config.Config{}
			cfg.Security.URLAllowlist.Enabled = true
			cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}
			svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

			result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
				AccountIDs: []int64{1},
				Credentials: map[string]any{
					"base_url": rawBaseURL,
				},
			})

			require.Nil(t, result)
			require.Error(t, err)
			require.Equal(t, accountInvalidBaseURLCode, infraerrors.Reason(err))
			require.Empty(t, repo.bulkUpdateIDs)
		})
	}
}

func TestAdminService_BulkUpdateAccounts_NormalizesAllowedBaseURL(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		Credentials: map[string]any{
			"base_url": " https://api.openai.com/ ",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []int64{1}, repo.bulkUpdateIDs)
	require.Equal(t, "https://api.openai.com", repo.lastBulkUpdate.Credentials["base_url"])
}

func TestAdminService_BulkUpdateAccounts_RejectsDisallowedBaiduDocumentAIURL(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		getByIDsAccounts: []*Account{
			{
				ID:       1,
				Platform: PlatformBaiduDocumentAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"async_bearer_token": "token",
				},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.DocumentAIHosts = []string{"paddleocr.aistudio-app.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		Credentials: map[string]any{
			"async_base_url": "https://example.com/api/v2/ocr",
		},
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, baiduDocumentAIInvalidCredentialsCode, infraerrors.Reason(err))
	require.Contains(t, err.Error(), "async_base_url")
	require.Empty(t, repo.bulkUpdateIDs)
}

func TestAdminService_BulkUpdateAccounts_RejectsDisallowedBaiduDocumentAIDirectURL(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		getByIDsAccounts: []*Account{
			{
				ID:       1,
				Platform: PlatformBaiduDocumentAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"direct_token": "direct-token",
				},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.DocumentAIHosts = []string{"paddleocr.aistudio-app.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		Credentials: map[string]any{
			"direct_api_urls": map[string]any{
				DocumentAIModelPPOCRV5Server: "https://example.com/api/v2/ocr/direct",
			},
		},
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, baiduDocumentAIInvalidCredentialsCode, infraerrors.Reason(err))
	require.Contains(t, err.Error(), "direct_api_urls")
	require.Empty(t, repo.bulkUpdateIDs)
}

func TestAdminService_BulkUpdateAccounts_NormalizesAllowedBaiduDocumentAIURLs(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		getByIDsAccounts: []*Account{
			{
				ID:       1,
				Platform: PlatformBaiduDocumentAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"async_bearer_token": "token",
					"direct_token":       "direct-token",
				},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.DocumentAIHosts = []string{"paddleocr.aistudio-app.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		Credentials: map[string]any{
			"async_base_url": " https://paddleocr.aistudio-app.com/api/v2/ocr/ ",
			"direct_api_urls": map[string]any{
				DocumentAIModelPPOCRV5Server: " https://paddleocr.aistudio-app.com/api/v2/ocr/direct/ ",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []int64{1}, repo.bulkUpdateIDs)
	require.Equal(t, "https://paddleocr.aistudio-app.com/api/v2/ocr", repo.lastBulkUpdate.Credentials["async_base_url"])
	directURLs, ok := repo.lastBulkUpdate.Credentials["direct_api_urls"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://paddleocr.aistudio-app.com/api/v2/ocr/direct", directURLs[DocumentAIModelPPOCRV5Server])
}

// TestAdminService_BulkUpdateAccounts_MixedChannelPreCheckBlocksOnExistingConflict verifies
// that the global pre-check detects a conflict with existing group members and returns an
// error before any DB write is performed.
func TestAdminService_BulkUpdateAccounts_MixedChannelPreCheckBlocksOnExistingConflict(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		getByIDsAccounts: []*Account{
			{ID: 1, Platform: PlatformAntigravity, Extra: map[string]any{"mixed_scheduling": true}},
		},
		// Group 10 already contains an Anthropic account.
		listByGroupData: map[int64][]Account{
			10: {{ID: 99, Platform: PlatformAnthropic}},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &groupRepoStubForAdmin{getByID: &Group{ID: 10, Name: "target-group", Platform: PlatformAnthropic}},
	}

	groupIDs := []int64{10}
	input := &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		GroupIDs:   &groupIDs,
	}

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.Nil(t, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mixed channel")
	// No BindGroups should have been called since the check runs before any write.
	require.Empty(t, repo.bindGroupsCalls)
}
