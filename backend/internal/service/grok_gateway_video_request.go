package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/tidwall/gjson"
)

const (
	grokVideoDefaultSize       = "1792x1024"
	grokVideoDefaultAspect     = "3:2"
	grokVideoDefaultQuality    = "standard"
	grokVideoDefaultResolution = "480p"
	grokVideoDefaultSeconds    = 6
)

var grokVideoSizeToAspectRatio = map[string]string{
	"1280x720":  "16:9",
	"720x1280":  "9:16",
	"1792x1024": "3:2",
	"1024x1792": "2:3",
	"1024x1024": "1:1",
}

var grokVideoQualityToResolution = map[string]string{
	"standard": "480p",
	"high":     "720p",
}

type grokVideoWorkflowRequest struct {
	EntryPoint     string
	RequestedModel string
	Prompt         string
	ImageURL       string
	VideoURL       string
	ModeID         string
	AspectRatio    string
	Resolution     string
	Seconds        int
	Quality        string
	Size           string
	Stream         bool
}

type grokVideoResult struct {
	RequestID     string
	Status        string
	Model         string
	UpstreamModel string
	URL           string
	ThumbnailURL  string
	Resolution    string
	AspectRatio   string
	Seconds       int
	MimeType      string
	Provider      string
	CompletedAt   time.Time
}

func grokIsVideoRequestModel(models ...string) bool {
	for _, model := range models {
		normalized := NormalizeGrokPublicModelID(model)
		if normalized == "" {
			normalized = strings.TrimSpace(model)
		}
		if GrokIsVideoModel(normalized) {
			return true
		}
	}
	return false
}

func grokBuildVideoWorkflowRequestFromChatBody(body []byte) (*grokVideoWorkflowRequest, error) {
	var req apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse grok chat video request: %w", err)
	}
	prompt := grokChatPrompt(req.Messages)
	imageURL, videoURL := grokChatMessagesMedia(req.Messages)
	return grokBuildVideoWorkflowRequest(body, "chat", req.Model, prompt, imageURL, videoURL, req.Stream), nil
}

func grokBuildVideoWorkflowRequestFromResponsesBody(body []byte) (*grokVideoWorkflowRequest, error) {
	var req apicompat.ResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse grok responses video request: %w", err)
	}
	prompt := grokResponsesPrompt(req.Input)
	imageURL, videoURL := grokResponsesInputMedia(req.Input)
	return grokBuildVideoWorkflowRequest(body, "responses", req.Model, prompt, imageURL, videoURL, req.Stream), nil
}

func grokBuildVideoWorkflowRequestFromVideosBody(body []byte) (*grokVideoWorkflowRequest, error) {
	prompt := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "prompt").String(),
		gjson.GetBytes(body, "input").String(),
	))
	imageURL := grokExtractVideoInputURL(body,
		"image_reference.image_url",
		"image_reference.url",
		"image.url",
		"image_url",
		"source_image.url",
		"source_image_url",
		"input_image.url",
		"input_image",
	)
	videoURL := grokExtractVideoInputURL(body,
		"video.url",
		"video_url",
		"source_video.url",
		"source_video_url",
		"input_video.url",
		"input_video",
	)
	return grokBuildVideoWorkflowRequest(body, "videos", gjson.GetBytes(body, "model").String(), prompt, imageURL, videoURL, false), nil
}

func grokBuildVideoWorkflowRequest(
	body []byte,
	entryPoint string,
	model string,
	prompt string,
	imageURL string,
	videoURL string,
	stream bool,
) *grokVideoWorkflowRequest {
	requestedModel := NormalizeGrokPublicModelID(model)
	if requestedModel == "" {
		requestedModel = GrokModelImagineVideo
	}
	size := grokNormalizeVideoSize(firstNonEmptyString(
		gjson.GetBytes(body, "size").String(),
		gjson.GetBytes(body, "video_config.size").String(),
	))
	quality := grokNormalizeVideoQuality(firstNonEmptyString(
		gjson.GetBytes(body, "quality").String(),
		gjson.GetBytes(body, "video_config.quality").String(),
	))
	aspectRatio := grokNormalizeVideoAspectRatio(firstNonEmptyString(
		gjson.GetBytes(body, "aspect_ratio").String(),
		gjson.GetBytes(body, "video_config.aspect_ratio").String(),
		size,
	))
	resolution := grokNormalizeVideoResolution(firstNonEmptyString(
		gjson.GetBytes(body, "resolution").String(),
		gjson.GetBytes(body, "video_config.resolution").String(),
		grokVideoQualityToResolution[quality],
	))
	if resolution == "" {
		resolution = grokVideoDefaultResolution
	}
	seconds := grokExtractPositiveJSONInt(body,
		"seconds",
		"video_config.seconds",
		"duration_seconds",
		"video_config.duration_seconds",
		"duration",
		"video_length",
		"video_config.video_length",
		"length",
	)
	if seconds <= 0 {
		seconds = grokVideoDefaultSeconds
	}

	return &grokVideoWorkflowRequest{
		EntryPoint:     strings.TrimSpace(entryPoint),
		RequestedModel: requestedModel,
		Prompt:         strings.TrimSpace(prompt),
		ImageURL:       strings.TrimSpace(imageURL),
		VideoURL:       strings.TrimSpace(videoURL),
		ModeID:         strings.TrimSpace(gjson.GetBytes(body, "mode_id").String()),
		AspectRatio:    aspectRatio,
		Resolution:     resolution,
		Seconds:        seconds,
		Quality:        quality,
		Size:           size,
		Stream:         stream,
	}
}

func (r *grokVideoResult) MediaVideo() *apicompat.MediaVideo {
	if r == nil {
		return nil
	}
	status := strings.TrimSpace(r.Status)
	if status == "" {
		status = "completed"
	}
	provider := strings.TrimSpace(r.Provider)
	if provider == "" {
		provider = "grok"
	}
	mimeType := strings.TrimSpace(r.MimeType)
	if mimeType == "" && strings.HasSuffix(strings.ToLower(strings.TrimSpace(r.URL)), ".mp4") {
		mimeType = "video/mp4"
	}
	return &apicompat.MediaVideo{
		RequestID:    strings.TrimSpace(r.RequestID),
		Status:       status,
		URL:          strings.TrimSpace(r.URL),
		ThumbnailURL: strings.TrimSpace(r.ThumbnailURL),
		Model:        strings.TrimSpace(r.Model),
		Seconds:      r.Seconds,
		Resolution:   strings.TrimSpace(r.Resolution),
		AspectRatio:  strings.TrimSpace(r.AspectRatio),
		MimeType:     mimeType,
		Provider:     provider,
	}
}

func grokBuildVideoResponsesResponse(result *grokVideoResult, model string) *apicompat.ResponsesResponse {
	if strings.TrimSpace(model) == "" && result != nil {
		model = result.Model
	}
	fallback := grokVideoFallbackText(result)
	video := (*apicompat.MediaVideo)(nil)
	if result != nil {
		video = result.MediaVideo()
	}
	content := make([]apicompat.ResponsesContentPart, 0, 2)
	if fallback != "" {
		content = append(content, apicompat.ResponsesContentPart{
			Type: "output_text",
			Text: fallback,
		})
	}
	if video != nil {
		content = append(content, apicompat.ResponsesContentPart{
			Type:  "output_video",
			Video: video,
		})
	}
	if len(content) == 0 {
		content = append(content, apicompat.ResponsesContentPart{
			Type: "output_text",
			Text: "",
		})
	}
	responseID := "resp_grok_video"
	if result != nil && strings.TrimSpace(result.RequestID) != "" {
		responseID = result.RequestID
	}
	return &apicompat.ResponsesResponse{
		ID:     responseID,
		Object: "response",
		Model:  strings.TrimSpace(model),
		Status: "completed",
		Output: []apicompat.ResponsesOutput{{
			Type:    "message",
			ID:      "msg_" + responseID,
			Role:    "assistant",
			Status:  "completed",
			Content: content,
		}},
		Usage: &apicompat.ResponsesUsage{InputTokens: 0, OutputTokens: 0, TotalTokens: 0},
	}
}

func grokBuildVideoCreateResponse(result *grokVideoResult, req *grokVideoWorkflowRequest) map[string]any {
	now := time.Now().Unix()
	if result != nil && !result.CompletedAt.IsZero() {
		now = result.CompletedAt.Unix()
	}
	model := ""
	if req != nil {
		model = strings.TrimSpace(req.RequestedModel)
	}
	if model == "" && result != nil {
		model = strings.TrimSpace(result.Model)
	}
	requestID := ""
	videoObject := (*apicompat.MediaVideo)(nil)
	if result != nil {
		requestID = strings.TrimSpace(result.RequestID)
		videoObject = result.MediaVideo()
	}
	response := map[string]any{
		"id":           "video_" + strings.ReplaceAll(firstNonEmptyString(requestID, strconv.FormatInt(now, 10)), ":", "_"),
		"object":       "video",
		"created_at":   now,
		"completed_at": now,
		"status":       "completed",
		"model":        model,
		"url":          "",
		"request_id":   requestID,
	}
	if req != nil {
		response["prompt"] = strings.TrimSpace(req.Prompt)
		response["size"] = strings.TrimSpace(req.Size)
		response["seconds"] = req.Seconds
		response["quality"] = strings.TrimSpace(req.Quality)
	}
	if videoObject != nil {
		response["url"] = strings.TrimSpace(videoObject.URL)
		response["video"] = videoObject
		response["data"] = []map[string]any{{
			"url":           strings.TrimSpace(videoObject.URL),
			"thumbnail_url": strings.TrimSpace(videoObject.ThumbnailURL),
		}}
	}
	return response
}

func grokBuildVideoStatusResponse(result *grokVideoResult) map[string]any {
	requestID := ""
	videoObject := (*apicompat.MediaVideo)(nil)
	status := "processing"
	url := ""
	model := ""
	if result != nil {
		requestID = strings.TrimSpace(result.RequestID)
		videoObject = result.MediaVideo()
		status = firstNonEmptyString(strings.TrimSpace(result.Status), status)
		url = strings.TrimSpace(result.URL)
		model = strings.TrimSpace(result.Model)
	}
	response := map[string]any{
		"request_id": requestID,
		"status":     status,
		"model":      model,
		"url":        url,
	}
	if videoObject != nil {
		response["video"] = videoObject
		response["data"] = []map[string]any{{
			"url":           strings.TrimSpace(videoObject.URL),
			"thumbnail_url": strings.TrimSpace(videoObject.ThumbnailURL),
		}}
	}
	return response
}

func grokVideoFallbackText(result *grokVideoResult) string {
	if result == nil {
		return ""
	}
	return strings.TrimSpace(result.URL)
}

func grokChatMessagesMedia(messages []apicompat.ChatMessage) (string, string) {
	for _, message := range messages {
		imageURL, videoURL := grokExtractMediaFromRawContent(message.Content)
		if imageURL != "" || videoURL != "" {
			return imageURL, videoURL
		}
	}
	return "", ""
}

func grokResponsesInputMedia(raw json.RawMessage) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}
	var items []apicompat.ResponsesInputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return "", ""
	}
	for _, item := range items {
		imageURL, videoURL := grokExtractMediaFromRawContent(item.Content)
		if imageURL != "" || videoURL != "" {
			return imageURL, videoURL
		}
	}
	return "", ""
}

func grokExtractMediaFromRawContent(raw json.RawMessage) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}
	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", ""
	}
	for _, part := range parts {
		partType := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", part["type"])))
		switch partType {
		case "image_url", "input_image":
			if imageURL := grokExtractMediaURL(part, "image_url"); imageURL != "" {
				return imageURL, ""
			}
		case "video_url", "input_video":
			if videoURL := grokExtractMediaURL(part, "video_url"); videoURL != "" {
				return "", videoURL
			}
		}
	}
	return "", ""
}

func grokExtractMediaURL(part map[string]any, key string) string {
	if part == nil {
		return ""
	}
	if nested, ok := part[key]; ok {
		switch typed := nested.(type) {
		case string:
			return strings.TrimSpace(typed)
		case map[string]any:
			return strings.TrimSpace(firstNonEmptyString(
				fmt.Sprintf("%v", typed["url"]),
				fmt.Sprintf("%v", typed["image_url"]),
				fmt.Sprintf("%v", typed["video_url"]),
			))
		}
	}
	return strings.TrimSpace(firstNonEmptyString(
		fmt.Sprintf("%v", part["url"]),
		fmt.Sprintf("%v", part["image_url"]),
		fmt.Sprintf("%v", part["video_url"]),
	))
}

func grokExtractVideoInputURL(body []byte, paths ...string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		value := strings.TrimSpace(gjson.GetBytes(body, path).String())
		if value != "" {
			return value
		}
	}
	return ""
}

func grokNormalizeVideoSize(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	for size := range grokVideoSizeToAspectRatio {
		if normalized == size {
			return size
		}
	}
	return grokVideoDefaultSize
}

func grokNormalizeVideoQuality(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if resolution, ok := grokVideoQualityToResolution[normalized]; ok && resolution != "" {
		return normalized
	}
	return grokVideoDefaultQuality
}

func grokNormalizeVideoAspectRatio(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return grokVideoDefaultAspect
	}
	if mapped, ok := grokVideoSizeToAspectRatio[normalized]; ok {
		return mapped
	}
	switch normalized {
	case "16:9", "9:16", "3:2", "2:3", "1:1":
		return normalized
	default:
		return grokVideoDefaultAspect
	}
}

func grokExtractPositiveJSONInt(body []byte, paths ...string) int {
	for _, path := range paths {
		if path == "" {
			continue
		}
		value := gjson.GetBytes(body, path)
		switch value.Type {
		case gjson.Number:
			if n := int(value.Int()); n > 0 {
				return n
			}
		case gjson.String:
			if n, err := strconv.Atoi(strings.TrimSpace(value.String())); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}
