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
	// lastFailoverErr 记录跨分组 failover 时最后一次上游错误，
	// 用于所有分组都耗尽后向客户端映射真实上游状态码，而不是笼统的 502。
	lastFailoverErr      *service.UpstreamFailoverError
	lastFailoverPlatform string
	groupSwitchCount     int
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
	// gatewayMessagesAccountSlotRetrySelection 表示等槽期间账号已不可调度，
	// 调用方应把该账号加入排除列表后重新选号。
	gatewayMessagesAccountSlotRetrySelection
)

type gatewayMessagesAccountSlot struct {
	account *service.Account
	release func()
	result  gatewayMessagesAccountSlotResult
}
