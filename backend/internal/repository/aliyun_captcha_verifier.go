package repository

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	captchaclient "github.com/alibabacloud-go/captcha-20230305/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/dara"
)

type aliyunCaptchaVerifier struct{}

func NewAliyunCaptchaVerifier() service.AliyunCaptchaVerifier {
	return &aliyunCaptchaVerifier{}
}

func (v *aliyunCaptchaVerifier) VerifyAliyunCaptcha(ctx context.Context, cfg service.AliyunCaptchaVerifyConfig, captchaVerifyParam string) error {
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "cn-shanghai"
	}
	apiCfg := &openapiutil.Config{
		AccessKeyId:     dara.String(strings.TrimSpace(cfg.AccessKeyID)),
		AccessKeySecret: dara.String(strings.TrimSpace(cfg.AccessKeySecret)),
		RegionId:        dara.String(region),
	}
	if prefix := strings.TrimSpace(cfg.Prefix); prefix != "" {
		apiCfg.Endpoint = dara.String(prefix + ".captcha.aliyuncs.com")
	}
	client, err := captchaclient.NewClient(apiCfg)
	if err != nil {
		logger.LegacyPrintf("repository.captcha", "[AliyunCaptcha] create client failed: %v", err)
		return service.ErrCaptchaNotConfigured
	}
	req := &captchaclient.VerifyIntelligentCaptchaRequest{
		CaptchaVerifyParam: dara.String(strings.TrimSpace(captchaVerifyParam)),
		SceneId:            dara.String(strings.TrimSpace(cfg.SceneID)),
	}
	resp, err := client.VerifyIntelligentCaptchaWithContext(ctx, req, &dara.RuntimeOptions{})
	if err != nil {
		logger.LegacyPrintf("repository.captcha", "[AliyunCaptcha] verify request failed: %v", err)
		return service.ErrCaptchaVerificationFailed
	}
	if resp == nil || resp.Body == nil || resp.Body.Result == nil ||
		resp.Body.Result.VerifyResult == nil || !*resp.Body.Result.VerifyResult {
		logger.LegacyPrintf("repository.captcha", "[AliyunCaptcha] verification rejected")
		return service.ErrCaptchaVerificationFailed
	}
	return nil
}
