package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type GrokReverseClient struct {
	httpUpstream HTTPUpstream
	cfg          *config.Config
}

func NewGrokReverseClient(httpUpstream HTTPUpstream, cfg *config.Config) *GrokReverseClient {
	return &GrokReverseClient{
		httpUpstream: httpUpstream,
		cfg:          cfg,
	}
}

func (c *GrokReverseClient) baseURL(account *Account) string {
	if account == nil {
		return defaultGrokReverseBaseURL
	}
	if base := strings.TrimSpace(account.GetCredential("base_url")); base != "" {
		return strings.TrimRight(base, "/")
	}
	return defaultGrokReverseBaseURL
}

func (c *GrokReverseClient) bearerToken(account *Account) string {
	if account == nil {
		return ""
	}
	return strings.TrimSpace(account.GetGrokSSOToken())
}

func (c *GrokReverseClient) do(
	ctx context.Context,
	account *Account,
	method string,
	path string,
	body []byte,
	contentType string,
) (*http.Response, error) {
	if c == nil || c.httpUpstream == nil {
		return nil, fmt.Errorf("grok reverse upstream is not configured")
	}
	token := c.bearerToken(account)
	if token == "" {
		return nil, fmt.Errorf("grok sso token is missing")
	}
	url := c.baseURL(account) + path
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	proxyURL := ""
	if account != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return c.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
}

func (c *GrokReverseClient) doWithTimeout(
	account *Account,
	method string,
	path string,
	body []byte,
	contentType string,
	timeout time.Duration,
) (*http.Response, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.do(ctx, account, method, path, body, contentType)
}

func (c *GrokReverseClient) DoChat(ctx context.Context, account *Account, path string, body []byte, contentType string) (*http.Response, error) {
	return c.do(ctx, account, http.MethodPost, path, body, contentType)
}

func (c *GrokReverseClient) DoResponses(ctx context.Context, account *Account, method string, path string, body []byte, contentType string) (*http.Response, error) {
	return c.do(ctx, account, method, path, body, contentType)
}

func (c *GrokReverseClient) DoAppChat(ctx context.Context, account *Account, body []byte) (*http.Response, error) {
	return c.do(ctx, account, http.MethodPost, "/rest/app-chat/conversations/new", body, "application/json")
}

func (c *GrokReverseClient) ProbeConversation(account *Account, conversationID string, timeout time.Duration) (*http.Response, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, fmt.Errorf("conversation_id is required")
	}
	path := "/rest/app-chat/conversations_v2/" + strings.TrimSpace(conversationID) + "?includeWorkspaces=true&includeTaskResult=true"
	return c.doWithTimeout(account, http.MethodGet, path, nil, "", timeout)
}

func (c *GrokReverseClient) ProbeAsset(account *Account, assetID string, timeout time.Duration) (*http.Response, error) {
	if strings.TrimSpace(assetID) == "" {
		return nil, fmt.Errorf("asset id is required")
	}
	return c.doWithTimeout(account, http.MethodGet, "/rest/app-chat/assets/"+strings.TrimSpace(assetID), nil, "", timeout)
}
