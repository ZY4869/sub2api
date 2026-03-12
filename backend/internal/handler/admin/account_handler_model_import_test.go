package admin

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestDefaultAvailableModels_PrefersTestExposure(t *testing.T) {
	h := &AccountHandler{
		modelRegistryService: service.NewModelRegistryService(newTestSettingRepo()),
	}

	models := h.defaultAvailableModels(context.Background(), &service.Account{Platform: service.PlatformOpenAI})
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	require.Contains(t, ids, "gpt-5.4")
	require.NotContains(t, ids, "gpt-5-codex")
}
