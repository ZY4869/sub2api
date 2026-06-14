package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

type parsedBackgroundTestOutput struct {
	ResponseText            string
	ErrorMessage            string
	ResolvedModelID         string
	ResolvedPlatform        string
	ResolvedSourceProtocol  string
	BlacklistAdviceDecision string
}

// RunTestBackgroundDetailed executes an account test in-memory (no real HTTP client),
// captures the SSE output, and returns a structured result for admin actions.
func (s *AccountTestService) RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	startedAt := time.Now()

	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = (&http.Request{}).WithContext(ctx)
	if operationType := normalizeSystemUsageOperationType(input.OperationType); operationType != "" {
		ginCtx.Set(accountTestOpsProbeActionBaseContextKey, operationType)
	}

	testMode := string(normalizeAccountTestMode(input.TestMode))
	testErr := s.TestAccountConnection(
		ginCtx,
		input.AccountID,
		strings.TrimSpace(input.ModelID),
		strings.TrimSpace(input.Prompt),
		normalizeTestSourceProtocol(input.SourceProtocol),
		NormalizeModelProvider(input.TargetProvider),
		strings.TrimSpace(input.TargetModelID),
		testMode,
	)

	finishedAt := time.Now()
	parsed := parseTestSSEOutputDetailed(w.Body.String())

	status := "success"
	errMsg := parsed.ErrorMessage
	if testErr != nil || errMsg != "" {
		status = "failed"
		if errMsg == "" && testErr != nil {
			errMsg = infraerrors.Message(testErr)
			if strings.TrimSpace(errMsg) == "" {
				errMsg = testErr.Error()
			}
		}
	}

	currentLifecycleState := ""
	lifecycleReasonCode := ""
	lifecycleReasonMessage := ""
	needsReauth := false
	reauthDeadlineAt := ""
	if s != nil && s.accountRepo != nil {
		if account, err := s.accountRepo.GetByID(ctx, input.AccountID); err == nil && account != nil {
			if status == "success" {
				ClearAccountReauthState(ctx, s.accountRepo, account)
				if refreshed, refreshErr := s.accountRepo.GetByID(ctx, input.AccountID); refreshErr == nil && refreshed != nil {
					account = refreshed
				}
			}
			currentLifecycleState = account.LifecycleState
			lifecycleReasonCode = account.LifecycleReasonCode
			lifecycleReasonMessage = account.LifecycleReasonMessage
			if reauth := AccountReauthStatusFromExtra(account.Extra); reauth != nil {
				needsReauth = true
				if !reauth.DeadlineAt.IsZero() {
					reauthDeadlineAt = reauth.DeadlineAt.UTC().Format(time.RFC3339)
				}
			}
		}
	}

	result := &BackgroundAccountTestResult{
		Status:                  status,
		ResponseText:            parsed.ResponseText,
		ErrorMessage:            errMsg,
		LatencyMs:               finishedAt.Sub(startedAt).Milliseconds(),
		StartedAt:               startedAt,
		FinishedAt:              finishedAt,
		ResolvedModelID:         parsed.ResolvedModelID,
		ResolvedPlatform:        parsed.ResolvedPlatform,
		ResolvedSourceProtocol:  parsed.ResolvedSourceProtocol,
		BlacklistAdviceDecision: parsed.BlacklistAdviceDecision,
		CurrentLifecycleState:   currentLifecycleState,
		LifecycleReasonCode:     lifecycleReasonCode,
		LifecycleReasonMessage:  lifecycleReasonMessage,
		NeedsReauth:             needsReauth,
		ReauthDeadlineAt:        reauthDeadlineAt,
	}
	return result, nil
}

// RunTestBackground preserves the legacy scheduled-test result shape.
func (s *AccountTestService) RunTestBackground(ctx context.Context, input ScheduledTestExecutionInput) (*ScheduledTestResult, error) {
	result, err := s.RunTestBackgroundDetailed(ctx, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return &ScheduledTestResult{
		Status:       result.Status,
		ResponseText: result.ResponseText,
		ErrorMessage: result.ErrorMessage,
		LatencyMs:    result.LatencyMs,
		StartedAt:    result.StartedAt,
		FinishedAt:   result.FinishedAt,
	}, nil
}

// parseTestSSEOutputDetailed extracts key execution details from captured SSE output.
func parseTestSSEOutputDetailed(body string) parsedBackgroundTestOutput {
	result := parsedBackgroundTestOutput{}
	var texts []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !sseDataPrefix.MatchString(line) {
			continue
		}
		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		var event TestEvent
		if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
			continue
		}
		switch event.Type {
		case "test_start":
			if event.Model != "" {
				result.ResolvedModelID = strings.TrimSpace(event.Model)
			}
		case "content":
			if event.Text != "" {
				texts = append(texts, event.Text)
			}
			runtimeMeta, ok := event.Data.(map[string]any)
			if !ok || strings.TrimSpace(fmt.Sprint(runtimeMeta["kind"])) != "runtime_meta" {
				continue
			}
			key := strings.TrimSpace(fmt.Sprint(runtimeMeta["key"]))
			value := strings.TrimSpace(fmt.Sprint(runtimeMeta["value"]))
			switch key {
			case "resolved_platform":
				result.ResolvedPlatform = value
			case "resolved_protocol":
				result.ResolvedSourceProtocol = normalizeTestSourceProtocol(value)
			}
		case "blacklist_advice":
			if advice, ok := event.Data.(map[string]any); ok {
				result.BlacklistAdviceDecision = strings.TrimSpace(fmt.Sprint(advice["decision"]))
			}
		case "error":
			result.ErrorMessage = event.Error
		}
	}
	result.ResponseText = strings.Join(texts, "")
	return result
}
