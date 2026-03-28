package admin

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const grokImportPageSize = 200

type grokImportRequest struct {
	Content              string `json:"content" binding:"required"`
	SkipDefaultGroupBind *bool  `json:"skip_default_group_bind"`
}

type grokImportPreviewItem struct {
	Index            int    `json:"index"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	DetectedKind     string `json:"detected_kind"`
	CredentialMasked string `json:"credential_masked"`
	SourcePool       string `json:"source_pool,omitempty"`
	GrokTier         string `json:"grok_tier"`
	Priority         int    `json:"priority"`
	Concurrency      int    `json:"concurrency"`
	Status           string `json:"status"`
	Reason           string `json:"reason,omitempty"`
}

type grokImportPreviewResponse struct {
	DetectedKind string                         `json:"detected_kind,omitempty"`
	Total        int                            `json:"total"`
	Items        []grokImportPreviewItem        `json:"items"`
	Errors       []service.GrokImportParseError `json:"errors,omitempty"`
}

type grokImportResultItem struct {
	Index      int    `json:"index"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	AccountID  int64  `json:"account_id,omitempty"`
	SourcePool string `json:"source_pool,omitempty"`
}

type grokImportResult struct {
	DetectedKind string                         `json:"detected_kind,omitempty"`
	Created      int                            `json:"created"`
	Skipped      int                            `json:"skipped"`
	Failed       int                            `json:"failed"`
	Errors       []service.GrokImportParseError `json:"errors,omitempty"`
	Results      []grokImportResultItem         `json:"results"`
}

func (h *AccountHandler) PreviewGrokImport(c *gin.Context) {
	var req grokImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	parseResult, err := service.ParseGrokImportPayload(req.Content)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	existingKeys, err := h.listExistingGrokCredentialKeys(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]grokImportPreviewItem, 0, len(parseResult.Candidates))
	seen := make(map[string]struct{}, len(parseResult.Candidates))
	for _, candidate := range parseResult.Candidates {
		item := grokImportPreviewItem{
			Index:            candidate.Index,
			Name:             candidate.Name,
			Type:             candidate.Type,
			DetectedKind:     candidate.DetectedKind,
			CredentialMasked: candidate.CredentialMasked,
			SourcePool:       candidate.SourcePool,
			GrokTier:         candidate.Tier,
			Priority:         candidate.Priority,
			Concurrency:      candidate.Concurrency,
			Status:           "ready",
		}
		switch {
		case candidate.CredentialKey == "":
			item.Status = "failed"
			item.Reason = "missing_credential_key"
		case hasCredentialKey(existingKeys, candidate.CredentialKey):
			item.Status = "skipped"
			item.Reason = "already_exists"
		case hasCredentialKey(seen, candidate.CredentialKey):
			item.Status = "skipped"
			item.Reason = "duplicate_in_payload"
		default:
			seen[candidate.CredentialKey] = struct{}{}
		}
		items = append(items, item)
	}

	response.Success(c, grokImportPreviewResponse{
		DetectedKind: parseResult.DetectedKind,
		Total:        len(items),
		Items:        items,
		Errors:       parseResult.Errors,
	})
}

func (h *AccountHandler) ImportGrok(c *gin.Context) {
	var req grokImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	executeAdminIdempotentJSON(c, "admin.grok.import", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		parseResult, err := service.ParseGrokImportPayload(req.Content)
		if err != nil {
			return nil, err
		}

		skipDefaultGroupBind := false
		if req.SkipDefaultGroupBind != nil {
			skipDefaultGroupBind = *req.SkipDefaultGroupBind
		}

		existingKeys, err := h.listExistingGrokCredentialKeys(ctx)
		if err != nil {
			return nil, err
		}

		result := grokImportResult{
			DetectedKind: parseResult.DetectedKind,
			Created:      0,
			Skipped:      0,
			Failed:       len(parseResult.Errors),
			Errors:       parseResult.Errors,
			Results:      make([]grokImportResultItem, 0, len(parseResult.Candidates)+len(parseResult.Errors)),
		}
		seen := make(map[string]struct{}, len(parseResult.Candidates))

		for _, parseErr := range parseResult.Errors {
			result.Results = append(result.Results, grokImportResultItem{
				Index:  parseErr.Index,
				Status: "failed",
				Reason: parseErr.Message,
			})
		}

		for _, candidate := range parseResult.Candidates {
			entry := grokImportResultItem{
				Index:      candidate.Index,
				Name:       candidate.Name,
				Type:       candidate.Type,
				SourcePool: candidate.SourcePool,
			}
			switch {
			case candidate.CredentialKey == "":
				entry.Status = "failed"
				entry.Reason = "missing_credential_key"
				result.Failed++
			case hasCredentialKey(existingKeys, candidate.CredentialKey):
				entry.Status = "skipped"
				entry.Reason = "already_exists"
				result.Skipped++
			case hasCredentialKey(seen, candidate.CredentialKey):
				entry.Status = "skipped"
				entry.Reason = "duplicate_in_payload"
				result.Skipped++
			default:
				account, createErr := h.adminService.CreateAccount(ctx, &service.CreateAccountInput{
					Name:                 candidate.Name,
					Notes:                optionalStringPtr(candidate.Notes),
					Platform:             service.PlatformGrok,
					Type:                 candidate.Type,
					Credentials:          candidate.Credentials,
					Extra:                candidate.Extra,
					Concurrency:          candidate.Concurrency,
					Priority:             candidate.Priority,
					SkipDefaultGroupBind: skipDefaultGroupBind,
				})
				if createErr != nil {
					entry.Status = "failed"
					entry.Reason = createErr.Error()
					result.Failed++
				} else {
					entry.Status = "created"
					entry.AccountID = account.ID
					result.Created++
					existingKeys[candidate.CredentialKey] = struct{}{}
					seen[candidate.CredentialKey] = struct{}{}
				}
			}
			result.Results = append(result.Results, entry)
		}

		return result, nil
	})
}

func (h *AccountHandler) TestGrokAccount(c *gin.Context) {
	if h.accountTestService == nil {
		response.Error(c, 500, "Account test service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil || account == nil {
		response.NotFound(c, "Account not found")
		return
	}
	if !account.IsGrok() {
		response.BadRequest(c, "Account is not a Grok account")
		return
	}

	var req accountTestRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = strings.TrimSpace(req.Model)
	}

	if err := h.accountTestService.TestAccountConnection(c, accountID, modelID, "", "", string(service.AccountTestModeHealthCheck)); err != nil && !c.Writer.Written() {
		response.ErrorFrom(c, err)
	}
}

func (h *AccountHandler) listExistingGrokCredentialKeys(ctx context.Context) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	page := 1
	for {
		accounts, total, err := h.adminService.ListAccounts(ctx, page, grokImportPageSize, service.PlatformGrok, "", "", "", 0, service.AccountLifecycleAll, "")
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			if key := grokCredentialKey(&account); key != "" {
				result[key] = struct{}{}
			}
		}
		if int64(page*grokImportPageSize) >= total || len(accounts) == 0 {
			break
		}
		page++
	}
	return result, nil
}

func grokCredentialKey(account *service.Account) string {
	if account == nil {
		return ""
	}
	if account.IsGrokAPIKey() {
		value := service.NormalizeGrokCredentialValue(service.GrokDetectedKindAPIKey, account.GetGrokAPIKey())
		if value != "" {
			return service.GrokDetectedKindAPIKey + ":" + value
		}
	}
	if account.IsGrokSSO() {
		value := service.NormalizeGrokCredentialValue(service.GrokDetectedKindSSO, account.GetGrokSSOToken())
		if value != "" {
			return service.GrokDetectedKindSSO + ":" + value
		}
	}
	return ""
}

func hasCredentialKey(keys map[string]struct{}, key string) bool {
	_, ok := keys[key]
	return ok
}

func optionalStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
