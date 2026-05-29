package service

import (
	"context"
	"net/http"
)

type HTTPUpstreamProfile string

const (
	HTTPUpstreamProfileDefault HTTPUpstreamProfile = "default"
	HTTPUpstreamProfileOpenAI  HTTPUpstreamProfile = "openai"
)

type httpUpstreamProfileContextKey struct{}

func WithHTTPUpstreamProfile(ctx context.Context, profile HTTPUpstreamProfile) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	switch profile {
	case HTTPUpstreamProfileOpenAI:
		return context.WithValue(ctx, httpUpstreamProfileContextKey{}, profile)
	default:
		return ctx
	}
}

func HTTPUpstreamProfileFromContext(ctx context.Context) HTTPUpstreamProfile {
	if ctx == nil {
		return HTTPUpstreamProfileDefault
	}
	if profile, ok := ctx.Value(httpUpstreamProfileContextKey{}).(HTTPUpstreamProfile); ok && profile == HTTPUpstreamProfileOpenAI {
		return profile
	}
	return HTTPUpstreamProfileDefault
}

func MarkOpenAIHTTPUpstreamRequest(req *http.Request) *http.Request {
	if req == nil {
		return nil
	}
	return req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
}
