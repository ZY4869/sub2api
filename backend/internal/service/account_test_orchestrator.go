package service

import (
	"log/slog"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/gin-gonic/gin"
)

func (s *AccountTestService) TestAccountConnection(c *gin.Context, accountID int64, modelID string, prompt string, sourceProtocol string, targetProvider string, targetModelID string, testMode string) error {
	if c != nil && c.Request != nil {
		ctxWithMetadata := EnsureRequestMetadata(c.Request.Context())
		c.Request = c.Request.WithContext(ctxWithMetadata)
	}
	ctx := c.Request.Context()

	// Best-effort: attach an ops trace collector (if the caller has provided a test_run_id).
	s.ensureOpsCollector(c)

	runtimeMeta := accountTestRuntimeMeta{}
	normalizedTestMode := AccountTestModeHealthCheck
	var testErr error
	startedAt := time.Now()
	defer func() {
		s.recordSystemUsageForTestConnection(c, accountID, modelID, runtimeMeta, normalizedTestMode, startedAt, testErr)
		s.finalizeUpstreamTrace(c, accountID, runtimeMeta, modelID, normalizedTestMode, testErr)
	}()

	// Get account
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		testErr = s.sendErrorAndEnd(c, "Account not found")
		return testErr
	}
	if err := EnsureSupportedAccountPlatform(account); err != nil {
		testErr = err
		return testErr
	}

	if !IsProtocolGatewayAccount(account) && strings.TrimSpace(modelID) == "" {
		if defaultModelID, defaultSourceProtocol := s.resolveRestrictedDefaultTestModel(ctx, account); defaultModelID != "" {
			modelID = defaultModelID
			if strings.TrimSpace(sourceProtocol) == "" {
				sourceProtocol = defaultSourceProtocol
			}
		}
	}

	resolvedTarget, err := s.resolveGatewayTestTarget(ctx, account, modelID, sourceProtocol, targetProvider, targetModelID)
	if err != nil {
		reason := infraerrors.Reason(err)
		if reason == "TEST_PROBE_RESOLUTION_FAILED" {
			protocolruntime.RecordAccountProbeResolutionFailed(reason)
			slog.Warn(
				"account_probe_resolution_failed",
				"account_id", accountID,
				"requested_model_id", strings.TrimSpace(modelID),
				"source_protocol", normalizeTestSourceProtocol(sourceProtocol),
				"target_provider", NormalizeModelProvider(targetProvider),
				"target_model_id", strings.TrimSpace(targetModelID),
				"reason", reason,
				"error", err,
			)
		} else {
			protocolruntime.RecordAccountTestResolutionFailed(reason)
		}
		slog.Warn(
			"account_test_resolution_failed",
			"account_id", accountID,
			"requested_model_id", strings.TrimSpace(modelID),
			"source_protocol", normalizeTestSourceProtocol(sourceProtocol),
			"target_provider", NormalizeModelProvider(targetProvider),
			"target_model_id", strings.TrimSpace(targetModelID),
			"reason", reason,
			"error", err,
		)
		testErr = err
		return testErr
	}
	if resolvedTarget.SourceProtocol != "" {
		account = ResolveProtocolGatewayInboundAccount(account, resolvedTarget.SourceProtocol)
	}
	if resolvedTarget.ModelID != "" {
		modelID = resolvedTarget.ModelID
	}
	if err := s.ensureAllowedTestModel(ctx, account, modelID); err != nil {
		slog.Warn(
			"account_test_model_not_allowed",
			"account_id", accountID,
			"requested_model_id", strings.TrimSpace(modelID),
			"source_protocol", normalizeTestSourceProtocol(resolvedTarget.SourceProtocol),
			"target_provider", NormalizeModelProvider(resolvedTarget.TargetProvider),
			"target_model_id", strings.TrimSpace(resolvedTarget.TargetModelID),
			"error", err,
		)
		testErr = err
		return testErr
	}
	if account != nil && account.IsOpenAI() && !isOpenAIGPTImageProfileModelID(modelID) {
		modelID = resolveOpenAITestModelID(ctx, account, modelID, s.modelRegistryService)
	}
	simulatedClient := s.resolveGatewayTestSimulatedClient(ctx, account, resolvedTarget.SourceProtocol, modelID)
	normalizedTestMode = normalizeAccountTestMode(testMode)
	if account != nil && account.IsBaiduDocumentAI() {
		normalizedTestMode = AccountTestModeHealthCheck
	}
	runtimeMeta = buildAccountTestRuntimeMeta(
		account,
		normalizedTestMode,
		resolvedTarget.SourceProtocol,
		resolvedTarget.TargetProvider,
		resolvedTarget.TargetModelID,
		modelID,
		simulatedClient,
	)
	s.setResolvedTestRuntimeMeta(c, runtimeMeta)
	slog.Info(
		"account_test_start",
		"account_id", accountID,
		"test_mode", string(normalizedTestMode),
		"inbound_endpoint", runtimeMeta.InboundEndpoint,
		"source_protocol", runtimeMeta.SourceProtocol,
		"target_provider", runtimeMeta.TargetProvider,
		"target_model_id", runtimeMeta.TargetModelID,
		"resolved_model_id", runtimeMeta.ResolvedModelID,
		"compat_path", runtimeMeta.CompatPath,
		"runtime_platform", runtimeMeta.RuntimePlatform,
		"simulated_client", runtimeMeta.SimulatedClient,
	)

	if normalizedTestMode == AccountTestModeRealForward {
		testErr = s.testAccountConnectionRealForward(c, account, modelID, prompt, resolvedTarget.SourceProtocol, simulatedClient)
	} else {
		testErr = s.testAccountConnectionHealthCheck(c, account, modelID, prompt, resolvedTarget.SourceProtocol, simulatedClient)
	}
	if testErr != nil {
		slog.Warn(
			"account_test_complete",
			"account_id", accountID,
			"status", "failed",
			"test_mode", string(normalizedTestMode),
			"inbound_endpoint", runtimeMeta.InboundEndpoint,
			"source_protocol", runtimeMeta.SourceProtocol,
			"target_provider", runtimeMeta.TargetProvider,
			"target_model_id", runtimeMeta.TargetModelID,
			"resolved_model_id", runtimeMeta.ResolvedModelID,
			"compat_path", runtimeMeta.CompatPath,
			"runtime_platform", runtimeMeta.RuntimePlatform,
			"error", testErr,
		)
		return testErr
	}
	slog.Info(
		"account_test_complete",
		"account_id", accountID,
		"status", "success",
		"test_mode", string(normalizedTestMode),
		"inbound_endpoint", runtimeMeta.InboundEndpoint,
		"source_protocol", runtimeMeta.SourceProtocol,
		"target_provider", runtimeMeta.TargetProvider,
		"target_model_id", runtimeMeta.TargetModelID,
		"resolved_model_id", runtimeMeta.ResolvedModelID,
		"compat_path", runtimeMeta.CompatPath,
		"runtime_platform", runtimeMeta.RuntimePlatform,
	)
	return nil
}

func (s *AccountTestService) testAccountConnectionHealthCheck(c *gin.Context, account *Account, modelID string, prompt string, resolvedSourceProtocol string, simulatedClient string) error {
	if account == nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	if account.IsBaiduDocumentAI() {
		return s.testBaiduDocumentAIAccountConnection(c, account)
	}

	if account.IsOpenAI() || account.IsDeepSeek() {
		if isOpenAIGPTImageProfileModelID(modelID) {
			return s.testOpenAIImageAccountConnection(c, account, modelID, prompt, resolvedSourceProtocol, simulatedClient)
		}
		return s.testOpenAIAccountConnection(c, account, modelID, prompt, resolvedSourceProtocol, simulatedClient)
	}

	if account.IsGrok() {
		return s.testGrokAccountConnection(c, account, modelID)
	}

	if account.IsGemini() {
		return s.testGeminiAccountConnection(c, account, modelID, prompt, resolvedSourceProtocol, simulatedClient)
	}

	if RoutingPlatformForAccount(account) == PlatformAntigravity {
		return s.routeAntigravityTest(c, account, modelID, prompt)
	}

	return s.testClaudeAccountConnection(c, account, modelID, resolvedSourceProtocol, simulatedClient)
}
