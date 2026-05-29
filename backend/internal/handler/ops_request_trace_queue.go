package handler

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	opsRequestTraceTimeout      = 5 * time.Second
	opsRequestTraceWorkerMin    = 2
	opsRequestTraceWorkerMax    = 16
	opsRequestTraceQueueMin     = 256
	opsRequestTraceQueueMax     = 4096
	opsRequestTraceQueuePerWork = 64
	opsRequestTraceBodyLimit    = 1024 * 1024
	opsRequestTracePreviewLimit = 64 * 1024
	opsRequestTraceSampleRate   = 0.1
	opsRequestTraceSlowMs       = int64(3000)
)

type opsRequestTraceJob struct {
	ops   *service.OpsService
	input *service.OpsRecordRequestTraceInput
}

var (
	opsRequestTraceOnce  sync.Once
	opsRequestTraceQueue chan opsRequestTraceJob

	opsRequestTraceQueueLen atomic.Int64
	opsRequestTraceDropped  atomic.Int64
	opsRequestTraceStop     atomic.Bool
	opsRequestTraceWorkers  sync.WaitGroup
)

func startOpsRequestTraceWorkers() {
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < opsRequestTraceWorkerMin {
		workerCount = opsRequestTraceWorkerMin
	}
	if workerCount > opsRequestTraceWorkerMax {
		workerCount = opsRequestTraceWorkerMax
	}

	queueSize := workerCount * opsRequestTraceQueuePerWork
	if queueSize < opsRequestTraceQueueMin {
		queueSize = opsRequestTraceQueueMin
	}
	if queueSize > opsRequestTraceQueueMax {
		queueSize = opsRequestTraceQueueMax
	}

	opsRequestTraceQueue = make(chan opsRequestTraceJob, queueSize)
	opsRequestTraceWorkers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer opsRequestTraceWorkers.Done()
			for job := range opsRequestTraceQueue {
				opsRequestTraceQueueLen.Add(-1)
				if job.ops == nil || job.input == nil {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), opsRequestTraceTimeout)
				_ = job.ops.RecordRequestTrace(ctx, job.input)
				cancel()
			}
		}()
	}
}

func enqueueOpsRequestTrace(ops *service.OpsService, input *service.OpsRecordRequestTraceInput) {
	if ops == nil || input == nil || opsRequestTraceStop.Load() {
		return
	}
	opsRequestTraceOnce.Do(startOpsRequestTraceWorkers)
	select {
	case opsRequestTraceQueue <- opsRequestTraceJob{ops: ops, input: input}:
		opsRequestTraceQueueLen.Add(1)
	default:
		opsRequestTraceDropped.Add(1)
		log.Printf("[OpsRequestTrace] queue is full; dropping request trace (dropped_total=%d)", opsRequestTraceDropped.Load())
	}
}
