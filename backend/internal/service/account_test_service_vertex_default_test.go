package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/stretchr/testify/require"
)

func TestDefaultGeminiTestModelID_UsesVertexDefaultForVertexAccounts(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		defaultGeminiVertexValidationModel,
		defaultGeminiTestModelID(&Account{
			Platform: PlatformGemini,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"oauth_type": "vertex_ai",
			},
		}),
	)
	require.Equal(
		t,
		defaultGeminiVertexValidationModel,
		defaultGeminiTestModelID(&Account{
			Platform: PlatformGemini,
			Type:     AccountTypeAPIKey,
			Credentials: map[string]any{
				"gemini_api_variant": GeminiAPIKeyVariantVertexExpress,
			},
		}),
	)
	require.Equal(
		t,
		geminicli.DefaultTestModel,
		defaultGeminiTestModelID(&Account{
			Platform: PlatformGemini,
			Type:     AccountTypeAPIKey,
			Credentials: map[string]any{
				"gemini_api_variant": GeminiAPIKeyVariantAIStudio,
			},
		}),
	)
}
