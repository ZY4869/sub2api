package handler

import (
	"context"
	"log"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func startOpsErrorLogWorkers() {
	opsErrorLogMu.Lock()
	defer opsErrorLogMu.Unlock()

	if opsErrorLogStopping {
		return
	}

	workerCount, queueSize := opsErrorLogConfig()
	opsErrorLogQueue = make(chan opsErrorLogJob, queueSize)
	opsErrorLogQueueLen.Store(0)

	opsErrorLogWorkersWg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer opsErrorLogWorkersWg.Done()
			for {
				job, ok := <-opsErrorLogQueue
				if !ok {
					return
				}
				opsErrorLogQueueLen.Add(-1)
				batch := make([]opsErrorLogJob, 0, opsErrorLogBatchSize)
				batch = append(batch, job)

				timer := time.NewTimer(opsErrorLogBatchWindow)
			batchLoop:
				for len(batch) < opsErrorLogBatchSize {
					select {
					case nextJob, ok := <-opsErrorLogQueue:
						if !ok {
							if !timer.Stop() {
								select {
								case <-timer.C:
								default:
								}
							}
							flushOpsErrorLogBatch(batch)
							return
						}
						opsErrorLogQueueLen.Add(-1)
						batch = append(batch, nextJob)
					case <-timer.C:
						break batchLoop
					}
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				flushOpsErrorLogBatch(batch)
			}
		}()
	}
}

func flushOpsErrorLogBatch(batch []opsErrorLogJob) {
	if len(batch) == 0 {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[OpsErrorLogger] worker panic: %v\n%s", r, debug.Stack())
		}
	}()

	grouped := make(map[*service.OpsService][]*service.OpsInsertErrorLogInput, len(batch))
	var processed int64
	for _, job := range batch {
		if job.ops == nil || job.entry == nil {
			continue
		}
		grouped[job.ops] = append(grouped[job.ops], job.entry)
		processed++
	}
	if processed == 0 {
		return
	}

	for opsSvc, entries := range grouped {
		if opsSvc == nil || len(entries) == 0 {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), opsErrorLogTimeout)
		_ = opsSvc.RecordErrorBatch(ctx, entries)
		cancel()
	}
	opsErrorLogProcessed.Add(processed)
}

func enqueueOpsErrorLog(ops *service.OpsService, entry *service.OpsInsertErrorLogInput) {
	if ops == nil || entry == nil {
		return
	}
	select {
	case <-opsErrorLogShutdownCh:
		return
	default:
	}

	opsErrorLogMu.RLock()
	stopping := opsErrorLogStopping
	opsErrorLogMu.RUnlock()
	if stopping {
		return
	}

	opsErrorLogOnce.Do(startOpsErrorLogWorkers)

	opsErrorLogMu.RLock()
	defer opsErrorLogMu.RUnlock()
	if opsErrorLogStopping || opsErrorLogQueue == nil {
		return
	}

	select {
	case opsErrorLogQueue <- opsErrorLogJob{ops: ops, entry: entry}:
		opsErrorLogQueueLen.Add(1)
		opsErrorLogEnqueued.Add(1)
	default:
		// Queue is full; drop to avoid blocking request handling.
		opsErrorLogDropped.Add(1)
		maybeLogOpsErrorLogDrop()
	}
}

func StopOpsErrorLogWorkers() bool {
	opsErrorLogStopOnce.Do(func() {
		opsErrorLogShutdownOnce.Do(func() {
			close(opsErrorLogShutdownCh)
		})
		opsErrorLogDrained.Store(stopOpsErrorLogWorkers())
	})
	return opsErrorLogDrained.Load()
}

func stopOpsErrorLogWorkers() bool {
	opsErrorLogMu.Lock()
	opsErrorLogStopping = true
	ch := opsErrorLogQueue
	if ch != nil {
		close(ch)
	}
	opsErrorLogQueue = nil
	opsErrorLogMu.Unlock()

	if ch == nil {
		opsErrorLogQueueLen.Store(0)
		return true
	}

	done := make(chan struct{})
	go func() {
		opsErrorLogWorkersWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		opsErrorLogQueueLen.Store(0)
		return true
	case <-time.After(opsErrorLogDrainTimeout):
		return false
	}
}

func opsErrorLogConfig() (workerCount int, queueSize int) {
	workerCount = runtime.GOMAXPROCS(0) * 2
	if workerCount < opsErrorLogMinWorkerCount {
		workerCount = opsErrorLogMinWorkerCount
	}
	if workerCount > opsErrorLogMaxWorkerCount {
		workerCount = opsErrorLogMaxWorkerCount
	}

	queueSize = workerCount * opsErrorLogQueueSizePerWorker
	if queueSize < opsErrorLogMinQueueSize {
		queueSize = opsErrorLogMinQueueSize
	}
	if queueSize > opsErrorLogMaxQueueSize {
		queueSize = opsErrorLogMaxQueueSize
	}

	return workerCount, queueSize
}
