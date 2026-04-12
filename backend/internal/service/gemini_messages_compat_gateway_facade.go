package service

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *GeminiMessagesCompatService) Forward(ctx context.Context, c *gin.Context, account *Account, body []byte) (*ForwardResult, error) {
	if s == nil {
		return nil, nil
	}
	return NewGeminiCompatGatewayService(s).Forward(ctx, c, account, body)
}

func (s *GeminiMessagesCompatService) ForwardNative(ctx context.Context, c *gin.Context, account *Account, originalModel string, action string, stream bool, body []byte) (*ForwardResult, error) {
	if s == nil {
		return nil, nil
	}
	return NewGeminiNativeGatewayService(s).ForwardNative(ctx, c, account, originalModel, action, stream, body)
}

func (s *GeminiMessagesCompatService) BuildGeminiLiveUpstream(ctx context.Context, account *Account, constrained bool, ephemeralToken string) (*GeminiLiveUpstream, error) {
	if s == nil {
		return nil, nil
	}
	return NewGeminiLiveGatewayService(s).BuildGeminiLiveUpstream(ctx, account, constrained, ephemeralToken)
}

func (s *GeminiMessagesCompatService) ForwardGeminiPassthrough(ctx context.Context, input GeminiPublicPassthroughInput) (*GeminiPublicPassthroughOutput, error) {
	if s == nil {
		return nil, nil
	}
	path := strings.ToLower(strings.TrimSpace(firstNonEmptyString(input.UpstreamPath, input.Path)))
	switch {
	case input.ResourceKind == UpstreamResourceKindGeminiInteraction:
		return NewGeminiInteractionsGatewayService(s).ForwardGeminiPassthrough(ctx, input)
	case input.UpstreamPath == GeminiLiveAuthTokensPath, strings.Contains(path, "/v1beta/live"):
		return NewGeminiLiveGatewayService(s).ForwardGeminiPassthrough(ctx, input)
	case strings.Contains(path, "/v1beta/openai/"):
		return NewGeminiCompatGatewayService(s).ForwardGeminiPassthrough(ctx, input)
	default:
		return NewGeminiNativeGatewayService(s).ForwardGeminiPassthrough(ctx, input)
	}
}
