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

func TestGrokBuildVideoWorkflowRequestFromVideosBodyWithOperation(t *testing.T) {
	req, err := grokBuildVideoWorkflowRequestFromVideosBodyWithOperation([]byte(`{
		"prompt":"extend the motion",
		"model":"grok-imagine-video",
		"video_url":"https://cdn.example.com/input.mp4",
		"duration_seconds":8
	}`), grokVideoOperationExtension)

	require.NoError(t, err)
	require.Equal(t, grokVideoOperationExtension, req.Operation)
	require.Equal(t, "https://cdn.example.com/input.mp4", req.VideoURL)
	require.Equal(t, 8, req.Seconds)
}

func TestGrokBuildAPIKeyVideoPayload(t *testing.T) {
	req := &grokVideoWorkflowRequest{
		Operation:      grokVideoOperationCreate,
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

func TestGrokBuildAPIKeyVideoPayloadEditAndExtension(t *testing.T) {
	tests := []struct {
		name      string
		operation grokVideoOperation
		endpoint  string
	}{
		{name: "edit", operation: grokVideoOperationEdit, endpoint: grokEndpointVideosEdits},
		{name: "extension", operation: grokVideoOperationExtension, endpoint: grokEndpointVideosExtension},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &grokVideoWorkflowRequest{
				Operation:      tt.operation,
				RequestedModel: GrokModelImagineVideo,
				Prompt:         "continue with matching motion",
				VideoURL:       "https://cdn.example.com/input.mp4",
				AspectRatio:    "16:9",
				Resolution:     "720p",
				Seconds:        8,
			}
			body, endpoint, err := grokBuildAPIKeyVideoPayload(req, "grok-imagine-video")
			require.NoError(t, err)
			require.Equal(t, tt.endpoint, endpoint)

			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload))
			require.Equal(t, "https://cdn.example.com/input.mp4", payload["video_url"])
			require.Equal(t, "continue with matching motion", payload["prompt"])
		})
	}
}

func TestGrokValidateVideoWorkflowRequestRequiresVideoForEditAndExtension(t *testing.T) {
	for _, operation := range []grokVideoOperation{grokVideoOperationEdit, grokVideoOperationExtension} {
		req := &grokVideoWorkflowRequest{
			Operation:      operation,
			RequestedModel: GrokModelImagineVideo,
			Prompt:         "change the clip",
		}

		err := grokValidateVideoWorkflowRequest(req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "video_url is required")
	}
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
