package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestResolveGatewayTestTarget(t *testing.T) {
	registry := newGatewayResolutionTestRegistry(t)
	svc := &AccountTestService{modelRegistryService: registry}

	testAccount := func(accepted []string, extra map[string]any, manual []AccountManualModel) *Account {
		accountExtra := map[string]any{
			"gateway_protocol":           GatewayProtocolMixed,
			"gateway_accepted_protocols": accepted,
		}
		if len(manual) > 0 {
			rawManualModels := AccountManualModelsToExtraValue(manual, true)
			items := make([]any, 0, len(rawManualModels))
			for _, item := range rawManualModels {
				items = append(items, item)
			}
			accountExtra["manual_models"] = items
		}
		for key, value := range extra {
			accountExtra[key] = value
		}
		return &Account{
			ID:       42,
			Platform: PlatformProtocolGateway,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    accountExtra,
		}
	}

	tests := []struct {
		name           string
		account        *Account
		modelID        string
		sourceProtocol string
		targetProvider string
		targetModelID  string
		want           resolvedGatewayTestTarget
		wantReason     string
		assert         func(t *testing.T, got resolvedGatewayTestTarget)
	}{
		{
			name:           "explicit source protocol wins",
			account:        testAccount([]string{PlatformOpenAI, PlatformAnthropic}, nil, nil),
			sourceProtocol: PlatformAnthropic,
			want: resolvedGatewayTestTarget{
				SourceProtocol: PlatformAnthropic,
			},
		},
		{
			name:           "target provider resolves unique protocol and default model",
			account:        testAccount([]string{PlatformOpenAI, PlatformGemini}, nil, nil),
			targetProvider: PlatformGemini,
			assert: func(t *testing.T, got resolvedGatewayTestTarget) {
				require.Equal(t, PlatformGemini, got.SourceProtocol)
				require.Equal(t, PlatformGemini, got.TargetProvider)
				require.NotEmpty(t, got.ModelID)
			},
		},
		{
			name:          "target model resolves unique protocol",
			account:       testAccount([]string{PlatformOpenAI, PlatformAnthropic}, nil, nil),
			targetModelID: "anthropic-only",
			want: resolvedGatewayTestTarget{
				ModelID:        "anthropic-only",
				SourceProtocol: PlatformAnthropic,
				TargetModelID:  "anthropic-only",
			},
		},
		{
			name: "account defaults are injected before unique matching",
			account: testAccount([]string{PlatformOpenAI, PlatformAnthropic}, map[string]any{
				"gateway_test_provider": PlatformOpenAI,
				"gateway_test_model_id": "openai-default",
			}, nil),
			want: resolvedGatewayTestTarget{
				ModelID:        "openai-default",
				SourceProtocol: PlatformOpenAI,
				TargetProvider: PlatformOpenAI,
				TargetModelID:  "openai-default",
			},
		},
		{
			name:          "manual target model outside projection is rejected as invalid",
			account:       testAccount([]string{PlatformOpenAI, PlatformAnthropic}, nil, []AccountManualModel{{ModelID: "shared-model", SourceProtocol: PlatformOpenAI}, {ModelID: "shared-model", SourceProtocol: PlatformAnthropic}}),
			targetModelID: "shared-model",
			wantReason:    "TEST_TARGET_MODEL_INVALID",
		},
		{
			name:           "source protocol and provider compatibility is enforced",
			account:        testAccount([]string{PlatformOpenAI, PlatformAnthropic}, nil, nil),
			sourceProtocol: PlatformAnthropic,
			targetProvider: PlatformOpenAI,
			wantReason:     "TEST_TARGET_PROVIDER_INCOMPATIBLE",
		},
		{
			name:           "provider without unique protocol is rejected",
			account:        testAccount([]string{PlatformAnthropic, PlatformGemini}, nil, nil),
			targetProvider: PlatformAntigravity,
			wantReason:     "TEST_PROBE_RESOLUTION_FAILED",
		},
		{
			name:           "provider without default model is rejected",
			account:        testAccount([]string{PlatformAnthropic}, nil, nil),
			targetProvider: PlatformAntigravity,
			wantReason:     "TEST_TARGET_MODEL_REQUIRED",
		},
		{
			name:           "provider plus unknown model is rejected as invalid target model",
			account:        testAccount([]string{PlatformOpenAI, PlatformAnthropic}, nil, nil),
			targetProvider: PlatformOpenAI,
			targetModelID:  "missing-model",
			wantReason:     "TEST_TARGET_MODEL_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.resolveGatewayTestTarget(context.Background(), tt.account, tt.modelID, tt.sourceProtocol, tt.targetProvider, tt.targetModelID)
			if tt.wantReason != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantReason, infraerrors.Reason(err))
				return
			}

			require.NoError(t, err)
			if tt.assert != nil {
				tt.assert(t, got)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func newGatewayResolutionTestRegistry(t *testing.T) *ModelRegistryService {
	t.Helper()

	svc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	inputs := []UpsertModelRegistryEntryInput{
		{ID: "openai-default", Provider: PlatformOpenAI, Platforms: []string{PlatformOpenAI}, ExposedIn: []string{"runtime", "test"}},
		{ID: "anthropic-default", Provider: PlatformAnthropic, Platforms: []string{PlatformAnthropic}, ExposedIn: []string{"runtime", "test"}},
		{ID: "anthropic-only", Provider: PlatformAnthropic, Platforms: []string{PlatformAnthropic}, ExposedIn: []string{"runtime", "test"}},
		{ID: "gemini-default", Provider: PlatformGemini, Platforms: []string{PlatformGemini}, ExposedIn: []string{"runtime", "test"}},
	}

	activate := make([]string, 0, len(inputs))
	for _, input := range inputs {
		_, err := svc.UpsertEntry(context.Background(), input)
		require.NoError(t, err)
		activate = append(activate, input.ID)
	}

	_, err := svc.ActivateModels(context.Background(), activate)
	require.NoError(t, err)
	return svc
}
