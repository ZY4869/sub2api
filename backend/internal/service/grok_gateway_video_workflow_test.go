package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrokBuildVideoWorkflowRequestFromVideosBody(t *testing.T) {
	req, err := grokBuildVideoWorkflowRequestFromVideosBody([]byte(`{
		"prompt":"neon rain city",
		"model":"grok-imagine-video",
		"size":"1280x720",
		"quality":"high",
		"image_reference":{"image_url":"https://cdn.example.com/ref.png"}
	}`))
	require.NoError(t, err)
	require.NotNil(t, req)
	require.Equal(t, "videos", req.EntryPoint)
	require.Equal(t, GrokModelImagineVideo, req.RequestedModel)
	require.Equal(t, "neon rain city", req.Prompt)
	require.Equal(t, "https://cdn.example.com/ref.png", req.ImageURL)
	require.Equal(t, "16:9", req.AspectRatio)
	require.Equal(t, "720p", req.Resolution)
	require.Equal(t, grokVideoDefaultSeconds, req.Seconds)
}

func TestGrokBuildAPIKeyVideoPayload(t *testing.T) {
	req := &grokVideoWorkflowRequest{
		RequestedModel: GrokModelImagineVideo,
		Prompt:         "slow aerial shot",
		ImageURL:       "https://cdn.example.com/ref.png",
		AspectRatio:    "16:9",
		Resolution:     "720p",
		Seconds:        12,
	}
	body, endpoint, err := grokBuildAPIKeyVideoPayload(req, "grok-imagine-video")
	require.NoError(t, err)
	require.Equal(t, grokEndpointVideosGen, endpoint)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "grok-imagine-video", payload["model"])
	require.Equal(t, "slow aerial shot", payload["prompt"])
	require.Equal(t, "https://cdn.example.com/ref.png", payload["image_url"])
	require.Equal(t, "16:9", payload["aspect_ratio"])
	require.Equal(t, "720p", payload["resolution"])
	require.EqualValues(t, 12, payload["duration_seconds"])
}

func TestGrokParseVideoResultBody(t *testing.T) {
	result := grokParseVideoResultBody([]byte(`{
		"request_id":"vid_123",
		"status":"done",
		"video":{
			"url":"https://cdn.example.com/final.mp4",
			"model":"grok-imagine-video",
			"duration_seconds":12,
			"resolution":"720p",
			"aspect_ratio":"16:9",
			"thumbnail_url":"https://cdn.example.com/final.jpg",
			"mime_type":"video/mp4"
		}
	}`), GrokModelImagineVideo, "grok-imagine-video")

	require.NotNil(t, result)
	require.Equal(t, "vid_123", result.RequestID)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, GrokModelImagineVideo, result.Model)
	require.Equal(t, "grok-imagine-video", result.UpstreamModel)
	require.Equal(t, "https://cdn.example.com/final.mp4", result.URL)
	require.Equal(t, "https://cdn.example.com/final.jpg", result.ThumbnailURL)
	require.Equal(t, "720p", result.Resolution)
	require.Equal(t, "16:9", result.AspectRatio)
	require.Equal(t, 12, result.Seconds)
	require.Equal(t, "video/mp4", result.MimeType)
}
