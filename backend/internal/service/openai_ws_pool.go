package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type openAIWSAccountPool struct {
	mu            sync.Mutex
	conns         map[string]*openAIWSConn
	pinnedConns   map[string]int
	creating      int
	lastCleanupAt time.Time
	lastAcquire   *openAIWSAcquireRequest
	prewarmActive bool
	prewarmUntil  time.Time
	prewarmFails  int
	prewarmFailAt time.Time
}

type openAIWSConnPool struct {
	cfg *config.Config
	// 通过接口解耦底层 WS 客户端实现，默认使用 coder/websocket。
	clientDialer openAIWSClientDialer

	accounts sync.Map // key: int64(accountID), value: *openAIWSAccountPool
	seq      atomic.Uint64

	metrics openAIWSPoolMetrics

	workerStopCh chan struct{}
	workerWg     sync.WaitGroup
	closeOnce    sync.Once
}

func newOpenAIWSConnPool(cfg *config.Config) *openAIWSConnPool {
	pool := &openAIWSConnPool{
		cfg:          cfg,
		clientDialer: newDefaultOpenAIWSClientDialer(),
		workerStopCh: make(chan struct{}),
	}
	pool.startBackgroundWorkers()
	return pool
}

func (p *openAIWSConnPool) setClientDialerForTest(dialer openAIWSClientDialer) {
	if p == nil || dialer == nil {
		return
	}
	p.clientDialer = dialer
}

// Close 停止后台 worker 并关闭所有空闲连接，应在优雅关闭时调用。
func (p *openAIWSConnPool) Close() {
	if p == nil {
		return
	}
	p.closeOnce.Do(func() {
		if p.workerStopCh != nil {
			close(p.workerStopCh)
		}
		p.workerWg.Wait()
		// 遍历所有账户池，关闭全部空闲连接。
		p.accounts.Range(func(key, value any) bool {
			ap, ok := value.(*openAIWSAccountPool)
			if !ok || ap == nil {
				return true
			}
			ap.mu.Lock()
			for _, conn := range ap.conns {
				if conn != nil && !conn.isLeased() {
					conn.close()
				}
			}
			ap.mu.Unlock()
			return true
		})
	})
}

func (p *openAIWSConnPool) Acquire(ctx context.Context, req openAIWSAcquireRequest) (*openAIWSConnLease, error) {
	if p != nil {
		p.metrics.acquireTotal.Add(1)
	}
	return p.acquire(ctx, cloneOpenAIWSAcquireRequest(req), 0)
}

func (p *openAIWSConnPool) acquire(ctx context.Context, req openAIWSAcquireRequest, retry int) (*openAIWSConnLease, error) {
	if p == nil || req.Account == nil || req.Account.ID <= 0 {
		return nil, errors.New("invalid ws acquire request")
	}
	if stringsTrim(req.WSURL) == "" {
		return nil, errors.New("ws url is empty")
	}

	accountID := req.Account.ID
	effectiveMaxConns := p.effectiveMaxConnsByAccount(req.Account)
	if effectiveMaxConns <= 0 {
		return nil, errOpenAIWSConnQueueFull
	}
	var evicted []*openAIWSConn
	ap := p.getOrCreateAccountPool(accountID)
	ap.mu.Lock()
	ap.lastAcquire = cloneOpenAIWSAcquireRequestPtr(&req)
	now := time.Now()
	if ap.lastCleanupAt.IsZero() || now.Sub(ap.lastCleanupAt) >= openAIWSAcquireCleanupInterval {
		evicted = p.cleanupAccountLocked(ap, now, effectiveMaxConns)
		ap.lastCleanupAt = now
	}
	pickStartedAt := time.Now()
	allowReuse := !req.ForceNewConn
	preferredConnID := stringsTrim(req.PreferredConnID)
	forcePreferredConn := allowReuse && req.ForcePreferredConn

	if allowReuse {
		if forcePreferredConn {
			if preferredConnID == "" {
				p.recordConnPickDuration(time.Since(pickStartedAt))
				ap.mu.Unlock()
				closeOpenAIWSConns(evicted)
				return nil, errOpenAIWSPreferredConnUnavailable
			}
			preferredConn, ok := ap.conns[preferredConnID]
			if !ok || preferredConn == nil {
				p.recordConnPickDuration(time.Since(pickStartedAt))
				ap.mu.Unlock()
				closeOpenAIWSConns(evicted)
				return nil, errOpenAIWSPreferredConnUnavailable
			}
			if preferredConn.tryAcquire() {
				connPick := time.Since(pickStartedAt)
				p.recordConnPickDuration(connPick)
				ap.mu.Unlock()
				closeOpenAIWSConns(evicted)
				if p.shouldHealthCheckConn(preferredConn) {
					if err := preferredConn.pingWithTimeout(openAIWSConnHealthCheckTO); err != nil {
						preferredConn.close()
						p.evictConn(accountID, preferredConn.id)
						if retry < 1 {
							return p.acquire(ctx, req, retry+1)
						}
						return nil, err
					}
				}
				lease := &openAIWSConnLease{
					pool:      p,
					accountID: accountID,
					conn:      preferredConn,
					connPick:  connPick,
					reused:    true,
				}
				p.metrics.acquireReuseTotal.Add(1)
				p.ensureTargetIdleAsync(accountID)
				return lease, nil
			}

			connPick := time.Since(pickStartedAt)
			p.recordConnPickDuration(connPick)
			if int(preferredConn.waiters.Load()) >= p.queueLimitPerConn() {
				ap.mu.Unlock()
				closeOpenAIWSConns(evicted)
				return nil, errOpenAIWSConnQueueFull
			}
			preferredConn.waiters.Add(1)
			ap.mu.Unlock()
			closeOpenAIWSConns(evicted)
			defer preferredConn.waiters.Add(-1)
			waitStart := time.Now()
			p.metrics.acquireQueueWaitTotal.Add(1)

			if err := preferredConn.acquire(ctx); err != nil {
				if errors.Is(err, errOpenAIWSConnClosed) && retry < 1 {
					return p.acquire(ctx, req, retry+1)
				}
				return nil, err
			}
			if p.shouldHealthCheckConn(preferredConn) {
				if err := preferredConn.pingWithTimeout(openAIWSConnHealthCheckTO); err != nil {
					preferredConn.release()
					preferredConn.close()
					p.evictConn(accountID, preferredConn.id)
					if retry < 1 {
						return p.acquire(ctx, req, retry+1)
					}
					return nil, err
				}
			}

			queueWait := time.Since(waitStart)
			p.metrics.acquireQueueWaitMs.Add(queueWait.Milliseconds())
			lease := &openAIWSConnLease{
				pool:      p,
				accountID: accountID,
				conn:      preferredConn,
				queueWait: queueWait,
				connPick:  connPick,
				reused:    true,
			}
			p.metrics.acquireReuseTotal.Add(1)
			p.ensureTargetIdleAsync(accountID)
			return lease, nil
		}

		if preferredConnID != "" {
			if conn, ok := ap.conns[preferredConnID]; ok && conn.tryAcquire() {
				connPick := time.Since(pickStartedAt)
				p.recordConnPickDuration(connPick)
				ap.mu.Unlock()
				closeOpenAIWSConns(evicted)
				if p.shouldHealthCheckConn(conn) {
					if err := conn.pingWithTimeout(openAIWSConnHealthCheckTO); err != nil {
						conn.close()
						p.evictConn(accountID, conn.id)
						if retry < 1 {
							return p.acquire(ctx, req, retry+1)
						}
						return nil, err
					}
				}
				lease := &openAIWSConnLease{pool: p, accountID: accountID, conn: conn, connPick: connPick, reused: true}
				p.metrics.acquireReuseTotal.Add(1)
				p.ensureTargetIdleAsync(accountID)
				return lease, nil
			}
		}

		best := p.pickLeastBusyConnLocked(ap, "")
		if best != nil && best.tryAcquire() {
			connPick := time.Since(pickStartedAt)
			p.recordConnPickDuration(connPick)
			ap.mu.Unlock()
			closeOpenAIWSConns(evicted)
			if p.shouldHealthCheckConn(best) {
				if err := best.pingWithTimeout(openAIWSConnHealthCheckTO); err != nil {
					best.close()
					p.evictConn(accountID, best.id)
					if retry < 1 {
						return p.acquire(ctx, req, retry+1)
					}
					return nil, err
				}
			}
			lease := &openAIWSConnLease{pool: p, accountID: accountID, conn: best, connPick: connPick, reused: true}
			p.metrics.acquireReuseTotal.Add(1)
			p.ensureTargetIdleAsync(accountID)
			return lease, nil
		}
		for _, conn := range ap.conns {
			if conn == nil || conn == best {
				continue
			}
			if conn.tryAcquire() {
				connPick := time.Since(pickStartedAt)
				p.recordConnPickDuration(connPick)
				ap.mu.Unlock()
				closeOpenAIWSConns(evicted)
				if p.shouldHealthCheckConn(conn) {
					if err := conn.pingWithTimeout(openAIWSConnHealthCheckTO); err != nil {
						conn.close()
						p.evictConn(accountID, conn.id)
						if retry < 1 {
							return p.acquire(ctx, req, retry+1)
						}
						return nil, err
					}
				}
				lease := &openAIWSConnLease{pool: p, accountID: accountID, conn: conn, connPick: connPick, reused: true}
				p.metrics.acquireReuseTotal.Add(1)
				p.ensureTargetIdleAsync(accountID)
				return lease, nil
			}
		}
	}

	if req.ForceNewConn && len(ap.conns)+ap.creating >= effectiveMaxConns {
		if idle := p.pickOldestIdleConnLocked(ap); idle != nil {
			delete(ap.conns, idle.id)
			evicted = append(evicted, idle)
			p.metrics.scaleDownTotal.Add(1)
		}
	}

	if len(ap.conns)+ap.creating < effectiveMaxConns {
		connPick := time.Since(pickStartedAt)
		p.recordConnPickDuration(connPick)
		ap.creating++
		ap.mu.Unlock()
		closeOpenAIWSConns(evicted)

		conn, dialErr := p.dialConn(ctx, req)

		ap = p.getOrCreateAccountPool(accountID)
		ap.mu.Lock()
		ap.creating--
		if dialErr != nil {
			ap.prewarmFails++
			ap.prewarmFailAt = time.Now()
			ap.mu.Unlock()
			return nil, dialErr
		}
		ap.conns[conn.id] = conn
		ap.prewarmFails = 0
		ap.prewarmFailAt = time.Time{}
		ap.mu.Unlock()
		p.metrics.acquireCreateTotal.Add(1)

		if !conn.tryAcquire() {
			if err := conn.acquire(ctx); err != nil {
				conn.close()
				p.evictConn(accountID, conn.id)
				return nil, err
			}
		}
		lease := &openAIWSConnLease{pool: p, accountID: accountID, conn: conn, connPick: connPick}
		p.ensureTargetIdleAsync(accountID)
		return lease, nil
	}

	if req.ForceNewConn {
		p.recordConnPickDuration(time.Since(pickStartedAt))
		ap.mu.Unlock()
		closeOpenAIWSConns(evicted)
		return nil, errOpenAIWSConnQueueFull
	}

	target := p.pickLeastBusyConnLocked(ap, req.PreferredConnID)
	connPick := time.Since(pickStartedAt)
	p.recordConnPickDuration(connPick)
	if target == nil {
		ap.mu.Unlock()
		closeOpenAIWSConns(evicted)
		return nil, errOpenAIWSConnClosed
	}
	if int(target.waiters.Load()) >= p.queueLimitPerConn() {
		ap.mu.Unlock()
		closeOpenAIWSConns(evicted)
		return nil, errOpenAIWSConnQueueFull
	}
	target.waiters.Add(1)
	ap.mu.Unlock()
	closeOpenAIWSConns(evicted)
	defer target.waiters.Add(-1)
	waitStart := time.Now()
	p.metrics.acquireQueueWaitTotal.Add(1)

	if err := target.acquire(ctx); err != nil {
		if errors.Is(err, errOpenAIWSConnClosed) && retry < 1 {
			return p.acquire(ctx, req, retry+1)
		}
		return nil, err
	}
	if p.shouldHealthCheckConn(target) {
		if err := target.pingWithTimeout(openAIWSConnHealthCheckTO); err != nil {
			target.release()
			target.close()
			p.evictConn(accountID, target.id)
			if retry < 1 {
				return p.acquire(ctx, req, retry+1)
			}
			return nil, err
		}
	}

	queueWait := time.Since(waitStart)
	p.metrics.acquireQueueWaitMs.Add(queueWait.Milliseconds())
	lease := &openAIWSConnLease{pool: p, accountID: accountID, conn: target, queueWait: queueWait, connPick: connPick, reused: true}
	p.metrics.acquireReuseTotal.Add(1)
	p.ensureTargetIdleAsync(accountID)
	return lease, nil
}

func (p *openAIWSConnPool) pickOldestIdleConnLocked(ap *openAIWSAccountPool) *openAIWSConn {
	if ap == nil || len(ap.conns) == 0 {
		return nil
	}
	var oldest *openAIWSConn
	for _, conn := range ap.conns {
		if conn == nil || conn.isLeased() || conn.waiters.Load() > 0 || p.isConnPinnedLocked(ap, conn.id) {
			continue
		}
		if oldest == nil || conn.lastUsedAt().Before(oldest.lastUsedAt()) {
			oldest = conn
		}
	}
	return oldest
}

func (p *openAIWSConnPool) getOrCreateAccountPool(accountID int64) *openAIWSAccountPool {
	if p == nil || accountID <= 0 {
		return nil
	}
	if existing, ok := p.accounts.Load(accountID); ok {
		if ap, typed := existing.(*openAIWSAccountPool); typed && ap != nil {
			return ap
		}
	}
	ap := &openAIWSAccountPool{
		conns:       make(map[string]*openAIWSConn),
		pinnedConns: make(map[string]int),
	}
	actual, _ := p.accounts.LoadOrStore(accountID, ap)
	if typed, ok := actual.(*openAIWSAccountPool); ok && typed != nil {
		return typed
	}
	return ap
}

// ensureAccountPoolLocked 兼容旧调用。
func (p *openAIWSConnPool) ensureAccountPoolLocked(accountID int64) *openAIWSAccountPool {
	return p.getOrCreateAccountPool(accountID)
}

func (p *openAIWSConnPool) getAccountPool(accountID int64) (*openAIWSAccountPool, bool) {
	if p == nil || accountID <= 0 {
		return nil, false
	}
	value, ok := p.accounts.Load(accountID)
	if !ok || value == nil {
		return nil, false
	}
	ap, typed := value.(*openAIWSAccountPool)
	return ap, typed && ap != nil
}

func (p *openAIWSConnPool) isConnPinnedLocked(ap *openAIWSAccountPool, connID string) bool {
	if ap == nil || connID == "" || len(ap.pinnedConns) == 0 {
		return false
	}
	return ap.pinnedConns[connID] > 0
}

func (p *openAIWSConnPool) cleanupAccountLocked(ap *openAIWSAccountPool, now time.Time, maxConns int) []*openAIWSConn {
	if ap == nil {
		return nil
	}
	maxAge := p.maxConnAge()

	evicted := make([]*openAIWSConn, 0)
	for id, conn := range ap.conns {
		if conn == nil {
			delete(ap.conns, id)
			if len(ap.pinnedConns) > 0 {
				delete(ap.pinnedConns, id)
			}
			continue
		}
		select {
		case <-conn.closedCh:
			delete(ap.conns, id)
			if len(ap.pinnedConns) > 0 {
				delete(ap.pinnedConns, id)
			}
			evicted = append(evicted, conn)
			continue
		default:
		}
		if p.isConnPinnedLocked(ap, id) {
			continue
		}
		if maxAge > 0 && !conn.isLeased() && conn.age(now) > maxAge {
			delete(ap.conns, id)
			if len(ap.pinnedConns) > 0 {
				delete(ap.pinnedConns, id)
			}
			evicted = append(evicted, conn)
		}
	}

	if maxConns <= 0 {
		maxConns = p.maxConnsHardCap()
	}
	maxIdle := p.maxIdlePerAccount()
	if maxIdle < 0 || maxIdle > maxConns {
		maxIdle = maxConns
	}
	if maxIdle >= 0 && len(ap.conns) > maxIdle {
		idleConns := make([]*openAIWSConn, 0, len(ap.conns))
		for id, conn := range ap.conns {
			if conn == nil {
				delete(ap.conns, id)
				if len(ap.pinnedConns) > 0 {
					delete(ap.pinnedConns, id)
				}
				continue
			}
			// 有等待者的连接不能在清理阶段被淘汰，否则等待中的 acquire 会收到 closed 错误。
			if conn.isLeased() || conn.waiters.Load() > 0 || p.isConnPinnedLocked(ap, conn.id) {
				continue
			}
			idleConns = append(idleConns, conn)
		}
		sort.SliceStable(idleConns, func(i, j int) bool {
			return idleConns[i].lastUsedAt().Before(idleConns[j].lastUsedAt())
		})
		redundant := len(ap.conns) - maxIdle
		if redundant > len(idleConns) {
			redundant = len(idleConns)
		}
		for i := 0; i < redundant; i++ {
			conn := idleConns[i]
			delete(ap.conns, conn.id)
			if len(ap.pinnedConns) > 0 {
				delete(ap.pinnedConns, conn.id)
			}
			evicted = append(evicted, conn)
		}
		if redundant > 0 {
			p.metrics.scaleDownTotal.Add(int64(redundant))
		}
	}

	return evicted
}

func (p *openAIWSConnPool) pickLeastBusyConnLocked(ap *openAIWSAccountPool, preferredConnID string) *openAIWSConn {
	if ap == nil || len(ap.conns) == 0 {
		return nil
	}
	preferredConnID = stringsTrim(preferredConnID)
	if preferredConnID != "" {
		if conn, ok := ap.conns[preferredConnID]; ok {
			return conn
		}
	}
	var best *openAIWSConn
	var bestWaiters int32
	var bestLastUsed time.Time
	for _, conn := range ap.conns {
		if conn == nil {
			continue
		}
		waiters := conn.waiters.Load()
		lastUsed := conn.lastUsedAt()
		if best == nil ||
			waiters < bestWaiters ||
			(waiters == bestWaiters && lastUsed.Before(bestLastUsed)) {
			best = conn
			bestWaiters = waiters
			bestLastUsed = lastUsed
		}
	}
	return best
}

func accountPoolLoadLocked(ap *openAIWSAccountPool) (inflight int, waiters int) {
	if ap == nil {
		return 0, 0
	}
	for _, conn := range ap.conns {
		if conn == nil {
			continue
		}
		if conn.isLeased() {
			inflight++
		}
		waiters += int(conn.waiters.Load())
	}
	return inflight, waiters
}

// AccountPoolLoad 返回指定账号连接池的并发与排队快照。
func (p *openAIWSConnPool) AccountPoolLoad(accountID int64) (inflight int, waiters int, conns int) {
	if p == nil || accountID <= 0 {
		return 0, 0, 0
	}
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return 0, 0, 0
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	inflight, waiters = accountPoolLoadLocked(ap)
	return inflight, waiters, len(ap.conns)
}

func (p *openAIWSConnPool) ensureTargetIdleAsync(accountID int64) {
	if p == nil || accountID <= 0 {
		return
	}

	var req openAIWSAcquireRequest
	need := 0
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	if ap.lastAcquire == nil {
		return
	}
	if ap.prewarmActive {
		return
	}
	now := time.Now()
	if !ap.prewarmUntil.IsZero() && now.Before(ap.prewarmUntil) {
		return
	}
	if p.shouldSuppressPrewarmLocked(ap, now) {
		return
	}
	effectiveMaxConns := p.maxConnsHardCap()
	if ap.lastAcquire != nil && ap.lastAcquire.Account != nil {
		effectiveMaxConns = p.effectiveMaxConnsByAccount(ap.lastAcquire.Account)
	}
	target := p.targetConnCountLocked(ap, effectiveMaxConns)
	current := len(ap.conns) + ap.creating
	if current >= target {
		return
	}
	need = target - current
	if need <= 0 {
		return
	}
	req = cloneOpenAIWSAcquireRequest(*ap.lastAcquire)
	ap.prewarmActive = true
	if cooldown := p.prewarmCooldown(); cooldown > 0 {
		ap.prewarmUntil = now.Add(cooldown)
	}
	ap.creating += need
	p.metrics.scaleUpTotal.Add(int64(need))

	go p.prewarmConns(accountID, req, need)
}

func (p *openAIWSConnPool) prewarmConns(accountID int64, req openAIWSAcquireRequest, total int) {
	defer func() {
		if ap, ok := p.getAccountPool(accountID); ok && ap != nil {
			ap.mu.Lock()
			ap.prewarmActive = false
			ap.mu.Unlock()
		}
	}()

	for i := 0; i < total; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), p.dialTimeout()+openAIWSConnPrewarmExtraDelay)
		conn, err := p.dialConn(ctx, req)
		cancel()

		ap, ok := p.getAccountPool(accountID)
		if !ok || ap == nil {
			if conn != nil {
				conn.close()
			}
			return
		}
		ap.mu.Lock()
		if ap.creating > 0 {
			ap.creating--
		}
		if err != nil {
			ap.prewarmFails++
			ap.prewarmFailAt = time.Now()
			ap.mu.Unlock()
			continue
		}
		if len(ap.conns) >= p.effectiveMaxConnsByAccount(req.Account) {
			ap.mu.Unlock()
			conn.close()
			continue
		}
		ap.conns[conn.id] = conn
		ap.prewarmFails = 0
		ap.prewarmFailAt = time.Time{}
		ap.mu.Unlock()
	}
}

func (p *openAIWSConnPool) evictConn(accountID int64, connID string) {
	if p == nil || accountID <= 0 || stringsTrim(connID) == "" {
		return
	}
	var conn *openAIWSConn
	ap, ok := p.getAccountPool(accountID)
	if ok && ap != nil {
		ap.mu.Lock()
		if c, exists := ap.conns[connID]; exists {
			conn = c
			delete(ap.conns, connID)
			if len(ap.pinnedConns) > 0 {
				delete(ap.pinnedConns, connID)
			}
		}
		ap.mu.Unlock()
	}
	if conn != nil {
		conn.close()
	}
}

func (p *openAIWSConnPool) PinConn(accountID int64, connID string) bool {
	if p == nil || accountID <= 0 {
		return false
	}
	connID = stringsTrim(connID)
	if connID == "" {
		return false
	}
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return false
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	if _, exists := ap.conns[connID]; !exists {
		return false
	}
	if ap.pinnedConns == nil {
		ap.pinnedConns = make(map[string]int)
	}
	ap.pinnedConns[connID]++
	return true
}

func (p *openAIWSConnPool) UnpinConn(accountID int64, connID string) {
	if p == nil || accountID <= 0 {
		return
	}
	connID = stringsTrim(connID)
	if connID == "" {
		return
	}
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	if len(ap.pinnedConns) == 0 {
		return
	}
	count := ap.pinnedConns[connID]
	if count <= 1 {
		delete(ap.pinnedConns, connID)
		return
	}
	ap.pinnedConns[connID] = count - 1
}

func (p *openAIWSConnPool) dialConn(ctx context.Context, req openAIWSAcquireRequest) (*openAIWSConn, error) {
	if p == nil || p.clientDialer == nil {
		return nil, errors.New("openai ws client dialer is nil")
	}
	conn, status, handshakeHeaders, err := p.clientDialer.Dial(ctx, req.WSURL, req.Headers, req.ProxyURL)
	if err != nil {
		return nil, &openAIWSDialError{
			StatusCode:      status,
			ResponseHeaders: cloneHeader(handshakeHeaders),
			Err:             err,
		}
	}
	if conn == nil {
		return nil, &openAIWSDialError{
			StatusCode:      status,
			ResponseHeaders: cloneHeader(handshakeHeaders),
			Err:             errors.New("openai ws dialer returned nil connection"),
		}
	}
	id := p.nextConnID(req.Account.ID)
	return newOpenAIWSConn(id, req.Account.ID, conn, handshakeHeaders), nil
}

func (p *openAIWSConnPool) nextConnID(accountID int64) string {
	seq := p.seq.Add(1)
	buf := make([]byte, 0, 32)
	buf = append(buf, "oa_ws_"...)
	buf = strconv.AppendInt(buf, accountID, 10)
	buf = append(buf, '_')
	buf = strconv.AppendUint(buf, seq, 10)
	return string(buf)
}

func (p *openAIWSConnPool) shouldHealthCheckConn(conn *openAIWSConn) bool {
	if conn == nil {
		return false
	}
	return conn.idleDuration(time.Now()) >= openAIWSConnHealthCheckIdle
}
