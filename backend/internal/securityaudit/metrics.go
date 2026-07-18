package securityaudit

import "sync/atomic"

type AtomicMetrics struct {
	allow       atomic.Int64
	flag        atomic.Int64
	block       atomic.Int64
	unavailable atomic.Int64
	timeout     atomic.Int64
	failover    atomic.Int64
	bulkhead    atomic.Int64
	enqueued    atomic.Int64
	dropped     atomic.Int64
	processed   atomic.Int64
	failed      atomic.Int64
}

func NewAtomicMetrics() *AtomicMetrics { return &AtomicMetrics{} }

func (m *AtomicMetrics) ObserveResult(result *NormalizedResult) {
	if m == nil || result == nil {
		return
	}
	switch result.Decision {
	case EventCritical:
		m.block.Add(1)
	case EventFlag:
		m.flag.Add(1)
	default:
		m.allow.Add(1)
	}
}

func (m *AtomicMetrics) IncUnavailable() {
	if m != nil {
		m.unavailable.Add(1)
	}
}

func (m *AtomicMetrics) IncTimeout() {
	if m != nil {
		m.timeout.Add(1)
	}
}

func (m *AtomicMetrics) IncFailover() {
	if m != nil {
		m.failover.Add(1)
	}
}

func (m *AtomicMetrics) IncBulkhead() {
	if m != nil {
		m.bulkhead.Add(1)
	}
}

func (m *AtomicMetrics) IncEnqueued() {
	if m != nil {
		m.enqueued.Add(1)
	}
}

func (m *AtomicMetrics) IncDropped() {
	if m != nil {
		m.dropped.Add(1)
	}
}

func (m *AtomicMetrics) IncProcessed() {
	if m != nil {
		m.processed.Add(1)
	}
}

func (m *AtomicMetrics) IncFailed() {
	if m != nil {
		m.failed.Add(1)
	}
}

func (m *AtomicMetrics) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}
	return MetricsSnapshot{
		Allow:       m.allow.Load(),
		Flag:        m.flag.Load(),
		Block:       m.block.Load(),
		Unavailable: m.unavailable.Load(),
		Timeout:     m.timeout.Load(),
		Failover:    m.failover.Load(),
		Bulkhead:    m.bulkhead.Load(),
		Enqueued:    m.enqueued.Load(),
		Dropped:     m.dropped.Load(),
		Processed:   m.processed.Load(),
		Failed:      m.failed.Load(),
	}
}
