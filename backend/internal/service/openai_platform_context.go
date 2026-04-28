package service

import "context"

type openAIPlatformContextKey struct{}

func WithOpenAIPlatform(ctx context.Context, platform string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	switch platform {
	case PlatformCopilot:
		return context.WithValue(ctx, openAIPlatformContextKey{}, PlatformCopilot)
	case PlatformDeepSeek:
		return context.WithValue(ctx, openAIPlatformContextKey{}, PlatformDeepSeek)
	default:
		return context.WithValue(ctx, openAIPlatformContextKey{}, PlatformOpenAI)
	}
}

func OpenAIPlatformFromContext(ctx context.Context) string {
	if ctx == nil {
		return PlatformOpenAI
	}
	if platform, ok := ctx.Value(openAIPlatformContextKey{}).(string); ok && platform != "" {
		return platform
	}
	return PlatformOpenAI
}
