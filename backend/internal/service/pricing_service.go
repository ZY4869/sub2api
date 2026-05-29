package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type LiteLLMModelPricing struct {
	Currency                                 string     `json:"currency,omitempty"`
	USDToCNYRate                             float64    `json:"usd_to_cny_rate,omitempty"`
	FXRateDate                               string     `json:"fx_rate_date,omitempty"`
	FXLockedAt                               *time.Time `json:"fx_locked_at,omitempty"`
	InputCostPerToken                        float64    `json:"input_cost_per_token"`
	InputCostPerTokenPriority                float64    `json:"input_cost_per_token_priority"`
	InputTokenThreshold                      int        `json:"input_token_threshold,omitempty"`
	InputCostPerTokenAboveThreshold          float64    `json:"input_cost_per_token_above_threshold,omitempty"`
	InputCostPerTokenPriorityAboveThreshold  float64    `json:"input_cost_per_token_priority_above_threshold,omitempty"`
	OutputCostPerToken                       float64    `json:"output_cost_per_token"`
	OutputCostPerTokenPriority               float64    `json:"output_cost_per_token_priority"`
	OutputTokenThreshold                     int        `json:"output_token_threshold,omitempty"`
	OutputCostPerTokenAboveThreshold         float64    `json:"output_cost_per_token_above_threshold,omitempty"`
	OutputCostPerTokenPriorityAboveThreshold float64    `json:"output_cost_per_token_priority_above_threshold,omitempty"`
	CacheCreationInputTokenCost              float64    `json:"cache_creation_input_token_cost"`
	CacheCreationInputTokenCostAbove1hr      float64    `json:"cache_creation_input_token_cost_above_1hr"`
	CacheReadInputTokenCost                  float64    `json:"cache_read_input_token_cost"`
	CacheReadInputTokenCostPriority          float64    `json:"cache_read_input_token_cost_priority"`
	LongContextInputTokenThreshold           int        `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier           float64    `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier          float64    `json:"long_context_output_cost_multiplier,omitempty"`
	SupportsServiceTier                      bool       `json:"supports_service_tier"`
	LiteLLMProvider                          string     `json:"litellm_provider"`
	Mode                                     string     `json:"mode"`
	SupportsPromptCaching                    bool       `json:"supports_prompt_caching"`
	OutputCostPerImage                       float64    `json:"output_cost_per_image"` // 图片生成模型每张图片价格
	OutputCostPerImagePriority               float64    `json:"output_cost_per_image_priority"`
	OutputCostPerVideoRequest                float64    `json:"output_cost_per_video_request"`
}

// PricingRemoteClient 远程价格数据获取接口
type PricingRemoteClient interface {
	FetchPricingJSON(ctx context.Context, url string) ([]byte, error)
	FetchHashText(ctx context.Context, url string) (string, error)
}

// LiteLLMRawEntry 用于解析原始JSON数据
type LiteLLMRawEntry struct {
	Currency                                  string     `json:"currency"`
	USDToCNYRate                              *float64   `json:"usd_to_cny_rate"`
	FXRateDate                                string     `json:"fx_rate_date"`
	FXLockedAt                                *time.Time `json:"fx_locked_at"`
	InputCostPerToken                         *float64   `json:"input_cost_per_token"`
	InputCostPerTokenPriority                 *float64   `json:"input_cost_per_token_priority"`
	InputTokenThreshold                       *int       `json:"input_token_threshold"`
	InputCostPerTokenAboveThreshold           *float64   `json:"input_cost_per_token_above_threshold"`
	InputCostPerTokenAbove200kTokens          *float64   `json:"input_cost_per_token_above_200k_tokens"`
	InputCostPerTokenPriorityAboveThreshold   *float64   `json:"input_cost_per_token_priority_above_threshold"`
	InputCostPerTokenPriorityAbove200kTokens  *float64   `json:"input_cost_per_token_above_200k_tokens_priority"`
	OutputCostPerToken                        *float64   `json:"output_cost_per_token"`
	OutputCostPerTokenPriority                *float64   `json:"output_cost_per_token_priority"`
	OutputTokenThreshold                      *int       `json:"output_token_threshold"`
	OutputCostPerTokenAboveThreshold          *float64   `json:"output_cost_per_token_above_threshold"`
	OutputCostPerTokenAbove200kTokens         *float64   `json:"output_cost_per_token_above_200k_tokens"`
	OutputCostPerTokenPriorityAboveThreshold  *float64   `json:"output_cost_per_token_priority_above_threshold"`
	OutputCostPerTokenPriorityAbove200kTokens *float64   `json:"output_cost_per_token_above_200k_tokens_priority"`
	CacheCreationInputTokenCost               *float64   `json:"cache_creation_input_token_cost"`
	CacheCreationInputTokenCostAbove1hr       *float64   `json:"cache_creation_input_token_cost_above_1hr"`
	CacheReadInputTokenCost                   *float64   `json:"cache_read_input_token_cost"`
	CacheReadInputTokenCostPriority           *float64   `json:"cache_read_input_token_cost_priority"`
	LongContextInputTokenThreshold            *int       `json:"long_context_input_token_threshold"`
	LongContextInputCostMultiplier            *float64   `json:"long_context_input_cost_multiplier"`
	LongContextOutputCostMultiplier           *float64   `json:"long_context_output_cost_multiplier"`
	SupportsServiceTier                       bool       `json:"supports_service_tier"`
	LiteLLMProvider                           string     `json:"litellm_provider"`
	Mode                                      string     `json:"mode"`
	SupportsPromptCaching                     bool       `json:"supports_prompt_caching"`
	OutputCostPerImage                        *float64   `json:"output_cost_per_image"`
	OutputCostPerImagePriority                *float64   `json:"output_cost_per_image_priority"`
	OutputCostPerVideoRequest                 *float64   `json:"output_cost_per_video_request"`
}

// PricingService 动态价格服务
type PricingService struct {
	cfg          *config.Config
	remoteClient PricingRemoteClient
	mu           sync.RWMutex
	pricingData  map[string]*LiteLLMModelPricing
	lastUpdated  time.Time
	localHash    string
	fallbackLogs sync.Map

	// 停止信号
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewPricingService 创建价格服务
func NewPricingService(cfg *config.Config, remoteClient PricingRemoteClient) *PricingService {
	s := &PricingService{
		cfg:          cfg,
		remoteClient: remoteClient,
		pricingData:  make(map[string]*LiteLLMModelPricing),
		stopCh:       make(chan struct{}),
	}
	return s
}

// Initialize 初始化价格服务
func (s *PricingService) Initialize() error {
	// 确保数据目录存在
	if err := os.MkdirAll(s.cfg.Pricing.DataDir, 0755); err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Failed to create data directory: %v", err)
	}

	// 首次加载价格数据
	if err := s.checkAndUpdatePricing(); err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Initial load failed, using fallback: %v", err)
		if err := s.useFallbackPricing(); err != nil {
			return fmt.Errorf("failed to load pricing data: %w", err)
		}
	}

	// 启动定时更新
	s.startUpdateScheduler()

	logger.LegacyPrintf("service.pricing", "[Pricing] Service initialized with %d models", len(s.pricingData))
	return nil
}

// Stop 停止价格服务
func (s *PricingService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Service stopped")
}

// startUpdateScheduler 启动定时更新调度器
