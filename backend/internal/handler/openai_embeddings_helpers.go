package handler

import (
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

type openAIEmbeddingsRequest struct {
	apiKey              *service.APIKey
	subject             middleware2.AuthSubject
	body                []byte
	reqModel            string
	publicRequestModel  string
	runtimeRequestModel string
	requestPayloadHash  string
	publicCatalogEntry  *service.PublishedPublicCatalogEntry
	reqLog              *zap.Logger
}

type openAIEmbeddingsForwardInput struct {
	req                   *openAIEmbeddingsRequest
	currentAPIKey         *service.APIKey
	currentSubscription   *service.UserSubscription
	channelState          *service.GatewayChannelState
	runtimeSelectionModel string
	sessionHash           string
	excludedGroupIDs      map[int64]struct{}
	routingStart          time.Time
	streamStarted         *bool
}
