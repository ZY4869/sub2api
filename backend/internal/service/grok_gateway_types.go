package service

import "time"

const (
	GrokRouteModeAPIKey = "apikey"
	GrokRouteModeSSO    = "sso"

	defaultGrokAPIBaseURL     = "https://api.x.ai"
	defaultGrokReverseBaseURL = "https://grok.com"

	grokEndpointChatCompletions = "/v1/chat/completions"
	grokEndpointResponses       = "/v1/responses"
	grokEndpointImagesGen       = "/v1/images/generations"
	grokEndpointImagesEdits     = "/v1/images/edits"
	grokEndpointVideosGen       = "/v1/videos/generations"
	grokEndpointVideosStatus    = "/v1/videos/:request_id"
)

type GrokFailedUsageInfo struct {
	RequestID     string
	Model         string
	UpstreamModel string
	ErrorCode     string
	ErrorMessage  string
	ImageCount    int
	ImageSize     string
	MediaType     string
	Duration      time.Duration
}

type GrokGatewayForwardResult struct {
	Result            *ForwardResult
	RouteMode         string
	Endpoint          string
	MediaType         string
	UpstreamRequestID string
	SkipUsageRecord   bool
	FailedUsage       *GrokFailedUsageInfo
}

type GrokSSOProbeResult struct {
	Tier             string
	Capabilities     GrokCapabilities
	CapabilityModels []string
	VisibleModels    []string
	RequestedModel   string
	MappedModel      string
	ResponseID       string
	ConversationID   string
}
