package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const oauthStartRequestContextKey = "oauth_start_request"

type oauthStartCaptchaRequest struct {
	TurnstileToken           string `json:"turnstile_token"`
	TencentCaptchaTicket     string `json:"tencent_captcha_ticket"`
	TencentCaptchaRandstr    string `json:"tencent_captcha_randstr"`
	AliyunCaptchaVerifyParam string `json:"aliyun_captcha_verify_param"`
	Mode                     string `json:"mode"`
	Redirect                 string `json:"redirect"`
	AffCode                  string `json:"aff_code"`
}

func (h *AuthHandler) verifyOAuthStartCaptcha(c *gin.Context) bool {
	req := getOAuthStartRequest(c)
	if h.authService == nil {
		return true
	}
	if err := h.authService.VerifyCaptcha(c.Request.Context(), authCaptchaProof(req.TurnstileToken, req.TencentCaptchaTicket, req.TencentCaptchaRandstr, req.AliyunCaptchaVerifyParam), ip.GetTrustedClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	return true
}

func getOAuthStartRequest(c *gin.Context) oauthStartCaptchaRequest {
	if c == nil {
		return oauthStartCaptchaRequest{}
	}
	if cached, ok := c.Get(oauthStartRequestContextKey); ok {
		if req, ok := cached.(oauthStartCaptchaRequest); ok {
			return req
		}
	}
	var req oauthStartCaptchaRequest
	if c.Request != nil && c.Request.Method == http.MethodPost {
		_ = c.ShouldBindJSON(&req)
	}
	c.Set(oauthStartRequestContextKey, req)
	return req
}

func oauthStartValue(c *gin.Context, bodyValue, queryKey string) string {
	if strings.TrimSpace(bodyValue) != "" {
		return bodyValue
	}
	if c == nil {
		return ""
	}
	return c.Query(queryKey)
}

func respondOAuthStart(c *gin.Context, authorizeURL string) {
	if c.Request != nil && c.Request.Method == http.MethodPost {
		response.Success(c, gin.H{"authorize_url": authorizeURL})
		return
	}
	c.Redirect(http.StatusFound, authorizeURL)
}
