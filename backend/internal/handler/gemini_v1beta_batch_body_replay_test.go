package handler

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplayableGoogleBatchBodyCanReopenAfterFirstRead(t *testing.T) {
	body, err := newReplayableGoogleBatchBody(io.NopCloser(strings.NewReader("hello world")))
	require.NoError(t, err)
	defer body.Cleanup()

	first, err := body.Open()
	require.NoError(t, err)
	firstData, err := io.ReadAll(first)
	require.NoError(t, err)
	require.NoError(t, first.Close())

	second, err := body.Open()
	require.NoError(t, err)
	secondData, err := io.ReadAll(second)
	require.NoError(t, err)
	require.NoError(t, second.Close())

	require.Equal(t, "hello world", string(firstData))
	require.Equal(t, firstData, secondData)
}
