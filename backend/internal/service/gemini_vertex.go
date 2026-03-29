package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

func (a *Account) IsGeminiVertexAI() bool {
	if a == nil || EffectiveProtocol(a) != PlatformGemini || a.Type != AccountTypeOAuth {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(a.GeminiOAuthType()), "vertex_ai")
}

func (a *Account) GetGeminiVertexProjectID() string {
	if a == nil {
		return ""
	}
	return strings.TrimSpace(a.GetCredential("vertex_project_id"))
}

func (a *Account) GetGeminiVertexLocation() string {
	if a == nil {
		return ""
	}
	return strings.TrimSpace(a.GetCredential("vertex_location"))
}

func (a *Account) GetGeminiVertexBaseURL(defaultBaseURL string) string {
	if a == nil {
		return defaultBaseURL
	}
	baseURL := strings.TrimSpace(a.GetCredential("base_url"))
	if baseURL == "" {
		return defaultBaseURL
	}
	return baseURL
}

func (a *Account) GeminiVertexModelsPath() (string, error) {
	projectID := a.GetGeminiVertexProjectID()
	if projectID == "" {
		return "", fmt.Errorf("missing vertex_project_id")
	}
	location := a.GetGeminiVertexLocation()
	if location == "" {
		return "", fmt.Errorf("missing vertex_location")
	}
	return fmt.Sprintf(
		"/v1/projects/%s/locations/%s/publishers/google/models",
		url.PathEscape(projectID),
		url.PathEscape(location),
	), nil
}

func (a *Account) GeminiVertexModelPath(model string) (string, error) {
	modelID := strings.TrimSpace(model)
	if modelID == "" {
		return "", fmt.Errorf("missing model")
	}
	basePath, err := a.GeminiVertexModelsPath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", basePath, url.PathEscape(modelID)), nil
}

func (a *Account) GeminiVertexModelActionPath(model, action string) (string, error) {
	action = strings.TrimSpace(action)
	if action == "" {
		return "", fmt.Errorf("missing action")
	}
	modelPath, err := a.GeminiVertexModelPath(model)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", modelPath, action), nil
}

func buildGeminiVertexGETPath(account *Account, path string) (string, error) {
	if account == nil {
		return "", fmt.Errorf("account is nil")
	}
	path = strings.TrimSpace(path)
	switch {
	case path == "/v1beta/models":
		return account.GeminiVertexModelsPath()
	case strings.HasPrefix(path, "/v1beta/models/"):
		return account.GeminiVertexModelPath(strings.TrimPrefix(path, "/v1beta/models/"))
	default:
		return "", fmt.Errorf("unsupported vertex ai GET path: %s", path)
	}
}

func isGeminiCredentialConfigError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "missing project_id") ||
		strings.Contains(message, "missing vertex_project_id") ||
		strings.Contains(message, "missing vertex_location") ||
		strings.Contains(message, "access_token not found") ||
		strings.Contains(message, "vertex ai access token expired")
}

func geminiBaseURLForLogging(account *Account) string {
	if account == nil {
		return ""
	}
	if account.IsGeminiVertexAI() {
		return account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
	}
	return account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
}
