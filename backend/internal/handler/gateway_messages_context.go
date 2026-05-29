package handler

import (
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

type gatewayMessagesRequest struct {
	apiKey                *service.APIKey
	subject               middleware2.AuthSubject
	subscription          *service.UserSubscription
	reqLog                *zap.Logger
	body                  []byte
	parsedReq             *service.ParsedRequest
	publicRequestModel    string
	publicCatalogEntry    *service.PublishedPublicCatalogEntry
	reqModel              string
	reqStream             bool
	requestPayloadHash    string
	selectionModel        string
	bindingSelectionModel string
	streamStarted         bool
	selectedSessionHash   string
	allowedPlatforms      []string
	excludedGroupIDs      map[int64]struct{}
	isClaudeCodeClient    bool
	userWaitCounted       bool
	userReleaseFunc       func()
}

type gatewayMessagesRoute struct {
	apiKey                *service.APIKey
	subscription          *service.UserSubscription
	platform              string
	channelState          *service.GatewayChannelState
	runtimeSelectionModel string
	sessionKey            string
	hasBoundSession       bool
}

type gatewayMessagesAccountSlotResult int

const (
	gatewayMessagesAccountSlotReady gatewayMessagesAccountSlotResult = iota
	gatewayMessagesAccountSlotStop
	gatewayMessagesAccountSlotRetryGroup
)

type gatewayMessagesAccountSlot struct {
	account *service.Account
	release func()
	result  gatewayMessagesAccountSlotResult
}
