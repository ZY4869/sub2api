package kiro

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	SocialAuthEndpoint = "https://prod.us-east-1.auth.desktop.kiro.dev"
	DefaultRedirectURI = "http://127.0.0.1:19877/oauth/callback"
	BuilderIDStartURL  = "https://view.awsapps.com/start"
	DefaultIDCRegion   = "us-east-1"
	OIDCScopes         = "codewhisperer:completions,codewhisperer:analysis,codewhisperer:conversations,codewhisperer:transformations,codewhisperer:taskassist"
	SessionTTL         = 15 * time.Minute
)

const (
	OAuthMethodGitHub  = "github"
	OAuthMethodGoogle  = "google"
	OAuthMethodBuilder = "builder_id"
	OAuthMethodIDC     = "idc"
)

type OAuthSession struct {
	State        string
	CodeVerifier string
	RedirectURI  string
	ProxyURL     string
	Method       string
	Region       string
	StartURL     string
	ClientID     string
	ClientSecret string
	CreatedAt    time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*OAuthSession
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*OAuthSession),
	}
}

func (s *SessionStore) Set(sessionID string, session *OAuthSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session
}

func (s *SessionStore) Get(sessionID string) (*OAuthSession, bool) {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok || session == nil {
		return nil, false
	}
	if time.Since(session.CreatedAt) > SessionTTL {
		s.Delete(sessionID)
		return nil, false
	}
	return session, true
}

func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func GenerateState() (string, error) {
	return generateHex(32)
}

func GenerateSessionID() (string, error) {
	return generateHex(16)
}

func GenerateCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GenerateCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func NormalizeOAuthMethod(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case OAuthMethodGoogle:
		return OAuthMethodGoogle
	case OAuthMethodBuilder, "builder-id":
		return OAuthMethodBuilder
	case OAuthMethodIDC:
		return OAuthMethodIDC
	default:
		return OAuthMethodGitHub
	}
}

func BuildSocialAuthURL(method, redirectURI, codeChallenge, state string) (string, error) {
	idp := socialIDP(method)
	if idp == "" {
		return "", fmt.Errorf("unsupported kiro social oauth method: %s", method)
	}
	params := url.Values{}
	params.Set("idp", idp)
	params.Set("redirect_uri", redirectURI)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", state)
	params.Set("prompt", "select_account")
	return SocialAuthEndpoint + "/login?" + params.Encode(), nil
}

func BuildOIDCAuthURL(region, clientID, redirectURI, state, codeChallenge string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", strings.TrimSpace(clientID))
	params.Set("redirect_uri", redirectURI)
	params.Set("scopes", OIDCScopes)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	return OIDCEndpoint(region) + "/authorize?" + params.Encode()
}

func OIDCEndpoint(region string) string {
	normalized := strings.TrimSpace(region)
	if normalized == "" {
		normalized = DefaultIDCRegion
	}
	return fmt.Sprintf("https://oidc.%s.amazonaws.com", normalized)
}

func ResolveRedirectURI(redirectURI string) (string, error) {
	trimmed := strings.TrimSpace(redirectURI)
	if trimmed == "" {
		trimmed = DefaultRedirectURI
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("kiro oauth redirect_uri must be an absolute http loopback URL")
	}
	if !strings.EqualFold(parsed.Scheme, "http") {
		return "", fmt.Errorf("kiro oauth redirect_uri must use an http loopback URL")
	}
	host, err := normalizeLoopbackHost(parsed.Hostname())
	if err != nil {
		return "", err
	}
	parsed.Host = buildRedirectHost(host, parsed.Port())
	return parsed.String(), nil
}

func normalizeLoopbackHost(host string) (string, error) {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return "", fmt.Errorf("kiro oauth redirect_uri must use a loopback interface such as http://127.0.0.1:19877/oauth/callback")
	}
	if strings.EqualFold(trimmed, "localhost") {
		return "127.0.0.1", nil
	}
	ip := net.ParseIP(trimmed)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf("kiro oauth redirect_uri must use a loopback interface such as http://127.0.0.1:19877/oauth/callback")
	}
	return trimmed, nil
}

func buildRedirectHost(host string, port string) string {
	if port == "" {
		if strings.Contains(host, ":") {
			return "[" + host + "]"
		}
		return host
	}
	return net.JoinHostPort(host, port)
}

func socialIDP(method string) string {
	switch NormalizeOAuthMethod(method) {
	case OAuthMethodGoogle:
		return "Google"
	case OAuthMethodGitHub:
		return "Github"
	default:
		return ""
	}
}

func generateHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
