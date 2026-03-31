package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func toGoogleBatchGCSProfileDTO(profile service.GoogleBatchGCSProfile) dto.GoogleBatchGCSProfile {
	return dto.GoogleBatchGCSProfile{
		ProfileID:                    profile.ProfileID,
		Name:                         profile.Name,
		IsActive:                     profile.IsActive,
		Enabled:                      profile.Enabled,
		Bucket:                       profile.Bucket,
		Prefix:                       profile.Prefix,
		ProjectID:                    profile.ProjectID,
		ServiceAccountJSONConfigured: profile.ServiceAccountJSONConfigured,
		UpdatedAt:                    profile.UpdatedAt,
	}
}

func findGoogleBatchGCSProfile(items []service.GoogleBatchGCSProfile, profileID string) *service.GoogleBatchGCSProfile {
	target := strings.TrimSpace(profileID)
	for idx := range items {
		if items[idx].ProfileID == target {
			return &items[idx]
		}
	}
	return nil
}

type CreateGoogleBatchGCSProfileRequest struct {
	ProfileID          string `json:"profile_id"`
	Name               string `json:"name"`
	SetActive          bool   `json:"set_active"`
	Enabled            bool   `json:"enabled"`
	Bucket             string `json:"bucket"`
	Prefix             string `json:"prefix"`
	ProjectID          string `json:"project_id"`
	ServiceAccountJSON string `json:"service_account_json"`
}

type UpdateGoogleBatchGCSProfileRequest struct {
	Name               string `json:"name"`
	Enabled            bool   `json:"enabled"`
	Bucket             string `json:"bucket"`
	Prefix             string `json:"prefix"`
	ProjectID          string `json:"project_id"`
	ServiceAccountJSON string `json:"service_account_json"`
}

type TestGoogleBatchGCSConnectionRequest struct {
	ProfileID          string `json:"profile_id"`
	Enabled            bool   `json:"enabled"`
	Bucket             string `json:"bucket"`
	Prefix             string `json:"prefix"`
	ProjectID          string `json:"project_id"`
	ServiceAccountJSON string `json:"service_account_json"`
}

func (h *SettingHandler) GetGeminiRateCatalog(c *gin.Context) {
	catalog := h.settingService.GetGeminiRateCatalog(c.Request.Context())
	if catalog == nil {
		response.Success(c, dto.GeminiRateCatalog{})
		return
	}
	tiers := make([]dto.GeminiRateCatalogTier, 0, len(catalog.AIStudioTiers))
	for _, tier := range catalog.AIStudioTiers {
		rows := make([]dto.GeminiRateCatalogModelRow, 0, len(tier.ModelFamilies))
		for _, row := range tier.ModelFamilies {
			rows = append(rows, dto.GeminiRateCatalogModelRow{
				ModelFamily: row.ModelFamily,
				DisplayName: row.DisplayName,
				RPM:         row.RPM,
				TPM:         row.TPM,
				RPD:         row.RPD,
				Notes:       row.Notes,
			})
		}
		tiers = append(tiers, dto.GeminiRateCatalogTier{
			TierID:         tier.TierID,
			DisplayName:    tier.DisplayName,
			Qualification:  tier.Qualification,
			BillingTierCap: tier.BillingTierCap,
			ModelFamilies:  rows,
		})
	}
	byTier := make([]dto.GeminiRateCatalogBatchTier, 0, len(catalog.BatchLimits.ByTier))
	for _, tier := range catalog.BatchLimits.ByTier {
		entries := make([]dto.GeminiRateCatalogBatchRow, 0, len(tier.Entries))
		for _, entry := range tier.Entries {
			entries = append(entries, dto.GeminiRateCatalogBatchRow{
				ModelFamily:    entry.ModelFamily,
				DisplayName:    entry.DisplayName,
				EnqueuedTokens: entry.EnqueuedTokens,
			})
		}
		byTier = append(byTier, dto.GeminiRateCatalogBatchTier{
			TierID:  tier.TierID,
			Entries: entries,
		})
	}
	links := make([]dto.GeminiRateCatalogLink, 0, len(catalog.Links))
	for _, link := range catalog.Links {
		links = append(links, dto.GeminiRateCatalogLink{
			Label: link.Label,
			URL:   link.URL,
		})
	}
	response.Success(c, dto.GeminiRateCatalog{
		EffectiveDate:              catalog.EffectiveDate,
		RemainingQuotaAPISupported: catalog.RemainingQuotaAPISupported,
		AIStudioTiers:              tiers,
		BatchLimits: dto.GeminiRateCatalogBatchLimits{
			ConcurrentBatchRequests: catalog.BatchLimits.ConcurrentBatchRequests,
			InputFileSizeLimitBytes: catalog.BatchLimits.InputFileSizeLimitBytes,
			FileStorageLimitBytes:   catalog.BatchLimits.FileStorageLimitBytes,
			ByTier:                  byTier,
		},
		Links: links,
		Notes: append([]string(nil), catalog.Notes...),
	})
}

func (h *SettingHandler) ListGoogleBatchGCSProfiles(c *gin.Context) {
	result, err := h.settingService.ListGoogleBatchGCSProfiles(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.GoogleBatchGCSProfile, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toGoogleBatchGCSProfileDTO(item))
	}
	response.Success(c, dto.ListGoogleBatchGCSProfilesResponse{
		ActiveProfileID: result.ActiveProfileID,
		Items:           items,
	})
}

func (h *SettingHandler) CreateGoogleBatchGCSProfile(c *gin.Context) {
	var req CreateGoogleBatchGCSProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	created, err := h.settingService.CreateGoogleBatchGCSProfile(c.Request.Context(), &service.GoogleBatchGCSProfile{
		ProfileID:          req.ProfileID,
		Name:               req.Name,
		Enabled:            req.Enabled,
		Bucket:             req.Bucket,
		Prefix:             req.Prefix,
		ProjectID:          req.ProjectID,
		ServiceAccountJSON: req.ServiceAccountJSON,
	}, req.SetActive)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toGoogleBatchGCSProfileDTO(*created))
}

func (h *SettingHandler) UpdateGoogleBatchGCSProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	var req UpdateGoogleBatchGCSProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	updated, err := h.settingService.UpdateGoogleBatchGCSProfile(c.Request.Context(), profileID, &service.GoogleBatchGCSProfile{
		Name:               req.Name,
		Enabled:            req.Enabled,
		Bucket:             req.Bucket,
		Prefix:             req.Prefix,
		ProjectID:          req.ProjectID,
		ServiceAccountJSON: req.ServiceAccountJSON,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toGoogleBatchGCSProfileDTO(*updated))
}

func (h *SettingHandler) DeleteGoogleBatchGCSProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	if err := h.settingService.DeleteGoogleBatchGCSProfile(c.Request.Context(), profileID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *SettingHandler) SetActiveGoogleBatchGCSProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	active, err := h.settingService.SetActiveGoogleBatchGCSProfile(c.Request.Context(), profileID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toGoogleBatchGCSProfileDTO(*active))
}

func (h *SettingHandler) TestGoogleBatchGCSConnection(c *gin.Context) {
	var req TestGoogleBatchGCSConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	profile := &service.GoogleBatchGCSProfile{
		ProfileID:          req.ProfileID,
		Enabled:            req.Enabled,
		Bucket:             req.Bucket,
		Prefix:             req.Prefix,
		ProjectID:          req.ProjectID,
		ServiceAccountJSON: req.ServiceAccountJSON,
	}
	if strings.TrimSpace(profile.ServiceAccountJSON) == "" && strings.TrimSpace(req.ProfileID) != "" {
		existing, err := h.settingService.ListGoogleBatchGCSProfiles(c.Request.Context())
		if err == nil {
			if matched := findGoogleBatchGCSProfile(existing.Items, req.ProfileID); matched != nil {
				profile.ServiceAccountJSON = matched.ServiceAccountJSON
				if strings.TrimSpace(profile.Bucket) == "" {
					profile.Bucket = matched.Bucket
				}
				if strings.TrimSpace(profile.ProjectID) == "" {
					profile.ProjectID = matched.ProjectID
				}
				if strings.TrimSpace(profile.Prefix) == "" {
					profile.Prefix = matched.Prefix
				}
				if !req.Enabled {
					profile.Enabled = matched.Enabled
				}
			}
		}
	}
	if err := h.settingService.TestGoogleBatchGCSConnection(c.Request.Context(), profile); err != nil {
		response.Error(c, 400, "Google Batch GCS 连接测试失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Google Batch GCS 连接成功"})
}
