package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthGoogle is a Google-style error wrapper for API key auth.
func APIKeyAuthGoogle(apiKeyService *service.APIKeyService, cfg *config.Config) gin.HandlerFunc {
	return APIKeyAuthWithSubscriptionGoogle(apiKeyService, nil, cfg)
}

// APIKeyAuthWithSubscriptionGoogle behaves like ApiKeyAuthWithSubscription but returns Google-style errors:
// {"error":{"code":401,"message":"...","status":"UNAUTHENTICATED"}}
//
// It is intended for Gemini native endpoints (/v1beta) to match Gemini SDK expectations.
func APIKeyAuthWithSubscriptionGoogle(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if v := strings.TrimSpace(c.Query("api_key")); v != "" {
			abortWithGoogleError(c, 400, "Query parameter api_key is deprecated. Use Authorization header or key instead.")
			return
		}
		apiKeyString := extractAPIKeyForGoogle(c)
		if apiKeyString == "" {
			abortWithGoogleError(c, 401, "API key is required")
			return
		}

		apiKey, err := apiKeyService.GetByKey(c.Request.Context(), apiKeyString)
		if err != nil {
			if errors.Is(err, service.ErrAPIKeyNotFound) {
				abortWithGoogleError(c, 401, "Invalid API key")
				return
			}
			abortWithGoogleError(c, 500, "Failed to validate API key")
			return
		}

		if !apiKey.IsActive() {
			abortWithGoogleError(c, 401, "API key is disabled")
			return
		}
		if apiKey.User == nil {
			abortWithGoogleError(c, 401, "User associated with API key not found")
			return
		}
		if !apiKey.User.IsActive() {
			abortWithGoogleError(c, 401, "User account is not active")
			return
		}
		timeAccessStart := time.Now()
		eval := apiKey.EvaluateTimeAccess(timeAccessStart)
		protocolruntime.RecordTimePolicyDecision("api_key", eval.Allowed, eval.Reason, time.Since(timeAccessStart).Milliseconds())
		if !eval.Allowed {
			logAPIKeyTimeAccessDenied(c, apiKey, eval, timeAccessStart)
			abortWithGoogleTimeAccessDenied(c, eval)
			return
		}
		if apiKeyHasGroupBindings(apiKey) && !apiKeyHasUsableGroup(apiKey) {
			abortWithGoogleError(c, 403, "API key group is unavailable")
			return
		}

		// 生图专用 Key：限制可访问的入口（Google/Gemini 风格错误体）。
		if apiKey.ImageOnlyEnabled {
			method := ""
			path := ""
			if c.Request != nil {
				method = c.Request.Method
				if c.Request.URL != nil {
					path = c.Request.URL.Path
				}
			}
			if !isImageOnlyAllowedGatewayRequest(method, path) {
				abortWithGoogleError(c, 403, "生图专用 Key 仅允许调用图片生成接口")
				return
			}
		}

		// 简易模式：跳过余额和订阅检查
		if cfg.RunMode == config.RunModeSimple {
			c.Set(string(ContextKeyAPIKey), apiKey)
			c.Set(string(ContextKeyUser), AuthSubject{
				UserID:      apiKey.User.ID,
				Concurrency: apiKey.User.Concurrency,
			})
			c.Set(string(ContextKeyUserRole), apiKey.User.Role)
			setAPIKeyGroupContext(c, apiKey)
			_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)
			c.Next()
			return
		}

		dynamicGroupRouting := len(apiKey.GroupBindings) > 1
		isSubscriptionType := !dynamicGroupRouting && apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
		if isSubscriptionType && subscriptionService != nil {
			subscription, err := subscriptionService.GetActiveSubscription(
				c.Request.Context(),
				apiKey.User.ID,
				apiKey.Group.ID,
			)
			if err != nil {
				abortWithGoogleError(c, 403, "No active subscription found for this group")
				return
			}

			needsMaintenance, err := subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
			if err != nil {
				status := 403
				if errors.Is(err, service.ErrDailyLimitExceeded) ||
					errors.Is(err, service.ErrWeeklyLimitExceeded) ||
					errors.Is(err, service.ErrMonthlyLimitExceeded) {
					status = 429
				}
				abortWithGoogleError(c, status, err.Error())
				return
			}

			c.Set(string(ContextKeySubscription), subscription)

			if needsMaintenance {
				maintenanceCopy := *subscription
				subscriptionService.DoWindowMaintenance(&maintenanceCopy)
			}
		} else if !dynamicGroupRouting && googleRouteRequiresBillingHold(c) {
			if _, err := apiKeyService.TryReserveRequestBillingHold(c.Request.Context(), apiKey, cfg); err != nil {
				switch {
				case errors.Is(err, service.ErrInsufficientBalance):
					abortWithGoogleError(c, 403, "Insufficient account balance")
				case errors.Is(err, service.ErrBillingRequestReplayed), errors.Is(err, service.ErrBillingHoldAlreadyFinished):
					abortWithGoogleError(c, 409, "Billing request was already used. Please retry with a new request id.")
				default:
					abortWithGoogleError(c, 503, "Billing service temporarily unavailable. Please retry later.")
				}
				return
			}
		}

		c.Set(string(ContextKeyAPIKey), apiKey)
		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), apiKey.User.Role)
		setAPIKeyGroupContext(c, apiKey)
		_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)
		c.Next()
	}
}

func googleRouteRequiresBillingHold(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return true
	}
	method := strings.ToUpper(strings.TrimSpace(c.Request.Method))
	path := strings.ToLower(strings.TrimSpace(c.Request.URL.Path))
	if method == http.MethodGet || method == http.MethodDelete {
		return false
	}
	if strings.HasSuffix(path, "/models") || path == "/v1/models" || path == "/v1beta/models" {
		return false
	}
	return true
}

// extractAPIKeyForGoogle extracts API key for Google/Gemini endpoints.
// Priority: x-goog-api-key > Authorization: Bearer > x-api-key > query key
// This allows OpenClaw and other clients using Bearer auth to work with Gemini endpoints.
func extractAPIKeyForGoogle(c *gin.Context) string {
	// 1) preferred: Gemini native header
	if k := strings.TrimSpace(c.GetHeader("x-goog-api-key")); k != "" {
		return k
	}

	// 2) fallback: Authorization: Bearer <key>
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if k := strings.TrimSpace(parts[1]); k != "" {
				return k
			}
		}
	}

	// 3) x-api-key header (backward compatibility)
	if k := strings.TrimSpace(c.GetHeader("x-api-key")); k != "" {
		return k
	}

	// 4) query parameter key (for specific paths)
	if allowGoogleQueryKey(c.Request.URL.Path) {
		if v := strings.TrimSpace(c.Query("key")); v != "" {
			return v
		}
	}

	return ""
}

func allowGoogleQueryKey(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.HasPrefix(normalized, "/v1beta"),
		strings.HasPrefix(normalized, "/antigravity/v1beta"),
		strings.HasPrefix(normalized, "/v1/models"),
		strings.HasPrefix(normalized, "/v1alpha/authtokens"),
		strings.HasPrefix(normalized, "/upload/v1beta/files"),
		strings.HasPrefix(normalized, "/upload/v1beta/filesearchstores"):
		return true
	default:
		return false
	}
}

func abortWithGoogleError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleapi.HTTPStatusToGoogleStatus(status),
		},
	})
	c.Abort()
}

func abortWithGoogleTimeAccessDenied(c *gin.Context, eval service.TimeAccessEvaluation) {
	switch eval.Reason {
	case service.TimeAccessDecisionNotBefore:
		abortWithGoogleError(c, 403, "API key is not active yet")
	case service.TimeAccessDecisionNotAfter:
		abortWithGoogleError(c, 403, "API key has expired")
	case service.TimeAccessDecisionOutsideWindow:
		abortWithGoogleError(c, 403, "API key is outside its allowed calling window")
	default:
		abortWithGoogleError(c, 403, "API key is currently unavailable")
	}
}
