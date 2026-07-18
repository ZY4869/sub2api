package securityaudit

import (
	"context"
	"errors"
	"time"
)

type PromptScanner interface {
	Scan(ctx context.Context, endpoint ActiveEndpoint, chunk string, enabledScanners []string) (*NormalizedResult, error)
}

func scanWithFailover(ctx context.Context, scanner PromptScanner, cfg ActiveConfig, text string, metrics *AtomicMetrics) (*NormalizedResult, error) {
	endpoints := cfg.EnabledEndpoints()
	if scanner == nil || len(endpoints) == 0 {
		return nil, &GuardError{Code: ErrorCodeUnavailable, Retryable: true}
	}
	started := time.Now()
	var lastErr error
	for i, endpoint := range endpoints {
		chunk := TrimRunes(text, endpoint.InputLimit)
		result, err := scanner.Scan(ctx, endpoint, chunk, cfg.Scanners)
		if err == nil && result != nil {
			result.LatencyMS = int(time.Since(started).Milliseconds())
			metrics.ObserveResult(result)
			return result, nil
		}
		if err == nil {
			err = &GuardError{Code: ErrorCodeInvalidResponse}
		}
		lastErr = err
		var guardErr *GuardError
		if !errors.As(err, &guardErr) || !guardErr.Retryable {
			break
		}
		if i < len(endpoints)-1 {
			metrics.IncFailover()
		}
	}
	observeGuardFailure(metrics, lastErr)
	return nil, lastErr
}

func observeGuardFailure(metrics *AtomicMetrics, err error) {
	if metrics == nil {
		return
	}
	metrics.IncUnavailable()
	var guardErr *GuardError
	if errors.As(err, &guardErr) && guardErr.Timeout {
		metrics.IncTimeout()
	}
}

func guardErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var guardErr *GuardError
	if errors.As(err, &guardErr) && guardErr.Code != "" {
		return guardErr.Code
	}
	return ErrorCodeUnavailable
}
