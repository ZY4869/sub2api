package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCanonicalRequestModel_StripsClaudeMillionContextSuffix(t *testing.T) {
	svc := &GatewayService{}

	require.Equal(t, "claude-sonnet-4.5", svc.resolveCanonicalRequestModel(context.Background(), "claude-sonnet-4.5[1m]"))
	require.Equal(t, "deepseek-v4-pro", svc.resolveCanonicalRequestModel(context.Background(), "deepseek-v4-pro[1m]"))
}
