package handler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/tidwall/gjson"
)

type geminiLiveUsageState struct {
	mu            sync.Mutex
	usage         service.ClaudeUsage
	mediaType     string
	requestID     string
	upstreamModel string
}

func (s *geminiLiveUsageState) observeServerFrame(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage.InputTokens = maxInt(s.usage.InputTokens, int(gjson.GetBytes(payload, "usageMetadata.promptTokenCount").Int()))
	s.usage.OutputTokens = maxInt(s.usage.OutputTokens, int(gjson.GetBytes(payload, "usageMetadata.responseTokenCount").Int()))
	s.usage.CacheReadInputTokens = maxInt(s.usage.CacheReadInputTokens, int(gjson.GetBytes(payload, "usageMetadata.cachedContentTokenCount").Int()))
	s.requestID = firstNonEmptyHandlerString(
		s.requestID,
		strings.TrimSpace(gjson.GetBytes(payload, "responseId").String()),
		strings.TrimSpace(gjson.GetBytes(payload, "serverContent.turnComplete.responseId").String()),
	)
	s.upstreamModel = firstNonEmptyHandlerString(
		s.upstreamModel,
		normalizeGeminiLiveModelName(gjson.GetBytes(payload, "setupComplete.model").String()),
		normalizeGeminiLiveModelName(gjson.GetBytes(payload, "model").String()),
	)
	if s.mediaType == "" {
		for _, raw := range gjson.GetBytes(payload, "usageMetadata.responseTokensDetails").Array() {
			modality := strings.ToLower(strings.TrimSpace(raw.Get("modality").String()))
			switch modality {
			case "audio":
				s.mediaType = "audio"
			case "image":
				s.mediaType = "image"
			case "video":
				s.mediaType = "video"
			}
			if s.mediaType != "" {
				break
			}
		}
	}
	return strings.TrimSpace(gjson.GetBytes(payload, "sessionResumptionUpdate.newHandle").String())
}

func (s *geminiLiveUsageState) snapshot() (service.ClaudeUsage, string, string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage, s.mediaType, s.requestID, s.upstreamModel
}

func detectGeminiLiveRequestedModel(payload []byte) string {
	return normalizeGeminiLiveModelName(firstNonEmptyHandlerString(
		gjson.GetBytes(payload, "setup.model").String(),
		gjson.GetBytes(payload, "model").String(),
	))
}

func detectGeminiLiveSessionHash(payload []byte) string {
	for _, seed := range []string{
		gjson.GetBytes(payload, "setup.sessionResumption.handle").String(),
		gjson.GetBytes(payload, "setup.session_resumption.handle").String(),
		gjson.GetBytes(payload, "setup.sessionResumptionConfig.handle").String(),
	} {
		trimmed := strings.TrimSpace(seed)
		if trimmed == "" {
			continue
		}
		return service.DeriveSessionHashFromSeed("gemini-live:" + trimmed)
	}
	return ""
}

func normalizeGeminiLiveModelName(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "models/")
	return strings.TrimSpace(trimmed)
}

func geminiLiveAuthTokenProxyRequested(path string) bool {
	normalized := strings.ToLower(strings.Trim(strings.TrimSpace(path), "/"))
	return strings.HasSuffix(normalized, "v1alpha/authtokens") ||
		strings.HasSuffix(normalized, "live/authtokens") ||
		strings.HasSuffix(normalized, "live/auth-tokens") ||
		strings.HasSuffix(normalized, "live/auth-token")
}

func dialGeminiLiveUpstream(ctx context.Context, upstream *service.GeminiLiveUpstream) (*coderws.Conn, http.Header, error) {
	if upstream == nil {
		return nil, nil, errors.New("gemini live upstream is nil")
	}
	options := &coderws.DialOptions{HTTPHeader: upstream.Headers.Clone()}
	if proxyURL := strings.TrimSpace(upstream.ProxyURL); proxyURL != "" {
		parsedProxyURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, nil, err
		}
		options.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy:               http.ProxyURL(parsedProxyURL),
				MaxIdleConns:        16,
				MaxIdleConnsPerHost: 8,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
				ForceAttemptHTTP2:   true,
			},
		}
	}
	conn, resp, err := coderws.Dial(ctx, upstream.URL, options)
	if err != nil {
		if resp != nil {
			return nil, resp.Header, err
		}
		return nil, nil, err
	}
	conn.SetReadLimit(16 * 1024 * 1024)
	if resp == nil {
		return conn, nil, nil
	}
	return conn, resp.Header.Clone(), nil
}

func relayGeminiLiveFrames(ctx context.Context, src *coderws.Conn, dst *coderws.Conn, onFrame func([]byte)) error {
	for {
		msgType, payload, err := src.Read(ctx)
		if err != nil {
			return err
		}
		if onFrame != nil {
			onFrame(payload)
		}
		if err := dst.Write(ctx, msgType, payload); err != nil {
			return err
		}
	}
}

func geminiLiveCloseIsGraceful(err error) bool {
	if err == nil {
		return true
	}
	switch coderws.CloseStatus(err) {
	case -1:
		return false
	case coderws.StatusNormalClosure, coderws.StatusGoingAway:
		return true
	default:
		return false
	}
}

func maxInt(current int, candidate int) int {
	if candidate > current {
		return candidate
	}
	return current
}
