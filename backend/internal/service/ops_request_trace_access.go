package service

import "context"

type opsRequestTraceAdminRawAccessContextKey struct{}

// WithOpsRequestTraceAdminRawAccess marks the request-details context as being
// handled by an authenticated admin, so raw payload access can follow admin
// privileges without widening reviewer access.
func WithOpsRequestTraceAdminRawAccess(ctx context.Context, allowed bool) context.Context {
	if !allowed {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, opsRequestTraceAdminRawAccessContextKey{}, true)
}

func hasOpsRequestTraceAdminRawAccess(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	allowed, _ := ctx.Value(opsRequestTraceAdminRawAccessContextKey{}).(bool)
	return allowed
}
