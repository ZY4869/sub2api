package handler

import (
	"log"
	"time"
)

func OpsErrorLogQueueLength() int64 {
	return opsErrorLogQueueLen.Load()
}

func OpsErrorLogQueueCapacity() int {
	opsErrorLogMu.RLock()
	ch := opsErrorLogQueue
	opsErrorLogMu.RUnlock()
	if ch == nil {
		return 0
	}
	return cap(ch)
}

func OpsErrorLogDroppedTotal() int64 {
	return opsErrorLogDropped.Load()
}

func OpsErrorLogEnqueuedTotal() int64 {
	return opsErrorLogEnqueued.Load()
}

func OpsErrorLogProcessedTotal() int64 {
	return opsErrorLogProcessed.Load()
}

func OpsErrorLogSanitizedTotal() int64 {
	return opsErrorLogSanitized.Load()
}

func maybeLogOpsErrorLogDrop() {
	now := time.Now().Unix()

	for {
		last := opsErrorLogLastDropLogAt.Load()
		if last != 0 && now-last < 60 {
			return
		}
		if opsErrorLogLastDropLogAt.CompareAndSwap(last, now) {
			break
		}
	}

	queued := opsErrorLogQueueLen.Load()
	queueCap := OpsErrorLogQueueCapacity()

	log.Printf(
		"[OpsErrorLogger] queue is full; dropping logs (queued=%d cap=%d enqueued_total=%d dropped_total=%d processed_total=%d sanitized_total=%d)",
		queued,
		queueCap,
		opsErrorLogEnqueued.Load(),
		opsErrorLogDropped.Load(),
		opsErrorLogProcessed.Load(),
		opsErrorLogSanitized.Load(),
	)
}
