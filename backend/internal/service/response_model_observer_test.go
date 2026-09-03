package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseModelObservation_FirstModelWinsAndConflictIsMarked(t *testing.T) {
	ctx := EnsureRequestMetadata(context.Background())

	observeResponseModelInContext(ctx, "gpt-5.4")
	observeResponseModelInContext(ctx, "gpt-5.3")

	model, ok := UpstreamResponseModelMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "gpt-5.4", model)
	mismatch, ok := UpstreamModelMismatchMetadataFromContext(ctx)
	require.True(t, ok)
	require.True(t, mismatch)
}

func TestResponseModelObservation_TruncatesLongValues(t *testing.T) {
	ctx := EnsureRequestMetadata(context.Background())
	observeOpenAIResponseModel(ctx, []byte(`{"response":{"model":"`+strings.Repeat("x", 240)+`"}}`))

	model, ok := UpstreamResponseModelMetadataFromContext(ctx)
	require.True(t, ok)
	require.Len(t, model, maxObservedResponseModelLength)
}

func TestResponseModelObservation_SSEReadsResponseModelAndConflict(t *testing.T) {
	observation := responseModelObservationFromSSE("data: {\"type\":\"response.created\",\"response\":{\"model\":\"gpt-5.4\"}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"model\":\"gpt-5.3\"}}\n\n")

	require.Equal(t, "gpt-5.4", observation.model)
	require.True(t, observation.conflict)
}
