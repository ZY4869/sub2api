package repository

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/captcha/v20190722"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type tencentCaptchaVerifier struct{}

func NewTencentCaptchaVerifier() service.TencentCaptchaVerifier {
	return &tencentCaptchaVerifier{}
}

func (v *tencentCaptchaVerifier) VerifyTencentCaptcha(ctx context.Context, cfg service.TencentCaptchaVerifyConfig, proof service.TencentCaptchaProof) error {
	appID, err := strconv.ParseUint(strings.TrimSpace(cfg.AppID), 10, 64)
	if err != nil || appID == 0 {
		return service.ErrCaptchaNotConfigured
	}
	credential := common.NewCredential(strings.TrimSpace(cfg.CloudSecretID), strings.TrimSpace(cfg.CloudSecretKey))
	client, err := v20190722.NewClient(credential, "ap-guangzhou", profile.NewClientProfile())
	if err != nil {
		logger.LegacyPrintf("repository.captcha", "[TencentCaptcha] create client failed: %v", err)
		return service.ErrCaptchaNotConfigured
	}
	req := v20190722.NewDescribeCaptchaResultRequest()
	req.CaptchaType = common.Uint64Ptr(9)
	req.CaptchaAppId = common.Uint64Ptr(appID)
	req.AppSecretKey = common.StringPtr(strings.TrimSpace(cfg.AppSecretKey))
	req.Ticket = common.StringPtr(strings.TrimSpace(proof.Ticket))
	req.Randstr = common.StringPtr(strings.TrimSpace(proof.Randstr))
	req.UserIp = common.StringPtr(strings.TrimSpace(proof.RemoteIP))
	resp, err := client.DescribeCaptchaResultWithContext(ctx, req)
	if err != nil {
		logger.LegacyPrintf("repository.captcha", "[TencentCaptcha] verify request failed: %v", err)
		return service.ErrCaptchaVerificationFailed
	}
	if resp == nil || resp.Response == nil || resp.Response.CaptchaCode == nil || *resp.Response.CaptchaCode != 1 {
		logger.LegacyPrintf("repository.captcha", "[TencentCaptcha] verification rejected")
		return service.ErrCaptchaVerificationFailed
	}
	return nil
}
