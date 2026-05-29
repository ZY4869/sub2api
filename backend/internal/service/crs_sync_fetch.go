package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

// fetchCRSExport validates the connection parameters, authenticates with CRS,
// and returns the exported accounts. Shared by SyncFromCRS and PreviewFromCRS.
func (s *CRSSyncService) fetchCRSExport(ctx context.Context, baseURL, username, password string) (*crsExportResponse, error) {
	if s.cfg == nil {
		return nil, errors.New("config is not available")
	}
	normalizedURL, err := validateCRSBaseURL(s.cfg, baseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return nil, errors.New("username and password are required")
	}

	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:            20 * time.Second,
		ValidateResolvedIP: true,
		AllowPrivateHosts:  s.cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return nil, fmt.Errorf("create http client failed: %w", err)
	}

	adminToken, err := crsLogin(ctx, client, normalizedURL, username, password)
	if err != nil {
		return nil, err
	}

	return crsExportAccounts(ctx, client, normalizedURL, adminToken)
}

func validateCRSBaseURL(cfg *config.Config, baseURL string) (string, error) {
	if cfg == nil {
		return "", errors.New("config is not available")
	}
	normalizedURL := strings.TrimSpace(baseURL)
	if cfg.Security.URLAllowlist.Enabled {
		return normalizeBaseURL(normalizedURL, cfg.Security.URLAllowlist.CRSHosts, cfg.Security.URLAllowlist.AllowPrivateHosts)
	}
	normalized, err := urlvalidator.ValidateHTTPURL(normalizedURL, cfg.Security.URLAllowlist.AllowInsecureHTTP, urlvalidator.ValidationOptions{
		AllowPrivate: cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}

func crsLogin(ctx context.Context, client *http.Client, baseURL, username, password string) (string, error) {
	payload := map[string]any{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/web/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("crs login failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var parsed crsLoginResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("crs login parse failed: %w", err)
	}
	if !parsed.Success || strings.TrimSpace(parsed.Token) == "" {
		msg := parsed.Message
		if msg == "" {
			msg = parsed.Error
		}
		if msg == "" {
			msg = "unknown error"
		}
		return "", errors.New("crs login failed: " + msg)
	}
	return parsed.Token, nil
}

func crsExportAccounts(ctx context.Context, client *http.Client, baseURL, adminToken string) (*crsExportResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/admin/sync/export-accounts?include_secrets=true", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("crs export failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var parsed crsExportResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("crs export parse failed: %w", err)
	}
	if !parsed.Success {
		msg := parsed.Message
		if msg == "" {
			msg = parsed.Error
		}
		if msg == "" {
			msg = "unknown error"
		}
		return nil, errors.New("crs export failed: " + msg)
	}
	return &parsed, nil
}
