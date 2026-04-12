package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func (h *GatewayHandler) forwardGeminiLiveWebSocket(c *gin.Context) {
	if h == nil || h.geminiLiveService == nil {
		googleErrorKey(c, http.StatusServiceUnavailable, "gateway.gemini.live_service_missing", "Gemini Live service not configured")
		return
	}
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		googleErrorKey(c, http.StatusUnauthorized, "gateway.gemini.invalid_api_key", "Invalid API key")
		return
	}
	authSubject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		googleErrorKey(c, http.StatusInternalServerError, "gateway.gemini.user_context_missing", "User context not found")
		return
	}
	subscription, _ := middleware.GetSubscriptionFromContext(c)
	clientIP := ip.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	reqLog := requestLogger(c, "handler.gemini_v1beta.live", zap.Int64("user_id", authSubject.UserID), zap.Int64("api_key_id", apiKey.ID), zap.Any("group_id", apiKey.GroupID))

	clientConn, err := coderws.Accept(c.Writer, c.Request, &coderws.AcceptOptions{CompressionMode: coderws.CompressionContextTakeover})
	if err != nil {
		reqLog.Warn("gemini.live_accept_failed", zap.Error(err))
		return
	}
	defer func() { _ = clientConn.CloseNow() }()
	clientConn.SetReadLimit(16 * 1024 * 1024)

	ctx := c.Request.Context()
	readCtx, cancelRead := context.WithTimeout(ctx, 30*time.Second)
	msgType, firstMessage, err := clientConn.Read(readCtx)
	cancelRead()
	if err != nil || (msgType != coderws.MessageText && msgType != coderws.MessageBinary) || !gjson.ValidBytes(firstMessage) {
		closeOpenAIClientWS(clientConn, coderws.StatusPolicyViolation, "invalid Gemini Live setup payload")
		return
	}

	requestedModel := detectGeminiLiveRequestedModel(firstMessage)
	sessionHash := detectGeminiLiveSessionHash(firstMessage)
	setOpsRequestContext(c, requestedModel, true, firstMessage)

	currentAPIKey, currentSubscription, err := resolveSelectedGatewayAPIKey(c, h.settingService, h.gatewayService, h.billingCacheService, apiKey, subscription, requestedModel, []string{service.PlatformGemini}, nil)
	if err != nil {
		closeOpenAIClientWS(clientConn, coderws.StatusPolicyViolation, "billing or group selection failed")
		return
	}
	if !middleware.HasForcePlatform(c) && currentAPIKey.Group != nil && currentAPIKey.Group.Platform != service.PlatformGemini {
		closeOpenAIClientWS(clientConn, coderws.StatusPolicyViolation, "API key group platform is not gemini")
		return
	}

	selectionCtx := service.WithGeminiPublicProtocol(ctx, service.UpstreamProviderAIStudio)
	account, err := h.geminiLiveService.SelectAccountForModelWithExclusions(selectionCtx, currentAPIKey.GroupID, sessionHash, requestedModel, nil)
	if err != nil && requestedModel == "" {
		account, err = h.geminiLiveService.SelectAccountForAIStudioEndpoints(selectionCtx, currentAPIKey.GroupID)
	}
	if err != nil || account == nil {
		reqLog.Warn("gemini.live_account_select_failed", zap.Error(err), zap.String("model", requestedModel))
		closeOpenAIClientWS(clientConn, coderws.StatusTryAgainLater, "no available Gemini Live account")
		return
	}

	userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlot(ctx, authSubject.UserID, authSubject.Concurrency)
	if err != nil || !userAcquired {
		closeOpenAIClientWS(clientConn, coderws.StatusTryAgainLater, "too many concurrent requests, please retry later")
		return
	}
	defer wrapReleaseOnDone(ctx, userReleaseFunc)()
	accountReleaseFunc, accountAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
	if err != nil || !accountAcquired {
		closeOpenAIClientWS(clientConn, coderws.StatusTryAgainLater, "account is busy, please retry later")
		return
	}
	defer wrapReleaseOnDone(ctx, accountReleaseFunc)()

	if sessionHash != "" {
		_ = h.gatewayService.BindStickySession(ctx, currentAPIKey.GroupID, "gemini:"+sessionHash, account.ID)
	}
	upstreamCfg, err := h.geminiLiveService.BuildGeminiLiveUpstream(ctx, account, false, "")
	if err != nil {
		reqLog.Warn("gemini.live_upstream_prepare_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		closeOpenAIClientWS(clientConn, coderws.StatusInternalError, "failed to prepare Gemini Live upstream")
		return
	}
	dialCtx, cancelDial := context.WithTimeout(ctx, 20*time.Second)
	upstreamConn, handshakeHeaders, err := dialGeminiLiveUpstream(dialCtx, upstreamCfg)
	cancelDial()
	if err != nil {
		reqLog.Warn("gemini.live_upstream_dial_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		closeOpenAIClientWS(clientConn, coderws.StatusInternalError, "failed to connect Gemini Live upstream")
		return
	}
	defer func() { _ = upstreamConn.CloseNow() }()
	if err := upstreamConn.Write(ctx, msgType, firstMessage); err != nil {
		closeOpenAIClientWS(clientConn, coderws.StatusInternalError, "failed to send Gemini Live setup")
		return
	}

	setOpsSelectedAccount(c, account.ID, account.Platform)
	setOpsEndpointContext(c, requestedModel, service.RequestTypeStream)
	reqLog = reqLog.With(zap.Int64("account_id", account.ID), zap.String("model", requestedModel), zap.Bool("has_resumption_handle", sessionHash != ""))

	startedAt := time.Now()
	usageState := &geminiLiveUsageState{}
	relayCtx, relayCancel := context.WithCancel(ctx)
	group, groupCtx := errgroup.WithContext(relayCtx)
	group.Go(func() error {
		return relayGeminiLiveFrames(groupCtx, clientConn, upstreamConn, nil)
	})
	group.Go(func() error {
		return relayGeminiLiveFrames(groupCtx, upstreamConn, clientConn, func(payload []byte) {
			if newHandle := usageState.observeServerFrame(payload); newHandle != "" {
				_ = h.gatewayService.BindStickySession(ctx, currentAPIKey.GroupID, "gemini:"+service.DeriveSessionHashFromSeed("gemini-live:"+newHandle), account.ID)
			}
		})
	})
	relayErr := group.Wait()
	relayCancel()
	if relayErr != nil && !geminiLiveCloseIsGraceful(relayErr) {
		reqLog.Warn("gemini.live_relay_failed", zap.Error(relayErr))
		closeOpenAIClientWS(clientConn, coderws.StatusInternalError, "Gemini Live websocket proxy failed")
		return
	}

	usage, mediaType, requestID, upstreamModel := usageState.snapshot()
	if currentAPIKey.User != nil && usage.InputTokens+usage.OutputTokens+usage.CacheReadInputTokens > 0 {
		if requestID == "" {
			requestID = handshakeHeaders.Get("x-request-id")
		}
		h.submitUsageRecordTask(func(taskCtx context.Context) {
			if err := h.gatewayService.RecordUsageWithLongContext(taskCtx, &service.RecordUsageLongContextInput{
				Result: &service.ForwardResult{
					RequestID:     requestID,
					Usage:         usage,
					Model:         requestedModel,
					UpstreamModel: firstNonEmptyHandlerString(upstreamModel, requestedModel),
					Stream:        true,
					Duration:      time.Since(startedAt),
					MediaType:     mediaType,
				},
				APIKey:                currentAPIKey,
				User:                  currentAPIKey.User,
				Account:               account,
				Subscription:          currentSubscription,
				InboundEndpoint:       c.Request.URL.Path,
				UpstreamEndpoint:      service.EndpointGeminiLive,
				UserAgent:             userAgent,
				IPAddress:             clientIP,
				RequestBody:           firstMessage,
				RequestPayloadHash:    service.HashUsageRequestPayload(firstMessage),
				LongContextThreshold:  200000,
				LongContextMultiplier: 2.0,
				APIKeyService:         h.apiKeyService,
			}); err != nil {
				reqLog.Error("gemini.live_record_usage_failed", zap.Error(err))
			}
		})
	}
}
