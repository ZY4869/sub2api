package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type authSessionBindingContextKey struct{}

type AuthSessionBinding struct {
	ClientIP  string
	UserAgent string
}

func WithAuthSessionBinding(ctx context.Context, clientIP, userAgent string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	binding := AuthSessionBinding{
		ClientIP:  strings.TrimSpace(clientIP),
		UserAgent: strings.TrimSpace(userAgent),
	}
	return context.WithValue(ctx, authSessionBindingContextKey{}, binding)
}

func AuthSessionBindingFromContext(ctx context.Context) (AuthSessionBinding, bool) {
	if ctx == nil {
		return AuthSessionBinding{}, false
	}
	binding, ok := ctx.Value(authSessionBindingContextKey{}).(AuthSessionBinding)
	return binding, ok
}

func hashAuthSessionBindingValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func authSessionBindingHashesFromContext(ctx context.Context) (string, string) {
	binding, ok := AuthSessionBindingFromContext(ctx)
	if !ok {
		return "", ""
	}
	return hashAuthSessionBindingValue(binding.ClientIP), hashAuthSessionBindingValue(binding.UserAgent)
}
