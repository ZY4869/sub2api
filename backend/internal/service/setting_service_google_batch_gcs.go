package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
)

type googleBatchGCSProfilesStore struct {
	ActiveProfileID string                           `json:"active_profile_id"`
	Items           []googleBatchGCSProfileStoreItem `json:"items"`
}

type googleBatchGCSProfileStoreItem struct {
	ProfileID          string `json:"profile_id"`
	Name               string `json:"name"`
	Enabled            bool   `json:"enabled"`
	Bucket             string `json:"bucket"`
	Prefix             string `json:"prefix"`
	ProjectID          string `json:"project_id"`
	ServiceAccountJSON string `json:"service_account_json"`
	UpdatedAt          string `json:"updated_at"`
}

func (s *SettingService) ListGoogleBatchGCSProfiles(ctx context.Context) (*GoogleBatchGCSProfileList, error) {
	store, err := s.loadGoogleBatchGCSProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	return convertGoogleBatchGCSProfilesStore(store), nil
}

func (s *SettingService) CreateGoogleBatchGCSProfile(ctx context.Context, profile *GoogleBatchGCSProfile, setActive bool) (*GoogleBatchGCSProfile, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}
	profileID := strings.TrimSpace(profile.ProfileID)
	if profileID == "" {
		return nil, infraerrors.BadRequest("GOOGLE_BATCH_GCS_PROFILE_ID_REQUIRED", "profile_id is required")
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("GOOGLE_BATCH_GCS_PROFILE_NAME_REQUIRED", "name is required")
	}
	store, err := s.loadGoogleBatchGCSProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	if hasGoogleBatchGCSProfileID(store.Items, profileID) {
		return nil, ErrGoogleBatchGCSProfileExists
	}
	now := time.Now().UTC().Format(time.RFC3339)
	store.Items = append(store.Items, googleBatchGCSProfileStoreItem{
		ProfileID:          profileID,
		Name:               name,
		Enabled:            profile.Enabled,
		Bucket:             strings.TrimSpace(profile.Bucket),
		Prefix:             strings.TrimSpace(profile.Prefix),
		ProjectID:          strings.TrimSpace(profile.ProjectID),
		ServiceAccountJSON: strings.TrimSpace(profile.ServiceAccountJSON),
		UpdatedAt:          now,
	})
	if setActive || store.ActiveProfileID == "" {
		store.ActiveProfileID = profileID
	}
	if err := s.persistGoogleBatchGCSProfilesStore(ctx, store); err != nil {
		return nil, err
	}
	result := convertGoogleBatchGCSProfilesStore(store)
	created := findGoogleBatchGCSProfileByID(result.Items, profileID)
	if created == nil {
		return nil, ErrGoogleBatchGCSProfileNotFound
	}
	return created, nil
}

func (s *SettingService) UpdateGoogleBatchGCSProfile(ctx context.Context, profileID string, profile *GoogleBatchGCSProfile) (*GoogleBatchGCSProfile, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}
	targetID := strings.TrimSpace(profileID)
	if targetID == "" {
		return nil, infraerrors.BadRequest("GOOGLE_BATCH_GCS_PROFILE_ID_REQUIRED", "profile_id is required")
	}
	store, err := s.loadGoogleBatchGCSProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	idx := findGoogleBatchGCSProfileIndex(store.Items, targetID)
	if idx < 0 {
		return nil, ErrGoogleBatchGCSProfileNotFound
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("GOOGLE_BATCH_GCS_PROFILE_NAME_REQUIRED", "name is required")
	}
	target := store.Items[idx]
	target.Name = name
	target.Enabled = profile.Enabled
	target.Bucket = strings.TrimSpace(profile.Bucket)
	target.Prefix = strings.TrimSpace(profile.Prefix)
	target.ProjectID = strings.TrimSpace(profile.ProjectID)
	if trimmed := strings.TrimSpace(profile.ServiceAccountJSON); trimmed != "" {
		target.ServiceAccountJSON = trimmed
	}
	target.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	store.Items[idx] = target
	if err := s.persistGoogleBatchGCSProfilesStore(ctx, store); err != nil {
		return nil, err
	}
	result := convertGoogleBatchGCSProfilesStore(store)
	updated := findGoogleBatchGCSProfileByID(result.Items, targetID)
	if updated == nil {
		return nil, ErrGoogleBatchGCSProfileNotFound
	}
	return updated, nil
}

func (s *SettingService) DeleteGoogleBatchGCSProfile(ctx context.Context, profileID string) error {
	targetID := strings.TrimSpace(profileID)
	if targetID == "" {
		return infraerrors.BadRequest("GOOGLE_BATCH_GCS_PROFILE_ID_REQUIRED", "profile_id is required")
	}
	store, err := s.loadGoogleBatchGCSProfilesStore(ctx)
	if err != nil {
		return err
	}
	idx := findGoogleBatchGCSProfileIndex(store.Items, targetID)
	if idx < 0 {
		return ErrGoogleBatchGCSProfileNotFound
	}
	store.Items = append(store.Items[:idx], store.Items[idx+1:]...)
	if store.ActiveProfileID == targetID {
		store.ActiveProfileID = ""
		if len(store.Items) > 0 {
			store.ActiveProfileID = store.Items[0].ProfileID
		}
	}
	return s.persistGoogleBatchGCSProfilesStore(ctx, store)
}

func (s *SettingService) SetActiveGoogleBatchGCSProfile(ctx context.Context, profileID string) (*GoogleBatchGCSProfile, error) {
	targetID := strings.TrimSpace(profileID)
	if targetID == "" {
		return nil, infraerrors.BadRequest("GOOGLE_BATCH_GCS_PROFILE_ID_REQUIRED", "profile_id is required")
	}
	store, err := s.loadGoogleBatchGCSProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	idx := findGoogleBatchGCSProfileIndex(store.Items, targetID)
	if idx < 0 {
		return nil, ErrGoogleBatchGCSProfileNotFound
	}
	store.ActiveProfileID = targetID
	store.Items[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.persistGoogleBatchGCSProfilesStore(ctx, store); err != nil {
		return nil, err
	}
	result := convertGoogleBatchGCSProfilesStore(store)
	active := pickActiveGoogleBatchGCSProfile(result.Items, result.ActiveProfileID)
	if active == nil {
		return nil, ErrGoogleBatchGCSProfileNotFound
	}
	return active, nil
}

func (s *SettingService) GetActiveGoogleBatchGCSProfile(ctx context.Context) (*GoogleBatchGCSProfile, error) {
	result, err := s.ListGoogleBatchGCSProfiles(ctx)
	if err != nil {
		return nil, err
	}
	return pickActiveGoogleBatchGCSProfile(result.Items, result.ActiveProfileID), nil
}

func (s *SettingService) TestGoogleBatchGCSConnection(ctx context.Context, profile *GoogleBatchGCSProfile) error {
	if profile == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	if !profile.Enabled {
		return fmt.Errorf("google batch gcs profile is disabled")
	}
	if strings.TrimSpace(profile.Bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	creds, err := parseVertexServiceAccountCredentials(strings.TrimSpace(profile.ServiceAccountJSON))
	if err != nil {
		return err
	}
	assertion, err := buildVertexServiceAccountAssertion(creds, time.Now())
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("grant_type", vertexServiceAccountTokenPath)
	form.Set("assertion", assertion)
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:               20 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ValidateResolvedIP:    true,
	})
	if err != nil {
		return fmt.Errorf("build google batch gcs http client: %w", err)
	}
	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, creds.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build token request: %w", err)
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return fmt.Errorf("request gcs access token: %w", err)
	}
	defer func() { _ = tokenResp.Body.Close() }()
	tokenBody, _ := io.ReadAll(io.LimitReader(tokenResp.Body, 1<<20))
	if tokenResp.StatusCode < http.StatusOK || tokenResp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("gcs token exchange failed with status %d", tokenResp.StatusCode)
	}
	var tokenPayload vertexServiceAccountTokenResponse
	if err := json.Unmarshal(tokenBody, &tokenPayload); err != nil {
		return fmt.Errorf("parse gcs access token: %w", err)
	}
	accessToken := strings.TrimSpace(tokenPayload.AccessToken)
	if accessToken == "" {
		return fmt.Errorf("gcs access token is empty")
	}
	bucketURL := "https://storage.googleapis.com/storage/v1/b/" + url.PathEscape(strings.TrimSpace(profile.Bucket))
	bucketReq, err := http.NewRequestWithContext(ctx, http.MethodGet, bucketURL, nil)
	if err != nil {
		return fmt.Errorf("build bucket request: %w", err)
	}
	bucketReq.Header.Set("Authorization", "Bearer "+accessToken)
	bucketResp, err := client.Do(bucketReq)
	if err != nil {
		return fmt.Errorf("request bucket metadata: %w", err)
	}
	defer func() { _ = bucketResp.Body.Close() }()
	if bucketResp.StatusCode < http.StatusOK || bucketResp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(bucketResp.Body, 1<<20))
		return fmt.Errorf("bucket metadata request failed with status %d: %s", bucketResp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *SettingService) loadGoogleBatchGCSProfilesStore(ctx context.Context) (*googleBatchGCSProfilesStore, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyGoogleBatchGCSProfiles)
	if err != nil {
		if err == ErrSettingNotFound {
			return &googleBatchGCSProfilesStore{}, nil
		}
		return nil, fmt.Errorf("get google batch gcs profiles: %w", err)
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return &googleBatchGCSProfilesStore{}, nil
	}
	var store googleBatchGCSProfilesStore
	if err := json.Unmarshal([]byte(trimmed), &store); err != nil {
		return nil, fmt.Errorf("unmarshal google batch gcs profiles: %w", err)
	}
	normalized := normalizeGoogleBatchGCSProfilesStore(store)
	return &normalized, nil
}

func (s *SettingService) persistGoogleBatchGCSProfilesStore(ctx context.Context, store *googleBatchGCSProfilesStore) error {
	if store == nil {
		return fmt.Errorf("google batch gcs profiles store cannot be nil")
	}
	normalized := normalizeGoogleBatchGCSProfilesStore(*store)
	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal google batch gcs profiles: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyGoogleBatchGCSProfiles, string(data)); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func normalizeGoogleBatchGCSProfilesStore(store googleBatchGCSProfilesStore) googleBatchGCSProfilesStore {
	seen := make(map[string]struct{}, len(store.Items))
	normalized := googleBatchGCSProfilesStore{
		ActiveProfileID: strings.TrimSpace(store.ActiveProfileID),
		Items:           make([]googleBatchGCSProfileStoreItem, 0, len(store.Items)),
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for idx := range store.Items {
		item := store.Items[idx]
		item.ProfileID = strings.TrimSpace(item.ProfileID)
		if item.ProfileID == "" {
			item.ProfileID = fmt.Sprintf("profile-%d", idx+1)
		}
		if _, exists := seen[item.ProfileID]; exists {
			continue
		}
		seen[item.ProfileID] = struct{}{}
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			item.Name = item.ProfileID
		}
		item.Bucket = strings.TrimSpace(item.Bucket)
		item.Prefix = strings.Trim(strings.TrimSpace(item.Prefix), "/")
		item.ProjectID = strings.TrimSpace(item.ProjectID)
		item.ServiceAccountJSON = strings.TrimSpace(item.ServiceAccountJSON)
		item.UpdatedAt = strings.TrimSpace(item.UpdatedAt)
		if item.UpdatedAt == "" {
			item.UpdatedAt = now
		}
		normalized.Items = append(normalized.Items, item)
	}
	if findGoogleBatchGCSProfileIndex(normalized.Items, normalized.ActiveProfileID) < 0 {
		normalized.ActiveProfileID = ""
		if len(normalized.Items) > 0 {
			normalized.ActiveProfileID = normalized.Items[0].ProfileID
		}
	}
	return normalized
}

func convertGoogleBatchGCSProfilesStore(store *googleBatchGCSProfilesStore) *GoogleBatchGCSProfileList {
	if store == nil {
		return &GoogleBatchGCSProfileList{}
	}
	items := make([]GoogleBatchGCSProfile, 0, len(store.Items))
	for _, item := range store.Items {
		items = append(items, GoogleBatchGCSProfile{
			ProfileID:                    item.ProfileID,
			Name:                         item.Name,
			IsActive:                     item.ProfileID == store.ActiveProfileID,
			Enabled:                      item.Enabled,
			Bucket:                       item.Bucket,
			Prefix:                       item.Prefix,
			ProjectID:                    item.ProjectID,
			ServiceAccountJSON:           item.ServiceAccountJSON,
			ServiceAccountJSONConfigured: strings.TrimSpace(item.ServiceAccountJSON) != "",
			UpdatedAt:                    item.UpdatedAt,
		})
	}
	return &GoogleBatchGCSProfileList{
		ActiveProfileID: store.ActiveProfileID,
		Items:           items,
	}
}

func hasGoogleBatchGCSProfileID(items []googleBatchGCSProfileStoreItem, profileID string) bool {
	return findGoogleBatchGCSProfileIndex(items, profileID) >= 0
}

func findGoogleBatchGCSProfileIndex(items []googleBatchGCSProfileStoreItem, profileID string) int {
	target := strings.TrimSpace(profileID)
	for idx := range items {
		if items[idx].ProfileID == target {
			return idx
		}
	}
	return -1
}

func findGoogleBatchGCSProfileByID(items []GoogleBatchGCSProfile, profileID string) *GoogleBatchGCSProfile {
	target := strings.TrimSpace(profileID)
	for idx := range items {
		if items[idx].ProfileID == target {
			return &items[idx]
		}
	}
	return nil
}

func pickActiveGoogleBatchGCSProfile(items []GoogleBatchGCSProfile, profileID string) *GoogleBatchGCSProfile {
	return findGoogleBatchGCSProfileByID(items, profileID)
}
