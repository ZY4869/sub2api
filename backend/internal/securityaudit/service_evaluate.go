package securityaudit

import (
	"context"
	"errors"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"go.uber.org/zap"
)

func (s *PromptService) Evaluate(ctx context.Context, req Request) EvaluateResult {
	if s == nil || s.config == nil {
		return EvaluateResult{Allowed: true, Mode: ModeOff}
	}
	if s.config.BlockingActivationDegraded() {
		s.metrics.IncUnavailable()
		promptAuditLog(ctx).Warn("prompt_audit_guard_failed", requestLogFields(req, "", nil, ErrorCodeUnavailable)...)
		return EvaluateResult{Mode: ModeBlocking, Error: infraerrors.ServiceUnavailable(ErrorCodeUnavailable, "prompt guard unavailable")}
	}
	cfg, ok := s.config.Active()
	mode := s.config.EffectiveMode()
	if !ok || mode == ModeOff || !cfg.IncludesGroup(req.GroupID) {
		return EvaluateResult{Allowed: true, Mode: mode}
	}
	return s.evaluateActive(ctx, cfg, mode, req)
}

func (s *PromptService) evaluateActive(ctx context.Context, cfg ActiveConfig, mode Mode, req Request) EvaluateResult {
	snapshot, err := ExtractPromptSnapshot(req)
	if errors.Is(err, ErrNoPromptText) {
		return EvaluateResult{Allowed: true, Mode: mode}
	}
	if err != nil {
		promptAuditLog(ctx).Warn("prompt_audit_guard_failed", requestLogFields(req, "", nil, ErrorCodeInvalidResponse)...)
		return EvaluateResult{Mode: mode, Error: infraerrors.BadRequest(ErrorCodeInvalidResponse, "prompt audit request is invalid")}
	}
	if mode == ModeBlocking {
		return s.evaluateBlocking(ctx, cfg, snapshot, req)
	}
	s.enqueueAsyncBestEffort(cfg, snapshot, req)
	return EvaluateResult{Allowed: true, Mode: mode}
}

func (s *PromptService) evaluateBlocking(ctx context.Context, cfg ActiveConfig, snapshot PromptSnapshot, req Request) EvaluateResult {
	job := jobFromSnapshot(snapshot, req, ModeBlocking, cfg.ConfigVersion)
	result, err := scanWithFailover(ctx, s.scanner, cfg, snapshot.ScanText, s.metrics)
	if err != nil {
		code := guardErrorCode(err)
		promptAuditLog(ctx).Warn("prompt_audit_guard_failed", requestLogFields(req, snapshot.PromptHash, nil, code)...)
		return EvaluateResult{Mode: ModeBlocking, Error: infraerrors.ServiceUnavailable(code, "prompt guard unavailable")}
	}
	event, persistErr := s.persistBlockingEvent(ctx, cfg, snapshot, req, job, result)
	if persistErr != nil {
		promptAuditLog(ctx).Warn("prompt_audit_guard_failed", requestLogFields(req, snapshot.PromptHash, result, ErrorCodeUnavailable)...)
		return EvaluateResult{Mode: ModeBlocking, Error: infraerrors.ServiceUnavailable(ErrorCodeUnavailable, "prompt guard unavailable")}
	}
	if result.Action == ActionBlock {
		promptAuditLog(ctx).Warn("prompt_audit_guard_blocked", requestLogFields(req, snapshot.PromptHash, result, ErrorCodeBlocked)...)
		return EvaluateResult{Mode: ModeBlocking, Decision: result.Decision, Event: event, Error: infraerrors.Forbidden(ErrorCodeBlocked, "prompt blocked by guard")}
	}
	promptAuditLog(ctx).Info("prompt_audit_guard_allowed", requestLogFields(req, snapshot.PromptHash, result, "")...)
	return EvaluateResult{Allowed: true, Mode: ModeBlocking, Decision: result.Decision, Event: event}
}

func (s *PromptService) persistBlockingEvent(ctx context.Context, cfg ActiveConfig, snapshot PromptSnapshot, req Request, job *Job, result *NormalizedResult) (*Event, error) {
	if result.Decision == EventPass && !cfg.StorePassEvents {
		return nil, nil
	}
	if createErr := s.repo.CreateJob(ctx, job); createErr != nil {
		return nil, createErr
	}
	event, err := s.repo.CreateEvent(ctx, job, result, snapshot.FullPrompt)
	if err != nil {
		fields := requestLogFields(req, snapshot.PromptHash, result, "prompt_audit_event_create_failed")
		promptAuditLog(ctx).Warn("prompt_audit_blocking_event_failed", fields...)
	}
	_ = s.repo.MarkJobDone(ctx, job.ID)
	return event, nil
}

func (s *PromptService) enqueueAsyncBestEffort(cfg ActiveConfig, snapshot PromptSnapshot, req Request) {
	if s == nil {
		return
	}
	select {
	case s.enqueueSlots <- struct{}{}:
	default:
		s.metrics.IncBulkhead()
		s.metrics.IncDropped()
		fields := requestLogFields(req, snapshot.PromptHash, nil, "prompt_audit_enqueue_busy")
		promptAuditLog(context.Background()).Warn("prompt_audit_enqueue_dropped", fields...)
		return
	}
	background := s.enqueueBackground()
	s.enqueueWG.Add(1)
	go func() {
		defer s.enqueueWG.Done()
		defer func() { <-s.enqueueSlots }()
		ctx, cancel := context.WithTimeout(background, 2*time.Second)
		defer cancel()
		s.enqueueAsync(ctx, cfg, snapshot, req)
	}()
}

func (s *PromptService) enqueueBackground() context.Context {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.background != nil {
		return s.background
	}
	return context.Background()
}

func (s *PromptService) enqueueAsync(ctx context.Context, cfg ActiveConfig, snapshot PromptSnapshot, req Request) {
	job := jobFromSnapshot(snapshot, req, ModeAsync, cfg.ConfigVersion)
	if err := s.repo.CreateJob(ctx, job); err != nil {
		s.metrics.IncDropped()
		promptAuditLog(ctx).Warn("prompt_audit_enqueue_failed", requestLogFields(req, snapshot.PromptHash, nil, "prompt_audit_enqueue_failed")...)
		return
	}
	payload := PromptPayload{ScanText: snapshot.ScanText, FullPrompt: snapshot.FullPrompt, ClientRequestID: req.ClientRequestID}
	ttl := time.Duration(DefaultPayloadTTLSeconds) * time.Second
	if err := s.payload.Set(ctx, job.ID, payload, ttl); err != nil {
		s.metrics.IncDropped()
		_ = s.repo.MarkJobFailed(ctx, job, "prompt_audit_payload_store_failed", "prompt audit payload store failed", false)
		promptAuditLog(ctx).Warn("prompt_audit_enqueue_failed", requestLogFields(req, snapshot.PromptHash, nil, "prompt_audit_payload_store_failed")...)
		return
	}
	s.metrics.IncEnqueued()
	promptAuditLog(ctx).Info("prompt_audit_guard_enqueued", append(
		requestLogFields(req, snapshot.PromptHash, nil, ""),
		zap.Int64("job_id", job.ID),
	)...)
}

func jobFromSnapshot(snapshot PromptSnapshot, req Request, mode Mode, configVersion int64) *Job {
	return &Job{
		Request: req, PromptHash: snapshot.PromptHash, RedactedPreview: snapshot.RedactedPreview,
		PromptLength: snapshot.PromptLength, MessageCount: snapshot.MessageCount,
		ExecutionMode: string(mode), ConfigVersion: configVersion, NextAttemptAt: time.Now().UTC(),
	}
}
