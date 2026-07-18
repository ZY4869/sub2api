package securityaudit

import "context"

func (s *PromptService) Runtime(ctx context.Context) RuntimeSnapshot {
	expected, activeVersion, loadedAt, loadErr := s.config.RuntimeState()
	cfg, ok := s.config.Active()
	queue, dbStatus := s.runtimeQueue(ctx)
	redisStatus := s.runtimeRedis(ctx)
	active, heartbeat, lastProcessed, workerErr := s.worker.Snapshot()
	metrics := s.metrics.Snapshot()
	workerTotal, capacity := 0, 0
	if ok {
		workerTotal, capacity = cfg.WorkerCount, cfg.QueueCapacity
	}
	return RuntimeSnapshot{
		ProcessStatus:         runtimeStatus(s.config.EffectiveMode(), loadErr, dbStatus, redisStatus),
		EffectiveMode:         s.config.EffectiveMode(),
		ExpectedConfigVersion: expected,
		ActiveConfigVersion:   activeVersion,
		ConfigLoadedAt:        loadedAt,
		ConfigLoadError:       loadErr,
		WorkerTotal:           workerTotal,
		WorkerActive:          active,
		QueueCapacity:         capacity,
		Queue:                 queue,
		DatabaseStatus:        dbStatus,
		RedisStatus:           redisStatus,
		Endpoints:             s.probeSnapshot(),
		GuardMetrics:          metrics,
		EnqueuedTotal:         metrics.Enqueued,
		DroppedTotal:          metrics.Dropped,
		ProcessedTotal:        metrics.Processed,
		FailedTotal:           metrics.Failed,
		WorkerHeartbeatAt:     heartbeat,
		LastProcessedAt:       lastProcessed,
		LastErrorCode:         workerErr,
	}
}

func (s *PromptService) runtimeQueue(ctx context.Context) (QueueStats, string) {
	if stats, err := s.repo.QueueStats(ctx); err == nil {
		return stats, "ok"
	}
	return QueueStats{}, "error"
}

func (s *PromptService) runtimeRedis(ctx context.Context) string {
	if s.payload == nil || s.payload.Ping(ctx) != nil {
		return "error"
	}
	return "ok"
}

func runtimeStatus(mode Mode, loadErr, dbStatus, redisStatus string) string {
	if mode == ModeOff {
		return "disabled"
	}
	if loadErr != "" || dbStatus != "ok" || redisStatus != "ok" {
		return "degraded"
	}
	return "running"
}
