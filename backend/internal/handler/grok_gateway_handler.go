package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type grokAction string

const (
	grokActionChat        grokAction = "chat"
	grokActionResponses   grokAction = "responses"
	grokActionImagesGen   grokAction = "images_generation"
	grokActionImagesEdits grokAction = "images_edits"
	grokActionVideosGen   grokAction = "videos_generation"
	grokActionVideoStatus grokAction = "videos_status"
)

type GrokGatewayHandler struct {
	gatewayService        *service.GatewayService
	grokGatewayService    *service.GrokGatewayService
	billingCacheService   *service.BillingCacheService
	apiKeyService         *service.APIKeyService
	usageRecordWorkerPool *service.UsageRecordWorkerPool
	concurrencyHelper     *ConcurrencyHelper
	maxAccountSwitches    int
	settingService        *service.SettingService
}

func NewGrokGatewayHandler(
	gatewayService *service.GatewayService,
	grokGatewayService *service.GrokGatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	cfg *config.Config,
) *GrokGatewayHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 3
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
	}
	return &GrokGatewayHandler{
		gatewayService:        gatewayService,
		grokGatewayService:    grokGatewayService,
		billingCacheService:   billingCacheService,
		apiKeyService:         apiKeyService,
		usageRecordWorkerPool: usageRecordWorkerPool,
		concurrencyHelper:     NewConcurrencyHelper(concurrencyService, SSEPingFormatComment, pingInterval),
		maxAccountSwitches:    maxAccountSwitches,
	}
}

func (h *GrokGatewayHandler) SetSettingService(settingService *service.SettingService) {
	h.settingService = settingService
}

func (h *GrokGatewayHandler) ChatCompletions(c *gin.Context) {
	h.handleRequest(c, grokActionChat)
}

func (h *GrokGatewayHandler) Responses(c *gin.Context) {
	h.handleRequest(c, grokActionResponses)
}

func (h *GrokGatewayHandler) ImagesGeneration(c *gin.Context) {
	h.handleRequest(c, grokActionImagesGen)
}

func (h *GrokGatewayHandler) ImagesEdits(c *gin.Context) {
	h.handleRequest(c, grokActionImagesEdits)
}

func (h *GrokGatewayHandler) VideosGeneration(c *gin.Context) {
	h.handleRequest(c, grokActionVideosGen)
}

func (h *GrokGatewayHandler) VideoStatus(c *gin.Context) {
	h.handleRequest(c, grokActionVideoStatus)
}

func (h *GrokGatewayHandler) handleRequest(c *gin.Context, action grokAction) {
	streamStarted := false
	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.grok_gateway."+string(action),
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	var body []byte
	var err error
	reqModel := ""
	reqStream := false
	if action != grokActionVideoStatus {
		body, err = pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
		if err != nil {
			if maxErr, ok := extractMaxBytesError(err); ok {
				h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
				return
			}
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
			return
		}
		if len(body) == 0 {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
			return
		}
		reqModel = strings.TrimSpace(gjson.GetBytes(body, "model").String())
		reqStream = gjson.GetBytes(body, "stream").Bool()
		setOpsRequestContext(c, reqModel, reqStream, body)
	} else {
		reqModel = strings.TrimSpace(c.Param("request_id"))
		setOpsRequestContext(c, reqModel, false, nil)
	}
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireUserSlot(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	sessionHash := generateGrokSessionHash(c, body)
	excludedGroupIDs := make(map[int64]struct{})

	for {
		if isRequestCanceled(c.Request.Context(), nil) {
			return
		}
		currentAPIKey, currentSubscription, err := resolveSelectedGatewayAPIKey(
			c,
			h.settingService,
			h.gatewayService,
			h.billingCacheService,
			apiKey,
			subscription,
			reqModel,
			grokCompatiblePlatforms,
			excludedGroupIDs,
		)
		if err != nil {
			reqLog.Info("grok.group_selection_failed", zap.Error(err))
			status, code, message := groupSelectionErrorDetails(err)
			h.handleStreamingAwareError(c, status, code, message, streamStarted)
			return
		}
		runtimeSelectionModel, channelState, err := bindGatewayChannelState(c, h.gatewayService, currentAPIKey.Group, reqModel)
		if err != nil {
			if errors.Is(err, service.ErrChannelModelNotAllowed) {
				h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel", streamStarted)
				return
			}
			h.handleStreamingAwareError(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing", streamStarted)
			return
		}

		switchCount := 0
		failedAccountIDs := make(map[int64]struct{})
		var lastFailoverErr *service.UpstreamFailoverError

		for {
			if isRequestCanceled(c.Request.Context(), nil) {
				return
			}
			selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), currentAPIKey.GroupID, sessionHash, runtimeSelectionModel, failedAccountIDs, "")
			if err != nil {
				reqLog.Warn("grok.account_select_failed", zap.Error(err), zap.Int("excluded_account_count", len(failedAccountIDs)))
				if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
					break
				}
				if lastFailoverErr != nil {
					h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
				} else {
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available Grok accounts", streamStarted)
				}
				return
			}
			if selection == nil || selection.Account == nil {
				if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
					break
				}
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available Grok accounts", streamStarted)
				return
			}

			account := selection.Account
			setOpsSelectedAccount(c, account.ID, account.Platform)
			requestType := service.RequestTypeSync
			if action != grokActionVideoStatus {
				requestType = service.RequestTypeFromLegacy(reqStream, false)
			}
			mappedModel := ""
			if action != grokActionVideoStatus {
				mappedModel = account.GetMappedModel(runtimeSelectionModel)
			}
			setOpsEndpointContext(c, mappedModel, requestType)
			accountReleaseFunc, acquired := h.acquireAccountSlot(c, currentAPIKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
			if !acquired {
				return
			}
			service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
			forwardStart := time.Now()
			result, err := h.forwardAction(c.Request.Context(), c, account, action, body)
			forwardDurationMs := time.Since(forwardStart).Milliseconds()
			if accountReleaseFunc != nil {
				accountReleaseFunc()
			}
			service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, forwardDurationMs)
			if err != nil {
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					failedAccountIDs[account.ID] = struct{}{}
					lastFailoverErr = failoverErr
					if switchCount >= h.maxAccountSwitches {
						h.submitFailedUsageRecordTask(
							"handler.grok_gateway."+string(action),
							c,
							currentAPIKey,
							currentSubscription,
							account,
							reqModel,
							reqStream,
							time.Duration(forwardDurationMs)*time.Millisecond,
							failoverErr,
							err,
							"",
							0,
							"",
						)
						h.handleFailoverExhausted(c, failoverErr, streamStarted)
						return
					}
					switchCount++
					reqLog.Warn("grok.upstream_failover_switching",
						zap.Int64("account_id", account.ID),
						zap.Int("upstream_status", failoverErr.StatusCode),
						zap.Int("switch_count", switchCount),
						zap.Int("max_switches", h.maxAccountSwitches),
					)
					continue
				}
				wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
				h.submitFailedUsageRecordTask(
					"handler.grok_gateway."+string(action),
					c,
					currentAPIKey,
					currentSubscription,
					account,
					reqModel,
					reqStream,
					time.Duration(forwardDurationMs)*time.Millisecond,
					nil,
					err,
					"",
					0,
					"",
				)
				reqLog.Warn("grok.forward_failed",
					zap.Int64("account_id", account.ID),
					zap.Bool("fallback_error_response_written", wroteFallback),
					zap.Error(err),
				)
				return
			}

			if result != nil && result.FailedUsage != nil {
				h.submitFailedUsageRecordTask(
					"handler.grok_gateway."+string(action),
					c,
					currentAPIKey,
					currentSubscription,
					account,
					firstNonEmptyString(result.FailedUsage.Model, reqModel),
					reqStream,
					result.FailedUsage.Duration,
					nil,
					errors.New(firstNonEmptyString(result.FailedUsage.ErrorMessage, "grok request failed")),
					result.FailedUsage.MediaType,
					result.FailedUsage.ImageCount,
					result.FailedUsage.ImageSize,
				)
				return
			}

			if result != nil && result.Result != nil && !result.SkipUsageRecord {
				userAgent := c.GetHeader("User-Agent")
				clientIP := ip.GetClientIP(c)
				h.submitUsageRecordTask(func(ctx context.Context) {
					ctx = reattachGatewayChannelState(ctx, channelState)
					if recordErr := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
						Result:           result.Result,
						APIKey:           currentAPIKey,
						User:             currentAPIKey.User,
						Account:          account,
						Subscription:     currentSubscription,
						InboundEndpoint:  GetInboundEndpoint(c),
						UpstreamEndpoint: GetUpstreamEndpoint(c, service.EffectiveProtocol(account)),
						UserAgent:        userAgent,
						IPAddress:        clientIP,
						APIKeyService:    h.apiKeyService,
					}); recordErr != nil {
						logger.L().With(
							zap.String("component", "handler.grok_gateway."+string(action)),
							zap.Int64("user_id", subject.UserID),
							zap.Int64("api_key_id", currentAPIKey.ID),
							zap.Int64("account_id", account.ID),
						).Error("grok.record_usage_failed", zap.Error(recordErr))
					}
				})
			}

			reqLog.Debug("grok.request_completed",
				zap.Int64("account_id", account.ID),
				zap.Int("switch_count", switchCount),
				zap.String("route_mode", func() string {
					if result == nil {
						return ""
					}
					return result.RouteMode
				}()),
				zap.String("media_type", func() string {
					if result == nil {
						return ""
					}
					return result.MediaType
				}()),
			)
			return
		}
	}
}

func (h *GrokGatewayHandler) forwardAction(ctx context.Context, c *gin.Context, account *service.Account, action grokAction, body []byte) (*service.GrokGatewayForwardResult, error) {
	switch action {
	case grokActionChat:
		return h.grokGatewayService.ForwardChatCompletions(ctx, c, account, body)
	case grokActionResponses:
		return h.grokGatewayService.ForwardResponses(ctx, c, account, body, c.Request.Method, c.Param("subpath"))
	case grokActionImagesGen:
		return h.grokGatewayService.ForwardImagesGeneration(ctx, c, account, body)
	case grokActionImagesEdits:
		return h.grokGatewayService.ForwardImagesEdits(ctx, c, account, body)
	case grokActionVideosGen:
		return h.grokGatewayService.ForwardVideosGeneration(ctx, c, account, body)
	case grokActionVideoStatus:
		return h.grokGatewayService.ForwardVideoStatus(ctx, c, account, c.Param("request_id"))
	default:
		return nil, fmt.Errorf("unsupported grok action: %s", action)
	}
}

func (h *GrokGatewayHandler) submitUsageRecordTask(task service.UsageRecordTask) {
	if task == nil {
		return
	}
	if h.usageRecordWorkerPool != nil {
		h.usageRecordWorkerPool.Submit(task)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	task(ctx)
}

func (h *GrokGatewayHandler) errorResponse(c *gin.Context, status int, errType string, message string) {
	c.JSON(status, gin.H{"error": gin.H{"type": errType, "message": message}})
}

func (h *GrokGatewayHandler) handleStreamingAwareError(c *gin.Context, status int, errType string, message string, streamStarted bool) {
	if streamStarted {
		if flusher, ok := c.Writer.(http.Flusher); ok {
			payload := fmt.Sprintf("event: error\ndata: {\"error\":{\"type\":%q,\"message\":%q}}\n\n", errType, message)
			_, _ = fmt.Fprint(c.Writer, payload)
			flusher.Flush()
		}
		return
	}
	h.errorResponse(c, status, errType, message)
}

func (h *GrokGatewayHandler) ensureForwardErrorResponse(c *gin.Context, streamStarted bool) bool {
	if c == nil || c.Writer == nil || c.Writer.Written() {
		return false
	}
	h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "Grok upstream request failed", streamStarted)
	return true
}

func (h *GrokGatewayHandler) handleFailoverExhausted(c *gin.Context, err *service.UpstreamFailoverError, streamStarted bool) {
	if err == nil {
		h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "Grok upstream request failed", streamStarted)
		return
	}
	message := strings.TrimSpace(service.ExtractUpstreamErrorMessage(err.ResponseBody))
	if message == "" {
		message = "Grok upstream request failed"
	}
	status := http.StatusBadGateway
	errType := "upstream_error"
	if err.StatusCode == http.StatusTooManyRequests {
		status = http.StatusTooManyRequests
		errType = "rate_limit_error"
	}
	h.handleStreamingAwareError(c, status, errType, message, streamStarted)
}

func (h *GrokGatewayHandler) acquireUserSlot(c *gin.Context, userID int64, userConcurrency int, reqStream bool, streamStarted *bool, reqLog *zap.Logger) (func(), bool) {
	ctx := c.Request.Context()
	userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlot(ctx, userID, userConcurrency)
	if err != nil {
		reqLog.Warn("grok.user_slot_acquire_failed", zap.Error(err))
		h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Concurrency limit exceeded for user, please retry later", *streamStarted)
		return nil, false
	}
	if userAcquired {
		return wrapReleaseOnDone(ctx, userReleaseFunc), true
	}

	maxWait := service.CalculateMaxWait(userConcurrency)
	canWait, waitErr := h.concurrencyHelper.IncrementWaitCount(ctx, userID, maxWait)
	if waitErr != nil {
		reqLog.Warn("grok.user_wait_counter_increment_failed", zap.Error(waitErr))
	} else if !canWait {
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return nil, false
	}
	waitCounted := waitErr == nil && canWait
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementWaitCount(ctx, userID)
		}
	}()

	userReleaseFunc, err = h.concurrencyHelper.AcquireUserSlotWithWait(c, userID, userConcurrency, reqStream, streamStarted)
	if err != nil {
		reqLog.Warn("grok.user_slot_acquire_failed_after_wait", zap.Error(err))
		h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Concurrency limit exceeded for user, please retry later", *streamStarted)
		return nil, false
	}
	if waitCounted {
		h.concurrencyHelper.DecrementWaitCount(ctx, userID)
		waitCounted = false
	}
	return wrapReleaseOnDone(ctx, userReleaseFunc), true
}

func (h *GrokGatewayHandler) acquireAccountSlot(c *gin.Context, groupID *int64, sessionHash string, selection *service.AccountSelectionResult, reqStream bool, streamStarted *bool, reqLog *zap.Logger) (func(), bool) {
	if selection == nil || selection.Account == nil {
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available Grok accounts", *streamStarted)
		return nil, false
	}
	ctx := c.Request.Context()
	account := selection.Account
	if selection.Acquired {
		if err := h.gatewayService.BindStickySession(ctx, groupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("grok.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}
		return wrapReleaseOnDone(ctx, selection.ReleaseFunc), true
	}
	if selection.WaitPlan == nil {
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available Grok accounts", *streamStarted)
		return nil, false
	}

	canWait, waitErr := h.concurrencyHelper.IncrementAccountWaitCount(ctx, account.ID, selection.WaitPlan.MaxWaiting)
	if waitErr != nil {
		reqLog.Warn("grok.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(waitErr))
	} else if !canWait {
		h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", *streamStarted)
		return nil, false
	}
	accountWaitCounted := waitErr == nil && canWait
	defer func() {
		if accountWaitCounted {
			h.concurrencyHelper.DecrementAccountWaitCount(ctx, account.ID)
		}
	}()

	accountReleaseFunc, err := h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, reqStream, streamStarted)
	if err != nil {
		reqLog.Warn("grok.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Concurrency limit exceeded for account, please retry later", *streamStarted)
		return nil, false
	}
	if accountWaitCounted {
		h.concurrencyHelper.DecrementAccountWaitCount(ctx, account.ID)
		accountWaitCounted = false
	}
	if err := h.gatewayService.BindStickySession(ctx, groupID, sessionHash, account.ID); err != nil {
		reqLog.Warn("grok.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
	}
	return wrapReleaseOnDone(ctx, accountReleaseFunc), true
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func generateGrokSessionHash(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}
	sessionID := strings.TrimSpace(c.GetHeader("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.GetHeader("conversation_id"))
	}
	if sessionID == "" && len(body) > 0 {
		sessionID = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	}
	if sessionID == "" {
		return ""
	}
	return service.DeriveSessionHashFromSeed(sessionID)
}
