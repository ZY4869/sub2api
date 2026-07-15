package grokoauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	DefaultAuthorizeURL = "https://auth.x.ai/oauth2/authorize"
	DefaultDeviceURL    = "https://auth.x.ai/oauth2/device/code"
	DefaultTokenURL     = "https://auth.x.ai/oauth2/token"
	DefaultUserInfoURL  = "https://auth.x.ai/oauth2/userinfo"
	DefaultClientID     = "b1a00492-073a-47ea-816f-4c329264a828"
	DefaultScope        = "openid profile email offline_access grok-cli:access api:access conversations:read conversations:write"
	DefaultRedirectURI  = "http://127.0.0.1:56121/callback"
	DefaultBaseURL      = "https://cli-chat-proxy.grok.com/v1"
	SessionTTL          = 30 * time.Minute
	DeviceGrantType     = "urn:ietf:params:oauth:grant-type:device_code"
)

type OAuthSession struct {
	State        string
	CodeVerifier string
	ClientID     string
	Scope        string
	RedirectURI  string
	ProxyURL     string
	BaseURL      string
	CreatedAt    time.Time

	DeviceCode                    string
	DeviceUserCode                string
	DeviceVerificationURI         string
	DeviceVerificationURIComplete string
	DeviceIntervalSeconds         int
	DeviceExpiresAt               time.Time
	DeviceLastPollAt              time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*OAuthSession
	stopOnce sync.Once
	stopCh   chan struct{}
}

type DevicePollReadStatus string

const (
	DevicePollReadNotFound DevicePollReadStatus = "not_found"
	DevicePollReadExpired  DevicePollReadStatus = "expired"
	DevicePollReadSlowDown DevicePollReadStatus = "slow_down"
	DevicePollReadReady    DevicePollReadStatus = "ready"
)

type DevicePollReadResult struct {
	Session         *OAuthSession
	Status          DevicePollReadStatus
	RemainingWait   time.Duration
	IntervalSeconds int
	ExpiresAt       time.Time
}

func NewSessionStore() *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*OAuthSession),
		stopCh:   make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *SessionStore) Set(sessionID string, session *OAuthSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session
}

func (s *SessionStore) Get(sessionID string) (*OAuthSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	if session.expired(time.Now()) {
		return nil, false
	}
	return session, true
}

func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *SessionStore) PrepareDevicePoll(sessionID string, now time.Time) DevicePollReadResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok || session == nil || strings.TrimSpace(session.DeviceCode) == "" {
		return DevicePollReadResult{Status: DevicePollReadNotFound}
	}
	interval := session.DeviceIntervalSeconds
	if interval <= 0 {
		interval = 5
		session.DeviceIntervalSeconds = interval
	}
	result := DevicePollReadResult{
		Session:         session,
		IntervalSeconds: interval,
		ExpiresAt:       session.DeviceExpiresAt,
	}
	if session.expired(now) {
		delete(s.sessions, sessionID)
		result.Status = DevicePollReadExpired
		return result
	}
	if !session.DeviceLastPollAt.IsZero() {
		nextAllowed := session.DeviceLastPollAt.Add(time.Duration(interval) * time.Second)
		if now.Before(nextAllowed) {
			result.Status = DevicePollReadSlowDown
			result.RemainingWait = nextAllowed.Sub(now)
			return result
		}
	}
	session.DeviceLastPollAt = now
	result.Status = DevicePollReadReady
	return result
}

func (s *SessionStore) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			for id, session := range s.sessions {
				if session.expired(time.Now()) {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

func GenerateState() (string, error) {
	bytes, err := randomBytes(32)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateSessionID() (string, error) {
	bytes, err := randomBytes(16)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateCodeVerifier() (string, error) {
	bytes, err := randomBytes(32)
	if err != nil {
		return "", err
	}
	return base64URLEncode(bytes), nil
}

func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64URLEncode(hash[:])
}

func BuildAuthorizationURL(authorizeURL, clientID, scope, redirectURI, state, codeChallenge string) (string, error) {
	authorizeURL = strings.TrimSpace(authorizeURL)
	if authorizeURL == "" {
		authorizeURL = DefaultAuthorizeURL
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		clientID = DefaultClientID
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = DefaultScope
	}
	redirectURI = strings.TrimSpace(redirectURI)
	if redirectURI == "" {
		redirectURI = DefaultRedirectURI
	}

	parsed, err := url.Parse(authorizeURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid authorize url")
	}
	params := parsed.Query()
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scope)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("plan", "generic")
	params.Set("referrer", "sub2api")
	parsed.RawQuery = params.Encode()
	return parsed.String(), nil
}

type AuthorizationInputKind string

const (
	AuthorizationInputUnknown        AuthorizationInputKind = "unknown"
	AuthorizationInputCallbackURL    AuthorizationInputKind = "callback_url"
	AuthorizationInputQueryString    AuthorizationInputKind = "query_string"
	AuthorizationInputBareAuthCode   AuthorizationInputKind = "bare_auth_code"
	AuthorizationInputDeviceUserCode AuthorizationInputKind = "device_user_code"
	AuthorizationInputDeviceURL      AuthorizationInputKind = "device_url"
)

type AuthorizationInput struct {
	Kind          AuthorizationInputKind
	Code          string
	State         string
	UserCode      string
	RequiresState bool
}

func ParseAuthorizationInput(raw string) AuthorizationInput {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return AuthorizationInput{Kind: AuthorizationInputUnknown}
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed != nil {
		if parsed.Scheme != "" && parsed.Host != "" {
			if code := strings.TrimSpace(parsed.Query().Get("code")); code != "" {
				return AuthorizationInput{
					Kind:          AuthorizationInputCallbackURL,
					Code:          code,
					State:         strings.TrimSpace(parsed.Query().Get("state")),
					RequiresState: true,
				}
			}
			if userCode := normalizeDeviceUserCode(parsed.Query().Get("user_code")); userCode != "" {
				return AuthorizationInput{Kind: AuthorizationInputDeviceURL, UserCode: userCode}
			}
		}
	}
	queryCandidate := strings.TrimPrefix(trimmed, "?")
	if strings.Contains(queryCandidate, "=") {
		if values, err := url.ParseQuery(queryCandidate); err == nil {
			if code := strings.TrimSpace(values.Get("code")); code != "" {
				return AuthorizationInput{
					Kind:          AuthorizationInputQueryString,
					Code:          code,
					State:         strings.TrimSpace(values.Get("state")),
					RequiresState: true,
				}
			}
			if userCode := normalizeDeviceUserCode(values.Get("user_code")); userCode != "" {
				return AuthorizationInput{Kind: AuthorizationInputDeviceURL, UserCode: userCode}
			}
		}
	}
	if userCode := normalizeDeviceUserCode(trimmed); userCode != "" {
		return AuthorizationInput{Kind: AuthorizationInputDeviceUserCode, UserCode: userCode}
	}
	return AuthorizationInput{Kind: AuthorizationInputBareAuthCode, Code: trimmed}
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int    `json:"interval,omitempty"`
}

type DeviceTokenError struct {
	Status      string
	Description string
}

func (e *DeviceTokenError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Description) != "" {
		return strings.TrimSpace(e.Description)
	}
	return strings.TrimSpace(e.Status)
}

func (s *OAuthSession) expired(now time.Time) bool {
	if s == nil {
		return true
	}
	if !s.DeviceExpiresAt.IsZero() && !now.Before(s.DeviceExpiresAt) {
		return true
	}
	return now.Sub(s.CreatedAt) > SessionTTL
}

type UserInfo struct {
	Sub           string `json:"sub,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Picture       string `json:"picture,omitempty"`
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func normalizeDeviceUserCode(value string) string {
	raw := strings.TrimSpace(value)
	trimmed := strings.ToUpper(raw)
	if trimmed == "" {
		return ""
	}
	for _, r := range trimmed {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return ""
		}
	}
	parts := strings.Split(trimmed, "-")
	if len(parts) == 1 {
		if len(parts[0]) != 8 {
			return ""
		}
	} else {
		for _, part := range parts {
			if len(part) != 4 {
				return ""
			}
		}
	}
	return trimmed
}
