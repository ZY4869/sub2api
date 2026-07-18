package securityaudit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type Worker struct {
	config  *ConfigManager
	repo    promptRepository
	payload PayloadStore
	scanner PromptScanner
	metrics *AtomicMetrics

	cancel context.CancelFunc
	wg     sync.WaitGroup
	active atomic.Int64
	beatNS atomic.Int64
	lastNS atomic.Int64
	errMu  sync.RWMutex
	err    string
}

func NewWorker(config *ConfigManager, repo promptRepository, payload PayloadStore, scanner PromptScanner, metrics *AtomicMetrics) *Worker {
	return &Worker{config: config, repo: repo, payload: payload, scanner: scanner, metrics: metrics}
}

func (w *Worker) Start(ctx context.Context) {
	if w == nil || w.cancel != nil {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	for i := 0; i < MaxWorkerCount; i++ {
		w.wg.Add(1)
		go w.loop(runCtx, i)
	}
}

func (w *Worker) Stop() {
	if w == nil {
		return
	}
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func (w *Worker) Snapshot() (active int64, heartbeat, lastProcessed *time.Time, errCode string) {
	if w == nil {
		return
	}
	active = w.active.Load()
	if ns := w.beatNS.Load(); ns > 0 {
		t := time.Unix(0, ns).UTC()
		heartbeat = &t
	}
	if ns := w.lastNS.Load(); ns > 0 {
		t := time.Unix(0, ns).UTC()
		lastProcessed = &t
	}
	w.errMu.RLock()
	errCode = w.err
	w.errMu.RUnlock()
	return
}

func (w *Worker) loop(ctx context.Context, workerID int) {
	defer w.wg.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.beatNS.Store(time.Now().UTC().UnixNano())
			cfg, ok := w.config.Active()
			if !ok || cfg.EffectiveMode() != ModeAsync || workerID >= cfg.WorkerCount {
				continue
			}
			w.claimAndProcess(ctx, cfg, workerID)
		}
	}
}

func (w *Worker) claimAndProcess(ctx context.Context, cfg ActiveConfig, workerID int) {
	jobs, err := w.repo.ClaimJobs(ctx, 8)
	if err != nil {
		w.setError("prompt_audit_claim_failed")
		promptAuditLog(ctx).Warn("prompt_audit_worker_claim_failed", zap.String("error_code", "prompt_audit_claim_failed"))
		return
	}
	for _, job := range jobs {
		w.active.Add(1)
		if err := w.process(ctx, cfg, job, workerID); err != nil {
			w.metrics.IncFailed()
		} else {
			w.metrics.IncProcessed()
			w.lastNS.Store(time.Now().UTC().UnixNano())
		}
		w.active.Add(-1)
	}
}

func (w *Worker) process(ctx context.Context, cfg ActiveConfig, job *Job, workerID int) error {
	payload, err := w.payload.Get(ctx, job.ID)
	if err != nil {
		return w.fail(ctx, job, "prompt_audit_payload_missing", err, false)
	}
	if job.ClientRequestID == "" {
		job.ClientRequestID = payload.ClientRequestID
	}
	result, err := scanWithFailover(ctx, w.scanner, cfg, payload.ScanText, w.metrics)
	if err != nil {
		var guardErr *GuardError
		retry := errors.As(err, &guardErr) && guardErr.Retryable
		return w.fail(ctx, job, guardErrorCode(err), err, retry)
	}
	if result.Decision == EventPass && !cfg.StorePassEvents {
		_ = w.payload.Delete(ctx, job.ID)
		return w.repo.MarkJobDone(ctx, job.ID)
	}
	fullPrompt := payload.FullPrompt
	if _, err := w.repo.CreateEvent(ctx, job, result, fullPrompt); err != nil {
		return w.fail(ctx, job, "prompt_audit_event_create_failed", err, true)
	}
	_ = w.payload.Delete(ctx, job.ID)
	if err := w.repo.MarkJobDone(ctx, job.ID); err != nil {
		return err
	}
	fields := append(jobLogFields(job, result, ""), zap.Int("worker_id", workerID))
	promptAuditLog(ctx).Info("prompt_audit_worker_done", fields...)
	return nil
}

func (w *Worker) fail(ctx context.Context, job *Job, code string, err error, retry bool) error {
	w.setError(code)
	_ = w.repo.MarkJobFailed(ctx, job, code, "prompt audit worker failed", retry)
	message := "prompt_audit_worker_failed"
	if retry {
		message = "prompt_audit_worker_retry"
	}
	fields := append(jobLogFields(job, nil, code), zap.Bool("retry", retry))
	promptAuditLog(ctx).Warn(message, fields...)
	return err
}

func (w *Worker) setError(code string) {
	w.errMu.Lock()
	w.err = code
	w.errMu.Unlock()
}
