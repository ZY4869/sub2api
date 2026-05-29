package repository

import (
	"sync/atomic"
	"time"
)

func (s *httpUpstreamService) evictIdleLocked(now time.Time) {
	ttl := s.clientIdleTTL()
	if ttl <= 0 {
		return
	}
	cutoff := now.Add(-ttl).UnixNano()
	for key, entry := range s.clients {
		if atomic.LoadInt64(&entry.inFlight) != 0 {
			continue
		}
		if atomic.LoadInt64(&entry.lastUsed) <= cutoff {
			s.removeClientLocked(key, entry)
		}
	}
}

func (s *httpUpstreamService) evictOldestIdleLocked() bool {
	var (
		oldestKey   string
		oldestEntry *upstreamClientEntry
		oldestTime  int64
	)
	for key, entry := range s.clients {
		if atomic.LoadInt64(&entry.inFlight) != 0 {
			continue
		}
		lastUsed := atomic.LoadInt64(&entry.lastUsed)
		if oldestEntry == nil || lastUsed < oldestTime {
			oldestKey = key
			oldestEntry = entry
			oldestTime = lastUsed
		}
	}
	if oldestEntry == nil {
		return false
	}
	s.removeClientLocked(oldestKey, oldestEntry)
	return true
}

func (s *httpUpstreamService) evictOverLimitLocked() bool {
	maxClients := s.maxUpstreamClients()
	if maxClients <= 0 {
		return false
	}
	evicted := false
	for len(s.clients) > maxClients {
		if !s.evictOldestIdleLocked() {
			return evicted
		}
		evicted = true
	}
	return evicted
}
