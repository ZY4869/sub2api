package admin

import (
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strings"
)

func toSoraS3SettingsDTO(settings *service.SoraS3Settings) dto.SoraS3Settings {
	if settings == nil {
		return dto.SoraS3Settings{}
	}
	return dto.SoraS3Settings{Enabled: settings.Enabled, Endpoint: settings.Endpoint, Region: settings.Region, Bucket: settings.Bucket, AccessKeyID: settings.AccessKeyID, SecretAccessKeyConfigured: settings.SecretAccessKeyConfigured, Prefix: settings.Prefix, ForcePathStyle: settings.ForcePathStyle, CDNURL: settings.CDNURL, DefaultStorageQuotaBytes: settings.DefaultStorageQuotaBytes}
}
func toSoraS3ProfileDTO(profile service.SoraS3Profile) dto.SoraS3Profile {
	return dto.SoraS3Profile{ProfileID: profile.ProfileID, Name: profile.Name, IsActive: profile.IsActive, Enabled: profile.Enabled, Endpoint: profile.Endpoint, Region: profile.Region, Bucket: profile.Bucket, AccessKeyID: profile.AccessKeyID, SecretAccessKeyConfigured: profile.SecretAccessKeyConfigured, Prefix: profile.Prefix, ForcePathStyle: profile.ForcePathStyle, CDNURL: profile.CDNURL, DefaultStorageQuotaBytes: profile.DefaultStorageQuotaBytes, UpdatedAt: profile.UpdatedAt}
}
func validateSoraS3RequiredWhenEnabled(enabled bool, endpoint, bucket, accessKeyID, secretAccessKey string, hasStoredSecret bool) error {
	if !enabled {
		return nil
	}
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("S3 Endpoint is required when enabled")
	}
	if strings.TrimSpace(bucket) == "" {
		return fmt.Errorf("S3 Bucket is required when enabled")
	}
	if strings.TrimSpace(accessKeyID) == "" {
		return fmt.Errorf("S3 Access Key ID is required when enabled")
	}
	if strings.TrimSpace(secretAccessKey) != "" || hasStoredSecret {
		return nil
	}
	return fmt.Errorf("S3 Secret Access Key is required when enabled")
}
func findSoraS3ProfileByID(items []service.SoraS3Profile, profileID string) *service.SoraS3Profile {
	for idx := range items {
		if items[idx].ProfileID == profileID {
			return &items[idx]
		}
	}
	return nil
}
func (h *SettingHandler) GetSoraS3Settings(c *gin.Context) {
	settings, err := h.settingService.GetSoraS3Settings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toSoraS3SettingsDTO(settings))
}
func (h *SettingHandler) ListSoraS3Profiles(c *gin.Context) {
	result, err := h.settingService.ListSoraS3Profiles(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.SoraS3Profile, 0, len(result.Items))
	for idx := range result.Items {
		items = append(items, toSoraS3ProfileDTO(result.Items[idx]))
	}
	response.Success(c, dto.ListSoraS3ProfilesResponse{ActiveProfileID: result.ActiveProfileID, Items: items})
}

type UpdateSoraS3SettingsRequest struct {
	ProfileID                string `json:"profile_id"`
	Enabled                  bool   `json:"enabled"`
	Endpoint                 string `json:"endpoint"`
	Region                   string `json:"region"`
	Bucket                   string `json:"bucket"`
	AccessKeyID              string `json:"access_key_id"`
	SecretAccessKey          string `json:"secret_access_key"`
	Prefix                   string `json:"prefix"`
	ForcePathStyle           bool   `json:"force_path_style"`
	CDNURL                   string `json:"cdn_url"`
	DefaultStorageQuotaBytes int64  `json:"default_storage_quota_bytes"`
}
type CreateSoraS3ProfileRequest struct {
	ProfileID                string `json:"profile_id"`
	Name                     string `json:"name"`
	SetActive                bool   `json:"set_active"`
	Enabled                  bool   `json:"enabled"`
	Endpoint                 string `json:"endpoint"`
	Region                   string `json:"region"`
	Bucket                   string `json:"bucket"`
	AccessKeyID              string `json:"access_key_id"`
	SecretAccessKey          string `json:"secret_access_key"`
	Prefix                   string `json:"prefix"`
	ForcePathStyle           bool   `json:"force_path_style"`
	CDNURL                   string `json:"cdn_url"`
	DefaultStorageQuotaBytes int64  `json:"default_storage_quota_bytes"`
}
type UpdateSoraS3ProfileRequest struct {
	Name                     string `json:"name"`
	Enabled                  bool   `json:"enabled"`
	Endpoint                 string `json:"endpoint"`
	Region                   string `json:"region"`
	Bucket                   string `json:"bucket"`
	AccessKeyID              string `json:"access_key_id"`
	SecretAccessKey          string `json:"secret_access_key"`
	Prefix                   string `json:"prefix"`
	ForcePathStyle           bool   `json:"force_path_style"`
	CDNURL                   string `json:"cdn_url"`
	DefaultStorageQuotaBytes int64  `json:"default_storage_quota_bytes"`
}

func (h *SettingHandler) CreateSoraS3Profile(c *gin.Context) {
	var req CreateSoraS3ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if strings.TrimSpace(req.Name) == "" {
		response.BadRequest(c, "Name is required")
		return
	}
	if strings.TrimSpace(req.ProfileID) == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, false); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	created, err := h.settingService.CreateSoraS3Profile(c.Request.Context(), &service.SoraS3Profile{ProfileID: req.ProfileID, Name: req.Name, Enabled: req.Enabled, Endpoint: req.Endpoint, Region: req.Region, Bucket: req.Bucket, AccessKeyID: req.AccessKeyID, SecretAccessKey: req.SecretAccessKey, Prefix: req.Prefix, ForcePathStyle: req.ForcePathStyle, CDNURL: req.CDNURL, DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes}, req.SetActive)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toSoraS3ProfileDTO(*created))
}
func (h *SettingHandler) UpdateSoraS3Profile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	var req UpdateSoraS3ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if strings.TrimSpace(req.Name) == "" {
		response.BadRequest(c, "Name is required")
		return
	}
	existingList, err := h.settingService.ListSoraS3Profiles(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	existing := findSoraS3ProfileByID(existingList.Items, profileID)
	if existing == nil {
		response.ErrorFrom(c, service.ErrSoraS3ProfileNotFound)
		return
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, existing.SecretAccessKeyConfigured); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updated, updateErr := h.settingService.UpdateSoraS3Profile(c.Request.Context(), profileID, &service.SoraS3Profile{Name: req.Name, Enabled: req.Enabled, Endpoint: req.Endpoint, Region: req.Region, Bucket: req.Bucket, AccessKeyID: req.AccessKeyID, SecretAccessKey: req.SecretAccessKey, Prefix: req.Prefix, ForcePathStyle: req.ForcePathStyle, CDNURL: req.CDNURL, DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes})
	if updateErr != nil {
		response.ErrorFrom(c, updateErr)
		return
	}
	response.Success(c, toSoraS3ProfileDTO(*updated))
}
func (h *SettingHandler) DeleteSoraS3Profile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	if err := h.settingService.DeleteSoraS3Profile(c.Request.Context(), profileID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *SettingHandler) SetActiveSoraS3Profile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	active, err := h.settingService.SetActiveSoraS3Profile(c.Request.Context(), profileID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toSoraS3ProfileDTO(*active))
}
func (h *SettingHandler) UpdateSoraS3Settings(c *gin.Context) {
	var req UpdateSoraS3SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	existing, err := h.settingService.GetSoraS3Settings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, existing.SecretAccessKeyConfigured); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	settings := &service.SoraS3Settings{Enabled: req.Enabled, Endpoint: req.Endpoint, Region: req.Region, Bucket: req.Bucket, AccessKeyID: req.AccessKeyID, SecretAccessKey: req.SecretAccessKey, Prefix: req.Prefix, ForcePathStyle: req.ForcePathStyle, CDNURL: req.CDNURL, DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes}
	if err := h.settingService.SetSoraS3Settings(c.Request.Context(), settings); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updatedSettings, err := h.settingService.GetSoraS3Settings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toSoraS3SettingsDTO(updatedSettings))
}
