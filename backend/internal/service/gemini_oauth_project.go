package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *GeminiOAuthService) fetchProjectID(ctx context.Context, accessToken, proxyURL string) (string, string, error) {
	if s.codeAssist == nil {
		return "", "", errors.New("code assist client not configured")
	}

	loadResp, loadErr := s.codeAssist.LoadCodeAssist(ctx, accessToken, proxyURL, nil)

	// Extract tierID from response (works whether CloudAICompanionProject is set or not)
	tierID := "LEGACY"
	if loadResp != nil {
		// First try to get tier from currentTier/paidTier fields
		if tier := loadResp.GetTier(); tier != "" {
			tierID = tier
		} else {
			// Fallback to extracting from allowedTiers
			tierID = extractTierIDFromAllowedTiers(loadResp.AllowedTiers)
		}
	}

	// If LoadCodeAssist returned a project, use it
	if loadErr == nil && loadResp != nil && strings.TrimSpace(loadResp.CloudAICompanionProject) != "" {
		return strings.TrimSpace(loadResp.CloudAICompanionProject), tierID, nil
	}

	// 关键逻辑：对齐 Gemini CLI 对“已注册用户”的处理方式。
	// 当 LoadCodeAssist 返回了 currentTier / paidTier（表示账号已注册）但没有返回 cloudaicompanionProject 时：
	// - 不要再调用 onboardUser（通常不会再分配 project_id，且可能触发 INVALID_ARGUMENT）
	// - 先尝试从 Cloud Resource Manager 获取可用项目；仍失败则提示用户手动填写 project_id
	if loadResp != nil {
		registeredTierID := strings.TrimSpace(loadResp.GetTier())
		if registeredTierID != "" {
			// 已注册但未返回 cloudaicompanionProject，这在 Google One 用户中较常见：需要用户自行提供 project_id。
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] User has tier (%s) but no cloudaicompanionProject, trying Cloud Resource Manager...", registeredTierID)

			// Try to get project from Cloud Resource Manager
			fallback, fbErr := fetchProjectIDFromResourceManager(ctx, accessToken, proxyURL)
			if fbErr == nil && strings.TrimSpace(fallback) != "" {
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Found project from Cloud Resource Manager: %s", fallback)
				return strings.TrimSpace(fallback), tierID, nil
			}

			// No project found - user must provide project_id manually
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] No project found from Cloud Resource Manager, user must provide project_id manually")
			return "", tierID, fmt.Errorf("user is registered (tier: %s) but no project_id available. Please provide Project ID manually in the authorization form, or create a project at https://console.cloud.google.com", registeredTierID)
		}
	}

	// 未检测到 currentTier/paidTier，视为新用户，继续调用 onboardUser
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] No currentTier/paidTier found, proceeding with onboardUser (tierID: %s)", tierID)

	req := &geminicli.OnboardUserRequest{
		TierID: tierID,
		Metadata: geminicli.LoadCodeAssistMetadata{
			IDEType:    "ANTIGRAVITY",
			Platform:   "PLATFORM_UNSPECIFIED",
			PluginType: "GEMINI",
		},
	}

	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := s.codeAssist.OnboardUser(ctx, accessToken, proxyURL, req)
		if err != nil {
			// If Code Assist onboarding fails (e.g. INVALID_ARGUMENT), fallback to Cloud Resource Manager projects.
			fallback, fbErr := fetchProjectIDFromResourceManager(ctx, accessToken, proxyURL)
			if fbErr == nil && strings.TrimSpace(fallback) != "" {
				return strings.TrimSpace(fallback), tierID, nil
			}
			return "", tierID, err
		}
		if resp.Done {
			if resp.Response != nil && resp.Response.CloudAICompanionProject != nil {
				switch v := resp.Response.CloudAICompanionProject.(type) {
				case string:
					return strings.TrimSpace(v), tierID, nil
				case map[string]any:
					if id, ok := v["id"].(string); ok {
						return strings.TrimSpace(id), tierID, nil
					}
				}
			}

			fallback, fbErr := fetchProjectIDFromResourceManager(ctx, accessToken, proxyURL)
			if fbErr == nil && strings.TrimSpace(fallback) != "" {
				return strings.TrimSpace(fallback), tierID, nil
			}
			return "", tierID, errors.New("onboardUser completed but no project_id returned")
		}
		time.Sleep(2 * time.Second)
	}

	fallback, fbErr := fetchProjectIDFromResourceManager(ctx, accessToken, proxyURL)
	if fbErr == nil && strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback), tierID, nil
	}
	if loadErr != nil {
		return "", tierID, fmt.Errorf("loadCodeAssist failed (%v) and onboardUser timeout after %d attempts", loadErr, maxAttempts)
	}
	return "", tierID, fmt.Errorf("onboardUser timeout after %d attempts", maxAttempts)
}

type googleCloudProject struct {
	ProjectID      string `json:"projectId"`
	DisplayName    string `json:"name"`
	LifecycleState string `json:"lifecycleState"`
}

type googleCloudProjectsResponse struct {
	Projects []googleCloudProject `json:"projects"`
}

func fetchProjectIDFromResourceManager(ctx context.Context, accessToken, proxyURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://cloudresourcemanager.googleapis.com/v1/projects", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create resource manager request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)

	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:           strings.TrimSpace(proxyURL),
		Timeout:            30 * time.Second,
		ValidateResolvedIP: true,
	})
	if err != nil {
		return "", fmt.Errorf("create http client failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("resource manager request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read resource manager response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resource manager HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var projectsResp googleCloudProjectsResponse
	if err := json.Unmarshal(bodyBytes, &projectsResp); err != nil {
		return "", fmt.Errorf("failed to parse resource manager response: %w", err)
	}

	active := make([]googleCloudProject, 0, len(projectsResp.Projects))
	for _, p := range projectsResp.Projects {
		if p.LifecycleState == "ACTIVE" && strings.TrimSpace(p.ProjectID) != "" {
			active = append(active, p)
		}
	}
	if len(active) == 0 {
		return "", errors.New("no ACTIVE projects found from resource manager")
	}

	// Prefer likely companion projects first.
	for _, p := range active {
		id := strings.ToLower(strings.TrimSpace(p.ProjectID))
		name := strings.ToLower(strings.TrimSpace(p.DisplayName))
		if strings.Contains(id, "cloud-ai-companion") || strings.Contains(name, "cloud ai companion") || strings.Contains(name, "code assist") {
			return strings.TrimSpace(p.ProjectID), nil
		}
	}
	// Then prefer "default".
	for _, p := range active {
		id := strings.ToLower(strings.TrimSpace(p.ProjectID))
		name := strings.ToLower(strings.TrimSpace(p.DisplayName))
		if strings.Contains(id, "default") || strings.Contains(name, "default") {
			return strings.TrimSpace(p.ProjectID), nil
		}
	}

	return strings.TrimSpace(active[0].ProjectID), nil
}
