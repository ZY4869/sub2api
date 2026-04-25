package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/stretchr/testify/require"
)

func TestInferAvailableTestModelMode_DoesNotTreatImageGenerationToolAsImage(t *testing.T) {
	mode := inferAvailableTestModelMode("gpt-5.4-mini", &modelregistry.ModelEntry{
		Capabilities: []string{"text", "image_generation_tool"},
	})
	require.Equal(t, "text", mode)
}

func TestInferAvailableTestModelMode_TreatsImageGenerationAsImage(t *testing.T) {
	mode := inferAvailableTestModelMode("gpt-5.4-mini", &modelregistry.ModelEntry{
		Capabilities: []string{"text", "image_generation"},
	})
	require.Equal(t, "image", mode)
}
