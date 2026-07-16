package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/grokoauth"
)

const (
	GrokDeviceStatusPending    = "pending"
	GrokDeviceStatusSlowDown   = "slow_down"
	GrokDeviceStatusAuthorized = "authorized"
	GrokDeviceStatusDenied     = "denied"
	GrokDeviceStatusExpired    = "expired"
)

type GrokStartDeviceFlowInput struct {
	ProxyID *int64
	BaseURL string
}

type GrokDeviceFlowStartResult struct {
	SessionID               string `json:"session_id"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	Interval                int    `json:"interval"`
	ExpiresAt               int64  `json:"expires_at"`
}

type GrokPollDeviceTokenInput struct {
	SessionID string
	ProxyID   *int64
}

type GrokDevicePollResult struct {
	Status       string         `json:"status"`
	Interval     int            `json:"interval"`
	ExpiresAt    int64          `json:"expires_at"`
	TokenInfo    *GrokTokenInfo `json:"token_info,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

func (s *GrokOAuthService) StartDeviceFlow(ctx context.Context, input *GrokStartDeviceFlowInput) (*GrokDeviceFlowStartResult, error) {
	startedAt := time.Now()
	if s == nil || s.oauthClient == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_OAUTH_UNAVAILABLE", "Grok OAuth service is unavailable")
	}
	if input == nil {
		input = &GrokStartDeviceFlowInput{}
	}
	sessionID, err := grokoauth.GenerateSessionID()
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_OAUTH_SESSION_FAILED", "failed to generate Grok OAuth device session").WithCause(err)
	}
	proxyURL, err := s.resolveProxyURL(ctx, input.ProxyID)
	if err != nil {
		return nil, err
	}
	clientID := s.oauthClientID()
	scope := s.oauthScope()
	baseURL := s.oauthBaseURL(input.BaseURL)
	requestID := requestIDFromContext(ctx)

	slog.Info("grok_oauth_device_start", "request_id", requestID, "session_id", sessionID, "has_proxy", proxyURL != "")
	deviceURL, err := s.validatedOAuthDeviceURL()
	if err != nil {
		return nil, err
	}
	deviceResp, err := s.oauthClient.StartDeviceFlow(ctx, deviceURL, clientID, scope, proxyURL)
	if err != nil {
		slog.Warn("grok_oauth_device_start_failed", "request_id", requestID, "session_id", sessionID, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err.Error())
		return nil, err
	}
	if deviceResp == nil || strings.TrimSpace(deviceResp.DeviceCode) == "" || strings.TrimSpace(deviceResp.UserCode) == "" {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_DEVICE_RESPONSE_INVALID", "Grok OAuth device response is missing required fields")
	}

	interval := deviceResp.Interval
	if interval <= 0 {
		interval = 5
	}
	expiresIn := deviceResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = int64(10 * time.Minute / time.Second)
	}
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresIn) * time.Second)
	s.sessionStore.Set(sessionID, &grokoauth.OAuthSession{
		ClientID:                      clientID,
		Scope:                         scope,
		ProxyURL:                      proxyURL,
		BaseURL:                       baseURL,
		CreatedAt:                     now,
		DeviceCode:                    strings.TrimSpace(deviceResp.DeviceCode),
		DeviceUserCode:                strings.TrimSpace(deviceResp.UserCode),
		DeviceVerificationURI:         strings.TrimSpace(deviceResp.VerificationURI),
		DeviceVerificationURIComplete: strings.TrimSpace(deviceResp.VerificationURIComplete),
		DeviceIntervalSeconds:         interval,
		DeviceExpiresAt:               expiresAt,
	})

	slog.Info("grok_oauth_device_started", "request_id", requestID, "session_id", sessionID, "interval", interval, "expires_at", expiresAt.UTC().Format(time.RFC3339), "duration_ms", time.Since(startedAt).Milliseconds())
	return &GrokDeviceFlowStartResult{
		SessionID:               sessionID,
		UserCode:                strings.TrimSpace(deviceResp.UserCode),
		VerificationURI:         strings.TrimSpace(deviceResp.VerificationURI),
		VerificationURIComplete: strings.TrimSpace(deviceResp.VerificationURIComplete),
		Interval:                interval,
		ExpiresAt:               expiresAt.Unix(),
	}, nil
}

func (s *GrokOAuthService) PollDeviceToken(ctx context.Context, input *GrokPollDeviceTokenInput) (*GrokDevicePollResult, error) {
	startedAt := time.Now()
	if s == nil || s.oauthClient == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_OAUTH_UNAVAILABLE", "Grok OAuth service is unavailable")
	}
	if input == nil {
		input = &GrokPollDeviceTokenInput{}
	}
	sessionID := strings.TrimSpace(input.SessionID)
	requestID := requestIDFromContext(ctx)
	pollRead := s.sessionStore.PrepareDevicePoll(sessionID, time.Now())
	if pollRead.Status == grokoauth.DevicePollReadNotFound {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_DEVICE_SESSION_NOT_FOUND", "Grok OAuth device session not found or expired")
	}
	session := pollRead.Session
	if pollRead.Status == grokoauth.DevicePollReadExpired {
		slog.Info("grok_oauth_device_poll", "request_id", requestID, "session_id", sessionID, "status", GrokDeviceStatusExpired, "interval", pollRead.IntervalSeconds, "duration_ms", time.Since(startedAt).Milliseconds())
		return &GrokDevicePollResult{
			Status:       GrokDeviceStatusExpired,
			Interval:     pollRead.IntervalSeconds,
			ExpiresAt:    unixOrZero(pollRead.ExpiresAt),
			ErrorMessage: "Grok OAuth device code expired, generate a new device code in Sub2api and try again",
		}, nil
	}
	if pollRead.Status == grokoauth.DevicePollReadSlowDown {
		slog.Info("grok_oauth_device_poll", "request_id", requestID, "session_id", sessionID, "status", GrokDeviceStatusSlowDown, "interval", pollRead.IntervalSeconds, "remaining_wait_ms", pollRead.RemainingWait.Milliseconds(), "duration_ms", time.Since(startedAt).Milliseconds())
		return &GrokDevicePollResult{
			Status:       GrokDeviceStatusSlowDown,
			Interval:     pollRead.IntervalSeconds,
			ExpiresAt:    unixOrZero(pollRead.ExpiresAt),
			ErrorMessage: "Grok OAuth device polling is too frequent",
		}, nil
	}

	proxyURL := session.ProxyURL
	if input.ProxyID != nil {
		var err error
		proxyURL, err = s.resolveProxyURL(ctx, input.ProxyID)
		if err != nil {
			return nil, err
		}
	}

	tokenURL, err := s.validatedOAuthTokenURL()
	if err != nil {
		return nil, err
	}
	tokenResp, err := s.oauthClient.PollDeviceToken(ctx, tokenURL, session.DeviceCode, session.ClientID, proxyURL)
	if err != nil {
		var deviceErr *grokoauth.DeviceTokenError
		if errors.As(err, &deviceErr) {
			return s.devicePollResultFromError(ctx, sessionID, session, deviceErr, startedAt), nil
		}
		slog.Warn("grok_oauth_device_token_failed", "request_id", requestID, "session_id", sessionID, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err.Error())
		return nil, err
	}

	tokenInfo := s.tokenInfoFromResponse(tokenResp, session.ClientID, session.Scope, session.BaseURL)
	s.enrichUserInfo(ctx, tokenInfo, proxyURL)
	s.sessionStore.Delete(sessionID)
	slog.Info("grok_oauth_device_authorized", "request_id", requestID, "session_id", sessionID, "status", GrokDeviceStatusAuthorized, "duration_ms", time.Since(startedAt).Milliseconds())
	return &GrokDevicePollResult{
		Status:    GrokDeviceStatusAuthorized,
		Interval:  session.DeviceIntervalSeconds,
		ExpiresAt: session.DeviceExpiresAt.Unix(),
		TokenInfo: tokenInfo,
	}, nil
}

func (s *GrokOAuthService) devicePollResultFromError(ctx context.Context, sessionID string, session *grokoauth.OAuthSession, err *grokoauth.DeviceTokenError, startedAt time.Time) *GrokDevicePollResult {
	status := strings.ToLower(strings.TrimSpace(err.Status))
	var resultStatus string
	message := strings.TrimSpace(err.Description)
	switch status {
	case "authorization_pending":
		resultStatus = GrokDeviceStatusPending
	case "slow_down":
		resultStatus = GrokDeviceStatusSlowDown
		session.DeviceIntervalSeconds += 5
	case "access_denied":
		resultStatus = GrokDeviceStatusDenied
		s.sessionStore.Delete(sessionID)
	case "expired_token":
		resultStatus = GrokDeviceStatusExpired
		s.sessionStore.Delete(sessionID)
	default:
		resultStatus = GrokDeviceStatusPending
	}
	if message == "" {
		message = defaultGrokDevicePollMessage(resultStatus)
	}
	slog.Info("grok_oauth_device_poll", "request_id", requestIDFromContext(ctx), "session_id", sessionID, "status", resultStatus, "interval", session.DeviceIntervalSeconds, "duration_ms", time.Since(startedAt).Milliseconds())
	return &GrokDevicePollResult{
		Status:       resultStatus,
		Interval:     session.DeviceIntervalSeconds,
		ExpiresAt:    session.DeviceExpiresAt.Unix(),
		ErrorMessage: message,
	}
}

func unixOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}

func defaultGrokDevicePollMessage(status string) string {
	switch status {
	case GrokDeviceStatusSlowDown:
		return "Grok OAuth device polling is too frequent"
	case GrokDeviceStatusDenied:
		return "Grok OAuth device authorization was denied"
	case GrokDeviceStatusExpired:
		return "Grok OAuth device code expired, generate a new device code in Sub2api and try again"
	default:
		return "Grok OAuth device authorization is still pending"
	}
}
