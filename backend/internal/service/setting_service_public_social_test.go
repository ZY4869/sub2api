//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSettingService_GetPublicSettings_ExposesSocialOAuthFlags(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyGitHubOAuthEnabled: "true",
			SettingKeyGoogleOAuthEnabled: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.GitHubOAuthEnabled)
	require.True(t, settings.GoogleOAuthEnabled)
}
