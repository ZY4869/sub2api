package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"strings"
	"time"
)

const geminiStickySessionTTL = time.Hour
const (
	geminiMaxRetries     = 5
	geminiRetryBaseDelay = 1 * time.Second
	geminiRetryMaxDelay  = 16 * time.Second
)
const geminiDummyThoughtSignature = "skip_thought_signature_validator"

type GeminiMessagesCompatService struct {
	accountRepo               AccountRepository
	groupRepo                 GroupRepository
	cache                     GatewayCache
	schedulerSnapshot         *SchedulerSnapshotService
	tokenProvider             *GeminiTokenProvider
	vertexCatalogService      VertexCatalogProvider
	rateLimitService          *RateLimitService
	httpUpstream              HTTPUpstream
	antigravityGatewayService *AntigravityGatewayService
	cfg                       *config.Config
	responseHeaderFilter      *responseheaders.CompiledHeaderFilter
}

func NewGeminiMessagesCompatService(accountRepo AccountRepository, groupRepo GroupRepository, cache GatewayCache, schedulerSnapshot *SchedulerSnapshotService, tokenProvider *GeminiTokenProvider, rateLimitService *RateLimitService, httpUpstream HTTPUpstream, antigravityGatewayService *AntigravityGatewayService, cfg *config.Config) *GeminiMessagesCompatService {
	return &GeminiMessagesCompatService{accountRepo: accountRepo, groupRepo: groupRepo, cache: cache, schedulerSnapshot: schedulerSnapshot, tokenProvider: tokenProvider, rateLimitService: rateLimitService, httpUpstream: httpUpstream, antigravityGatewayService: antigravityGatewayService, cfg: cfg, responseHeaderFilter: compileResponseHeaderFilter(cfg)}
}

func (s *GeminiMessagesCompatService) SetVertexCatalogService(vertexCatalogService VertexCatalogProvider) {
	s.vertexCatalogService = vertexCatalogService
}
func (s *GeminiMessagesCompatService) SelectAccountForAIStudioEndpoints(ctx context.Context, groupID *int64) (*Account, error) {
	accounts, err := s.listSchedulableAccountsOnce(ctx, groupID, PlatformGemini, true)
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}
	if len(accounts) == 0 {
		return nil, errors.New("no available Gemini accounts")
	}
	rank := func(a *Account) int {
		if a == nil {
			return 999
		}
		switch a.Type {
		case AccountTypeAPIKey:
			if strings.TrimSpace(a.GetCredential("api_key")) != "" {
				if a.IsGeminiVertexExpress() {
					return 1
				}
				return 0
			}
			return 9
		case AccountTypeOAuth:
			if strings.TrimSpace(a.GetCredential("project_id")) == "" {
				if a.IsGeminiVertexAI() {
					return 3
				}
				return 2
			}
			if strings.TrimSpace(a.GetCredential("oauth_type")) == "ai_studio" {
				return 4
			}
			return 5
		default:
			return 10
		}
	}
	var selected *Account
	for i := range accounts {
		acc := &accounts[i]
		if selected == nil {
			selected = acc
			continue
		}
		r1, r2 := rank(acc), rank(selected)
		if r1 < r2 {
			selected = acc
			continue
		}
		if r1 > r2 {
			continue
		}
		if acc.Priority < selected.Priority {
			selected = acc
		} else if acc.Priority == selected.Priority {
			switch {
			case acc.LastUsedAt == nil && selected.LastUsedAt != nil:
				selected = acc
			case acc.LastUsedAt != nil && selected.LastUsedAt == nil:
			case acc.LastUsedAt == nil && selected.LastUsedAt == nil:
				if acc.Type == AccountTypeOAuth && selected.Type != AccountTypeOAuth {
					selected = acc
				}
			default:
				if acc.LastUsedAt.Before(*selected.LastUsedAt) {
					selected = acc
				}
			}
		}
	}
	if selected == nil {
		return nil, errors.New("no available Gemini accounts")
	}
	return selected, nil
}

type geminiStreamResult struct {
	usage        *ClaudeUsage
	firstTokenMs *int
}

func (s *GeminiMessagesCompatService) extractImageSize(body []byte) string {
	var req struct {
		GenerationConfig *struct {
			ImageConfig *struct {
				ImageSize string `json:"imageSize"`
			} `json:"imageConfig"`
		} `json:"generationConfig"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "2K"
	}
	if req.GenerationConfig != nil && req.GenerationConfig.ImageConfig != nil {
		size := strings.ToUpper(strings.TrimSpace(req.GenerationConfig.ImageConfig.ImageSize))
		if size == "1K" || size == "2K" || size == "4K" {
			return size
		}
	}
	return "2K"
}
