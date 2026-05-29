package handler

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	opsErrorLogTimeout      = 5 * time.Second
	opsErrorLogDrainTimeout = 10 * time.Second
	opsErrorLogBatchWindow  = 200 * time.Millisecond

	opsErrorLogMinWorkerCount = 4
	opsErrorLogMaxWorkerCount = 32

	opsErrorLogQueueSizePerWorker = 128
	opsErrorLogMinQueueSize       = 256
	opsErrorLogMaxQueueSize       = 8192
	opsErrorLogBatchSize          = 32
)

type opsErrorLogJob struct {
	ops   *service.OpsService
	entry *service.OpsInsertErrorLogInput
}

var (
	opsErrorLogOnce  sync.Once
	opsErrorLogQueue chan opsErrorLogJob

	opsErrorLogStopOnce  sync.Once
	opsErrorLogWorkersWg sync.WaitGroup
	opsErrorLogMu        sync.RWMutex
	opsErrorLogStopping  bool
	opsErrorLogQueueLen  atomic.Int64
	opsErrorLogEnqueued  atomic.Int64
	opsErrorLogDropped   atomic.Int64
	opsErrorLogProcessed atomic.Int64
	opsErrorLogSanitized atomic.Int64

	opsErrorLogLastDropLogAt atomic.Int64

	opsErrorLogShutdownCh   = make(chan struct{})
	opsErrorLogShutdownOnce sync.Once
	opsErrorLogDrained      atomic.Bool
)
