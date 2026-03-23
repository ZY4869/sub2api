package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BatchCreateAccountsArchiveRequest struct {
	Enabled   bool   `json:"enabled"`
	GroupName string `json:"group_name"`
}

type BatchCreateAccountsRequest struct {
	Platform                string                            `json:"platform" binding:"required"`
	Type                    string                            `json:"type" binding:"required,oneof=oauth setup-token apikey upstream"`
	Items                   []string                          `json:"items" binding:"required,min=1"`
	NamePrefix              string                            `json:"name_prefix"`
	Notes                   *string                           `json:"notes"`
	Credentials             map[string]any                    `json:"credentials"`
	Extra                   map[string]any                    `json:"extra"`
	ProxyID                 *int64                            `json:"proxy_id"`
	Concurrency             int                               `json:"concurrency"`
	LoadFactor              *int                              `json:"load_factor"`
	Priority                int                               `json:"priority"`
	RateMultiplier          *float64                          `json:"rate_multiplier"`
	GroupIDs                []int64                           `json:"group_ids"`
	ExpiresAt               *int64                            `json:"expires_at"`
	AutoPauseOnExpired      *bool                             `json:"auto_pause_on_expired"`
	ConfirmMixedChannelRisk *bool                             `json:"confirm_mixed_channel_risk"`
	AutoImportModels        bool                              `json:"auto_import_models"`
	Archive                 BatchCreateAccountsArchiveRequest `json:"archive"`
}

type BatchCreateAccountsResult struct {
	CreatedCount     int                            `json:"created_count"`
	FailedCount      int                            `json:"failed_count"`
	ArchiveGroupID   *int64                         `json:"archive_group_id,omitempty"`
	ArchiveGroupName string                         `json:"archive_group_name,omitempty"`
	Results          []BatchCreateAccountLineResult `json:"results"`
}

type BatchCreateAccountLineResult struct {
	LineIndex   int    `json:"line_index"`
	RawPreview  string `json:"raw_preview"`
	Success     bool   `json:"success"`
	AccountID   int64  `json:"account_id,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	Message     string `json:"message"`
}

const (
	batchCreateDefaultConcurrency = 10
	batchCreateDefaultPriority    = 1
	batchCreateDefaultMultiplier  = 1.0
)

func (h *AccountHandler) BatchCreateAccounts(c *gin.Context) {
	var req BatchCreateAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !supportsBatchCreateAccountType(req.Platform, req.Type) {
		response.ErrorFrom(c, infraerrors.BadRequest("ACCOUNT_BATCH_CREATE_UNSUPPORTED", "current platform/type does not support direct batch credential import"))
		return
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		response.BadRequest(c, "rate_multiplier must be >= 0")
		return
	}
	if req.Archive.Enabled && strings.TrimSpace(req.Archive.GroupName) == "" {
		response.BadRequest(c, "archive.group_name is required when archive is enabled")
		return
	}

	executeAdminIdempotentJSON(c, "admin.accounts.batch_create_v2", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.executeBatchCreateAccounts(ctx, &req)
	})
}

func (h *AccountHandler) executeBatchCreateAccounts(ctx context.Context, req *BatchCreateAccountsRequest) (*BatchCreateAccountsResult, error) {
	namePrefix := resolveBatchCreateNamePrefix(req.Platform, req.NamePrefix, time.Now())
	result := &BatchCreateAccountsResult{
		Results: make([]BatchCreateAccountLineResult, 0, len(req.Items)),
	}

	var archiveGroup *service.Group
	var err error
	if req.Archive.Enabled {
		archiveGroup, err = h.resolveBatchCreateArchiveGroup(ctx, req.Platform, req.Archive.GroupName)
		if err != nil {
			return nil, err
		}
		result.ArchiveGroupName = archiveGroup.Name
		result.ArchiveGroupID = &archiveGroup.ID
	}

	slog.Info("admin_account_batch_create_started",
		"platform", req.Platform,
		"type", req.Type,
		"line_count", len(req.Items),
		"archive_enabled", req.Archive.Enabled,
		"archive_group_name", strings.TrimSpace(req.Archive.GroupName),
	)

	sequence := 0
	for index, item := range req.Items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		sequence++

		line, parseErr := parseBatchCreateLine(trimmed, req.Platform, req.Type)
		if parseErr != nil {
			result.FailedCount++
			result.Results = append(result.Results, BatchCreateAccountLineResult{
				LineIndex:  index + 1,
				RawPreview: batchCreatePreview(trimmed),
				Success:    false,
				Message:    parseErr.Error(),
			})
			continue
		}

		input, ignoredGroupBinding, buildErr := buildBatchCreateAccountInput(req, line, namePrefix, sequence, archiveGroup)
		if buildErr != nil {
			result.FailedCount++
			result.Results = append(result.Results, BatchCreateAccountLineResult{
				LineIndex:  index + 1,
				RawPreview: line.RawPreview,
				Success:    false,
				Message:    buildErr.Error(),
			})
			continue
		}

		credentials, extra, scopeErr := h.prepareAccountModelScope(ctx, req.Platform, req.Type, input.Credentials, input.Extra)
		if scopeErr != nil {
			result.FailedCount++
			result.Results = append(result.Results, BatchCreateAccountLineResult{
				LineIndex:  index + 1,
				RawPreview: line.RawPreview,
				Success:    false,
				Message:    scopeErr.Error(),
			})
			continue
		}
		input.Credentials = credentials
		input.Extra = extra

		account, createErr := h.adminService.CreateAccount(ctx, input)
		if createErr != nil {
			result.FailedCount++
			result.Results = append(result.Results, BatchCreateAccountLineResult{
				LineIndex:  index + 1,
				RawPreview: line.RawPreview,
				Success:    false,
				Message:    createErr.Error(),
			})
			continue
		}

		messageParts := []string{"created"}
		if ignoredGroupBinding {
			messageParts = append(messageParts, "archive mode ignored provided group_ids")
		}
		if req.AutoImportModels && !req.Archive.Enabled {
			messageParts = append(messageParts, h.importModelsAfterBatchCreate(ctx, account))
		}

		result.CreatedCount++
		result.Results = append(result.Results, BatchCreateAccountLineResult{
			LineIndex:   index + 1,
			RawPreview:  line.RawPreview,
			Success:     true,
			AccountID:   account.ID,
			AccountName: account.Name,
			Message:     strings.Join(compactBatchCreateMessages(messageParts), "; "),
		})
	}

	slog.Info("admin_account_batch_create_completed",
		"platform", req.Platform,
		"type", req.Type,
		"created_count", result.CreatedCount,
		"failed_count", result.FailedCount,
		"archive_enabled", req.Archive.Enabled,
		"archive_group_name", result.ArchiveGroupName,
	)

	return result, nil
}

func buildBatchCreateAccountInput(
	req *BatchCreateAccountsRequest,
	line *batchCreateLineOverrides,
	namePrefix string,
	sequence int,
	archiveGroup *service.Group,
) (*service.CreateAccountInput, bool, error) {
	name := strings.TrimSpace(line.Name)
	if name == "" {
		name = fmt.Sprintf("%s-%03d", namePrefix, sequence)
	}

	credentials := service.MergeCredentials(cloneStringAnyMap(req.Credentials), cloneStringAnyMap(line.Credentials))
	if strings.EqualFold(strings.TrimSpace(req.Platform), service.PlatformKiro) {
		credentials = service.NormalizeKiroCredentialsForStorage(credentials)
	}
	if err := validateBatchCreateCredentials(req.Platform, req.Type, credentials); err != nil {
		return nil, false, err
	}

	extra := service.MergeStringAnyMap(cloneStringAnyMap(req.Extra), cloneStringAnyMap(line.Extra))
	sanitizeExtraBaseRPM(extra)

	groupIDs := append([]int64(nil), req.GroupIDs...)
	if line.GroupIDs != nil {
		groupIDs = append([]int64(nil), (*line.GroupIDs)...)
	}
	ignoredGroupBinding := false
	status := service.StatusActive
	lifecycleState := service.AccountLifecycleNormal
	lifecycleReasonCode := ""
	lifecycleReasonMessage := ""
	if archiveGroup != nil {
		ignoredGroupBinding = len(groupIDs) > 0
		groupIDs = []int64{archiveGroup.ID}
		status = service.StatusDisabled
		lifecycleState = service.AccountLifecycleArchived
		lifecycleReasonCode = "batch_create_archive"
		lifecycleReasonMessage = "Created directly into archive area"
	}

	concurrency := req.Concurrency
	if line.Concurrency != nil {
		concurrency = *line.Concurrency
	}
	if concurrency <= 0 {
		concurrency = batchCreateDefaultConcurrency
	}

	priority := req.Priority
	if line.Priority != nil {
		priority = *line.Priority
	}
	if priority <= 0 {
		priority = batchCreateDefaultPriority
	}

	rateMultiplier := req.RateMultiplier
	if line.RateMultiplier != nil {
		rateMultiplier = line.RateMultiplier
	}
	if rateMultiplier == nil {
		defaultRateMultiplier := batchCreateDefaultMultiplier
		rateMultiplier = &defaultRateMultiplier
	}
	if *rateMultiplier < 0 {
		return nil, false, fmt.Errorf("rate_multiplier must be >= 0")
	}

	loadFactor := req.LoadFactor
	if line.LoadFactor != nil {
		loadFactor = line.LoadFactor
	}

	expiresAt := req.ExpiresAt
	if line.ExpiresAt != nil {
		expiresAt = line.ExpiresAt
	}

	autoPauseOnExpired := req.AutoPauseOnExpired
	if line.AutoPauseOnExpired != nil {
		autoPauseOnExpired = line.AutoPauseOnExpired
	}

	skipMixedChannelCheck := false
	if req.ConfirmMixedChannelRisk != nil {
		skipMixedChannelCheck = *req.ConfirmMixedChannelRisk
	}
	if line.ConfirmMixedChannelRisk != nil {
		skipMixedChannelCheck = *line.ConfirmMixedChannelRisk
	}

	return &service.CreateAccountInput{
		Name:                  name,
		Notes:                 firstNonNilStringPointer(line.Notes, req.Notes),
		Platform:              req.Platform,
		Type:                  req.Type,
		Credentials:           credentials,
		Extra:                 extra,
		ProxyID:               firstNonNilInt64Pointer(line.ProxyID, req.ProxyID),
		Concurrency:           concurrency,
		Priority:              priority,
		RateMultiplier:        rateMultiplier,
		LoadFactor:            loadFactor,
		GroupIDs:              groupIDs,
		Status:                status,
		LifecycleState:        lifecycleState,
		LifecycleReasonCode:   lifecycleReasonCode,
		LifecycleReasonMessage: lifecycleReasonMessage,
		ExpiresAt:             expiresAt,
		AutoPauseOnExpired:    autoPauseOnExpired,
		SkipMixedChannelCheck: skipMixedChannelCheck,
	}, ignoredGroupBinding, nil
}

func (h *AccountHandler) resolveBatchCreateArchiveGroup(ctx context.Context, platform string, groupName string) (*service.Group, error) {
	trimmedName := strings.TrimSpace(groupName)
	group, err := h.adminService.GetGroupByName(ctx, trimmedName)
	if err == nil && group != nil {
		if !strings.EqualFold(group.Platform, platform) {
			return nil, infraerrors.BadRequest("ACCOUNT_BATCH_CREATE_ARCHIVE_GROUP_PLATFORM_CONFLICT", fmt.Sprintf("group %q already exists under platform %s", trimmedName, group.Platform))
		}
		return group, nil
	}
	if err != nil && !errors.Is(err, service.ErrGroupNotFound) {
		return nil, err
	}

	created, createErr := h.adminService.CreateGroup(ctx, &service.CreateGroupInput{
		Name:             trimmedName,
		Description:      "Archive group for batch-created inactive accounts",
		Platform:         platform,
		RateMultiplier:   1,
		IsExclusive:      false,
		SubscriptionType: service.SubscriptionTypeStandard,
	})
	if createErr == nil {
		return created, nil
	}

	group, err = h.adminService.GetGroupByName(ctx, trimmedName)
	if err == nil && group != nil {
		if !strings.EqualFold(group.Platform, platform) {
			return nil, infraerrors.BadRequest("ACCOUNT_BATCH_CREATE_ARCHIVE_GROUP_PLATFORM_CONFLICT", fmt.Sprintf("group %q already exists under platform %s", trimmedName, group.Platform))
		}
		return group, nil
	}
	return nil, createErr
}

func (h *AccountHandler) importModelsAfterBatchCreate(ctx context.Context, account *service.Account) string {
	if h.accountModelImportService == nil || account == nil || !account.IsActive() {
		return ""
	}
	result, err := h.accountModelImportService.ImportAccountModels(ctx, account, "create")
	if err != nil {
		return "model import failed: " + err.Error()
	}
	if result == nil {
		return ""
	}
	return fmt.Sprintf("model import completed (%d imported)", result.ImportedCount)
}

func supportsBatchCreateAccountType(platform string, accountType string) bool {
	normalizedPlatform := strings.ToLower(strings.TrimSpace(platform))
	if normalizedPlatform == service.PlatformCopilot {
		return false
	}
	if normalizedPlatform == service.PlatformKiro {
		return strings.ToLower(strings.TrimSpace(accountType)) == service.AccountTypeOAuth
	}
	return true
}

func resolveBatchCreateNamePrefix(platform string, current string, now time.Time) string {
	if trimmed := strings.TrimSpace(current); trimmed != "" {
		return trimmed
	}
	normalizedPlatform := strings.TrimSpace(platform)
	if normalizedPlatform == "" {
		normalizedPlatform = "account"
	}
	return fmt.Sprintf("%s-batch-%s", normalizedPlatform, now.Format("20060102-1504"))
}

func compactBatchCreateMessages(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonNilStringPointer(values ...*string) *string {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonNilInt64Pointer(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			if *value == 0 {
				return nil
			}
			return value
		}
	}
	return nil
}
