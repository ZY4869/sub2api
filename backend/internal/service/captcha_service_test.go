package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type captchaSettingRepoStub struct {
	values map[string]string
}

func (s *captchaSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *captchaSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *captchaSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *captchaSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *captchaSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *captchaSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *captchaSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type fakeTencentCaptchaVerifier struct {
	calls int
	cfg   TencentCaptchaVerifyConfig
	proof TencentCaptchaProof
}

func (f *fakeTencentCaptchaVerifier) VerifyTencentCaptcha(ctx context.Context, cfg TencentCaptchaVerifyConfig, proof TencentCaptchaProof) error {
	f.calls++
	f.cfg = cfg
	f.proof = proof
	return nil
}

type fakeAliyunCaptchaVerifier struct {
	calls int
	cfg   AliyunCaptchaVerifyConfig
	token string
}

func (f *fakeAliyunCaptchaVerifier) VerifyAliyunCaptcha(ctx context.Context, cfg AliyunCaptchaVerifyConfig, captchaVerifyParam string) error {
	f.calls++
	f.cfg = cfg
	f.token = captchaVerifyParam
	return nil
}

func TestCaptchaServiceVerifyNoneProviderOptionalAndRequired(t *testing.T) {
	ctx := context.Background()
	svc := NewCaptchaService(NewSettingService(&captchaSettingRepoStub{values: map[string]string{}}, &config.Config{}), nil, nil, nil)

	require.NoError(t, svc.Verify(ctx, CaptchaProof{}, "203.0.113.8", false))
	require.ErrorIs(t, svc.Verify(ctx, CaptchaProof{}, "203.0.113.8", true), ErrCaptchaNotConfigured)
}

func TestCaptchaServiceVerifyTencentFailClosedAndPassesProof(t *testing.T) {
	ctx := context.Background()
	values := map[string]string{
		SettingKeyTencentCaptchaEnabled:      "true",
		SettingKeyTencentCaptchaAppID:        "100001",
		SettingKeyTencentCaptchaAppSecretKey: "app-secret",
	}
	missingSecretSvc := NewCaptchaService(NewSettingService(&captchaSettingRepoStub{values: values}, &config.Config{}), nil, &fakeTencentCaptchaVerifier{}, nil)
	require.ErrorIs(t, missingSecretSvc.Verify(ctx, CaptchaProof{TencentCaptchaTicket: "ticket", TencentCaptchaRandstr: "rand"}, "203.0.113.9", true), ErrCaptchaNotConfigured)

	values[SettingKeyTencentCaptchaCloudSecretID] = "cloud-id"
	values[SettingKeyTencentCaptchaCloudSecretKey] = "cloud-secret"
	verifier := &fakeTencentCaptchaVerifier{}
	svc := NewCaptchaService(NewSettingService(&captchaSettingRepoStub{values: values}, &config.Config{}), nil, verifier, nil)

	require.ErrorIs(t, svc.Verify(ctx, CaptchaProof{}, "203.0.113.9", true), ErrCaptchaVerificationFailed)
	require.NoError(t, svc.Verify(ctx, CaptchaProof{TencentCaptchaTicket: " ticket ", TencentCaptchaRandstr: " rand "}, "203.0.113.9", true))
	require.Equal(t, 1, verifier.calls)
	require.Equal(t, "100001", verifier.cfg.AppID)
	require.Equal(t, "ticket", verifier.proof.Ticket)
	require.Equal(t, "rand", verifier.proof.Randstr)
	require.Equal(t, "203.0.113.9", verifier.proof.RemoteIP)
}

func TestCaptchaServiceVerifyAliyunFailClosedAndPassesProof(t *testing.T) {
	ctx := context.Background()
	values := map[string]string{
		SettingKeyAliyunCaptchaEnabled:     "true",
		SettingKeyAliyunCaptchaSceneID:     "scene",
		SettingKeyAliyunCaptchaAccessKeyID: "ak",
	}
	missingSecretSvc := NewCaptchaService(NewSettingService(&captchaSettingRepoStub{values: values}, &config.Config{}), nil, nil, &fakeAliyunCaptchaVerifier{})
	require.ErrorIs(t, missingSecretSvc.Verify(ctx, CaptchaProof{AliyunCaptchaVerifyParam: "token"}, "203.0.113.10", true), ErrCaptchaNotConfigured)

	values[SettingKeyAliyunCaptchaAccessKeySecret] = "secret"
	values[SettingKeyAliyunCaptchaPrefix] = "prefix"
	values[SettingKeyAliyunCaptchaRegion] = "cn-shanghai"
	verifier := &fakeAliyunCaptchaVerifier{}
	svc := NewCaptchaService(NewSettingService(&captchaSettingRepoStub{values: values}, &config.Config{}), nil, nil, verifier)

	require.ErrorIs(t, svc.Verify(ctx, CaptchaProof{}, "203.0.113.10", true), ErrCaptchaVerificationFailed)
	require.NoError(t, svc.Verify(ctx, CaptchaProof{AliyunCaptchaVerifyParam: " verify-param "}, "203.0.113.10", true))
	require.Equal(t, 1, verifier.calls)
	require.Equal(t, "scene", verifier.cfg.SceneID)
	require.Equal(t, "prefix", verifier.cfg.Prefix)
	require.Equal(t, "cn-shanghai", verifier.cfg.Region)
	require.Equal(t, "verify-param", verifier.token)
}

func TestCaptchaProviderMutualExclusionAndSelection(t *testing.T) {
	require.NoError(t, ValidateCaptchaProviderMutualExclusion(false, false, false))
	require.NoError(t, ValidateCaptchaProviderMutualExclusion(true, false, false))
	require.NoError(t, ValidateCaptchaProviderMutualExclusion(false, true, false))
	require.NoError(t, ValidateCaptchaProviderMutualExclusion(false, false, true))
	require.ErrorIs(t, ValidateCaptchaProviderMutualExclusion(true, true, false), ErrCaptchaProviderConflict)
	require.ErrorIs(t, ValidateCaptchaProviderMutualExclusion(false, true, true), ErrCaptchaProviderConflict)

	require.Equal(t, CaptchaProviderNone, CaptchaProviderFromRuntime(CaptchaRuntimeSettings{}))
	require.Equal(t, CaptchaProviderTurnstile, CaptchaProviderFromRuntime(CaptchaRuntimeSettings{TurnstileEnabled: true}))
	require.Equal(t, CaptchaProviderTencent, CaptchaProviderFromRuntime(CaptchaRuntimeSettings{TencentEnabled: true}))
	require.Equal(t, CaptchaProviderAliyun, CaptchaProviderFromRuntime(CaptchaRuntimeSettings{AliyunEnabled: true}))
}
