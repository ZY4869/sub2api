package middleware

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type PanelRateLimiter struct {
	settings *service.SettingService
	limiter  panelRateLimitChecker
	now      func() time.Time

	mu        sync.Mutex
	cached    *service.PanelRateLimitSettings
	expiresAt time.Time
}

const panelRateLimitSettingsCacheTTL = 30 * time.Second

type panelRateLimitChecker interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration, opts basemiddleware.RateLimitOptions) (bool, error)
}

func NewPanelRateLimiter(settings *service.SettingService, limiter panelRateLimitChecker) *PanelRateLimiter {
	return &PanelRateLimiter{
		settings: settings,
		limiter:  limiter,
		now:      time.Now,
	}
}

func (r *PanelRateLimiter) InvalidateCache() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.cached = nil
	r.expiresAt = time.Time{}
	r.mu.Unlock()
}

func (r *PanelRateLimiter) User() gin.HandlerFunc {
	return r.limitAuthenticated("user", func(settings *service.PanelRateLimitSettings) int {
		return settings.UserRPM
	})
}

func (r *PanelRateLimiter) Heavy() gin.HandlerFunc {
	return r.limitAuthenticated("heavy", func(settings *service.PanelRateLimitSettings) int {
		return settings.HeavyRPM
	})
}

func (r *PanelRateLimiter) PublicIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, ok := r.currentSettings(c)
		if !ok || !settings.Enabled {
			c.Next()
			return
		}
		clientIP := ip.GetTrustedClientIP(c)
		if !isRateLimitablePublicIP(clientIP) {
			c.Next()
			return
		}
		r.apply(c, "panel:public-ip:"+clientIP, settings.PublicIPRPM)
	}
}

func (r *PanelRateLimiter) limitAuthenticated(scope string, selectLimit func(*service.PanelRateLimitSettings) int) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, ok := r.currentSettings(c)
		if !ok || !settings.Enabled {
			c.Next()
			return
		}
		if settings.ExemptAdmin {
			if role, ok := GetUserRoleFromContext(c); ok && role == service.RoleAdmin {
				c.Next()
				return
			}
		}
		subject, ok := GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 {
			c.Next()
			return
		}
		r.apply(c, "panel:"+scope+":user:"+strconv.FormatInt(subject.UserID, 10), selectLimit(settings))
	}
}

func (r *PanelRateLimiter) apply(c *gin.Context, key string, limit int) {
	allowed, err := r.limiter.Allow(c.Request.Context(), key, limit, time.Minute, basemiddleware.RateLimitOptions{
		FailureMode: basemiddleware.RateLimitFailOpen,
	})
	if err != nil {
		log.Printf("[PanelRateLimit] fail-open: key=%s err=%v", key, err)
	}
	if !allowed {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error":   "rate limit exceeded",
			"message": "Too many requests, please try again later",
		})
		return
	}
	c.Next()
}

func (r *PanelRateLimiter) currentSettings(c *gin.Context) (*service.PanelRateLimitSettings, bool) {
	if r == nil || r.settings == nil || r.limiter == nil {
		return service.DefaultPanelRateLimitSettings(), false
	}
	now := r.now()
	r.mu.Lock()
	if r.cached != nil && now.Before(r.expiresAt) {
		settings := r.cached
		r.mu.Unlock()
		return settings, true
	}
	r.mu.Unlock()

	settings, err := r.settings.GetPanelRateLimitSettings(c.Request.Context())
	if err != nil {
		log.Printf("[PanelRateLimit] settings unavailable, fail-open: %v", err)
		return service.DefaultPanelRateLimitSettings(), false
	}

	r.mu.Lock()
	r.cached = settings
	r.expiresAt = now.Add(panelRateLimitSettingsCacheTTL)
	r.mu.Unlock()
	return settings, true
}

func isRateLimitablePublicIP(raw string) bool {
	parsed := net.ParseIP(raw)
	if parsed == nil {
		return false
	}
	return parsed.IsGlobalUnicast() && !parsed.IsPrivate() && !parsed.IsLoopback() && !parsed.IsLinkLocalUnicast() && !parsed.IsLinkLocalMulticast() && !parsed.IsUnspecified()
}
