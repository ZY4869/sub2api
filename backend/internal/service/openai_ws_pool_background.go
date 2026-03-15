package service

import (
	"time"

	"golang.org/x/sync/errgroup"
)

func (p *openAIWSConnPool) startBackgroundWorkers() {
	if p == nil || p.workerStopCh == nil {
		return
	}
	p.workerWg.Add(2)
	go func() {
		defer p.workerWg.Done()
		p.runBackgroundPingWorker()
	}()
	go func() {
		defer p.workerWg.Done()
		p.runBackgroundCleanupWorker()
	}()
}

type openAIWSIdlePingCandidate struct {
	accountID int64
	conn      *openAIWSConn
}

func (p *openAIWSConnPool) runBackgroundPingWorker() {
	if p == nil {
		return
	}
	ticker := time.NewTicker(openAIWSBackgroundPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.runBackgroundPingSweep()
		case <-p.workerStopCh:
			return
		}
	}
}

func (p *openAIWSConnPool) runBackgroundPingSweep() {
	if p == nil {
		return
	}
	candidates := p.snapshotIdleConnsForPing()
	var g errgroup.Group
	g.SetLimit(10)
	for _, item := range candidates {
		item := item
		if item.conn == nil || item.conn.isLeased() || item.conn.waiters.Load() > 0 {
			continue
		}
		g.Go(func() error {
			if err := item.conn.pingWithTimeout(openAIWSConnHealthCheckTO); err != nil {
				p.evictConn(item.accountID, item.conn.id)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (p *openAIWSConnPool) snapshotIdleConnsForPing() []openAIWSIdlePingCandidate {
	if p == nil {
		return nil
	}
	candidates := make([]openAIWSIdlePingCandidate, 0)
	p.accounts.Range(func(key, value any) bool {
		accountID, ok := key.(int64)
		if !ok || accountID <= 0 {
			return true
		}
		ap, ok := value.(*openAIWSAccountPool)
		if !ok || ap == nil {
			return true
		}
		ap.mu.Lock()
		for _, conn := range ap.conns {
			if conn == nil || conn.isLeased() || conn.waiters.Load() > 0 {
				continue
			}
			candidates = append(candidates, openAIWSIdlePingCandidate{
				accountID: accountID,
				conn:      conn,
			})
		}
		ap.mu.Unlock()
		return true
	})
	return candidates
}

func (p *openAIWSConnPool) runBackgroundCleanupWorker() {
	if p == nil {
		return
	}
	ticker := time.NewTicker(openAIWSBackgroundSweepTicker)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.runBackgroundCleanupSweep(time.Now())
		case <-p.workerStopCh:
			return
		}
	}
}

func (p *openAIWSConnPool) runBackgroundCleanupSweep(now time.Time) {
	if p == nil {
		return
	}
	type cleanupResult struct {
		evicted []*openAIWSConn
	}
	results := make([]cleanupResult, 0)
	p.accounts.Range(func(_ any, value any) bool {
		ap, ok := value.(*openAIWSAccountPool)
		if !ok || ap == nil {
			return true
		}
		maxConns := p.maxConnsHardCap()
		ap.mu.Lock()
		if ap.lastAcquire != nil && ap.lastAcquire.Account != nil {
			maxConns = p.effectiveMaxConnsByAccount(ap.lastAcquire.Account)
		}
		evicted := p.cleanupAccountLocked(ap, now, maxConns)
		ap.lastCleanupAt = now
		ap.mu.Unlock()
		if len(evicted) > 0 {
			results = append(results, cleanupResult{evicted: evicted})
		}
		return true
	})
	for _, result := range results {
		closeOpenAIWSConns(result.evicted)
	}
}
