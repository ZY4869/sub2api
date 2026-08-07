package service

import (
	"context"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	CaptchaProviderNone      = "none"
	CaptchaProviderTurnstile = "turnstile"
	CaptchaProviderTencent   = "tencent"
	CaptchaProviderAliyun    = "aliyun"
)

var (
	ErrCaptchaVerificationFailed = infraerrors.BadRequest("CAPTCHA_VERIFICATION_FAILED", "captcha verification failed")
	ErrCaptchaNotConfigured      = infraerrors.ServiceUnavailable("CAPTCHA_NOT_CONFIGURED", "captcha not configured")
	ErrCaptchaProviderConflict   = infraerrors.BadRequest("CAPTCHA_PROVIDER_CONFLICT", "only one captcha provider can be enabled at a time")
)

type CaptchaProof struct {
	TurnstileToken            string `json:"turnstile_token"`
	TencentCaptchaTicket      string `json:"tencent_captcha_ticket"`
	TencentCaptchaRandstr     string `json:"tencent_captcha_randstr"`
	AliyunCaptchaVerifyParam  string `json:"aliyun_captcha_verify_param"`
}

type CaptchaRuntimeSettings struct {
	Provider string

	TurnstileEnabled   bool
	TurnstileSecretKey string

	TencentEnabled        bool
	TencentAppID          string
	TencentAppSecretKey   string
	TencentCloudSecretID  string
	TencentCloudSecretKey string

	AliyunEnabled         bool
	AliyunSceneID         string
	AliyunPrefix          string
	AliyunRegion          string
	AliyunAccessKeyID     string
	AliyunAccessKeySecret string
}

type TencentCaptchaVerifier interface {
	VerifyTencentCaptcha(ctx context.Context, cfg TencentCaptchaVerifyConfig, proof TencentCaptchaProof) error
}

type TencentCaptchaVerifyConfig struct {
	AppID          string
	AppSecretKey   string
	CloudSecretID  string
	CloudSecretKey string
}

type TencentCaptchaProof struct {
	Ticket   string
	Randstr  string
	RemoteIP string
}

type AliyunCaptchaVerifier interface {
	VerifyAliyunCaptcha(ctx context.Context, cfg AliyunCaptchaVerifyConfig, captchaVerifyParam string) error
}

type AliyunCaptchaVerifyConfig struct {
	SceneID         string
	Prefix          string
	Region          string
	AccessKeyID     string
	AccessKeySecret string
}

type CaptchaService struct {
	settings *SettingService
	turnstile *TurnstileService
	tencent   TencentCaptchaVerifier
	aliyun    AliyunCaptchaVerifier
}

func NewCaptchaService(settings *SettingService, turnstile *TurnstileService, tencent TencentCaptchaVerifier, aliyun AliyunCaptchaVerifier) *CaptchaService {
	return &CaptchaService{
		settings:  settings,
		turnstile: turnstile,
		tencent:   tencent,
		aliyun:    aliyun,
	}
}

func (s *CaptchaService) Verify(ctx context.Context, proof CaptchaProof, remoteIP string, required bool) error {
	if s == nil || s.settings == nil {
		if required {
			return ErrCaptchaNotConfigured
		}
		return nil
	}
	runtime := s.settings.GetCaptchaRuntime(ctx)
	switch runtime.Provider {
	case CaptchaProviderTencent:
		if s.tencent == nil || strings.TrimSpace(runtime.TencentCloudSecretID) == "" || strings.TrimSpace(runtime.TencentCloudSecretKey) == "" ||
			strings.TrimSpace(runtime.TencentAppID) == "" || strings.TrimSpace(runtime.TencentAppSecretKey) == "" {
			return ErrCaptchaNotConfigured
		}
		ticket := strings.TrimSpace(proof.TencentCaptchaTicket)
		randstr := strings.TrimSpace(proof.TencentCaptchaRandstr)
		if ticket == "" || randstr == "" {
			return ErrCaptchaVerificationFailed
		}
		return s.tencent.VerifyTencentCaptcha(ctx, TencentCaptchaVerifyConfig{
			AppID:          runtime.TencentAppID,
			AppSecretKey:   runtime.TencentAppSecretKey,
			CloudSecretID:  runtime.TencentCloudSecretID,
			CloudSecretKey: runtime.TencentCloudSecretKey,
		}, TencentCaptchaProof{Ticket: ticket, Randstr: randstr, RemoteIP: remoteIP})
	case CaptchaProviderAliyun:
		if s.aliyun == nil || strings.TrimSpace(runtime.AliyunSceneID) == "" ||
			strings.TrimSpace(runtime.AliyunAccessKeyID) == "" || strings.TrimSpace(runtime.AliyunAccessKeySecret) == "" {
			return ErrCaptchaNotConfigured
		}
		token := strings.TrimSpace(proof.AliyunCaptchaVerifyParam)
		if token == "" {
			token = strings.TrimSpace(proof.TurnstileToken)
		}
		if token == "" {
			return ErrCaptchaVerificationFailed
		}
		return s.aliyun.VerifyAliyunCaptcha(ctx, AliyunCaptchaVerifyConfig{
			SceneID:         runtime.AliyunSceneID,
			Prefix:          runtime.AliyunPrefix,
			Region:          runtime.AliyunRegion,
			AccessKeyID:     runtime.AliyunAccessKeyID,
			AccessKeySecret: runtime.AliyunAccessKeySecret,
		}, token)
	case CaptchaProviderTurnstile:
		if s.turnstile == nil || strings.TrimSpace(runtime.TurnstileSecretKey) == "" {
			return ErrCaptchaNotConfigured
		}
		return s.turnstile.VerifyToken(ctx, proof.TurnstileToken, remoteIP)
	default:
		if required {
			logger.LegacyPrintf("service.captcha", "%s", "[Captcha] required but no provider configured")
			return ErrCaptchaNotConfigured
		}
		return nil
	}
}

func CaptchaProviderFromRuntime(settings CaptchaRuntimeSettings) string {
	if settings.TencentEnabled {
		return CaptchaProviderTencent
	}
	if settings.AliyunEnabled {
		return CaptchaProviderAliyun
	}
	if settings.TurnstileEnabled {
		return CaptchaProviderTurnstile
	}
	return CaptchaProviderNone
}

func ValidateCaptchaProviderMutualExclusion(turnstileEnabled, tencentEnabled, aliyunEnabled bool) error {
	count := 0
	if turnstileEnabled {
		count++
	}
	if tencentEnabled {
		count++
	}
	if aliyunEnabled {
		count++
	}
	if count > 1 {
		return ErrCaptchaProviderConflict
	}
	return nil
}

func parseTencentCaptchaAppID(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, ErrCaptchaNotConfigured
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrCaptchaNotConfigured
	}
	return id, nil
}
