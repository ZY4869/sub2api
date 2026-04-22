package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/stretchr/testify/require"
)

func TestInferAvailableTestModelMode_TreatsImageGenerationToolAsImage(t *testing.T) {
	mode := inferAvailableTestModelMode("gpt-5.4-mini", &modelregistry.ModelEntry{
		Capabilities: []string{"text", "image_generation_tool"},
	})
	require.Equal(t, "image", mode)
}
