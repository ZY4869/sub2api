package timezone

import (
	"sync"
	"time"
)

var (
	nowMu   sync.RWMutex
	nowFunc = time.Now
)

func currentTime() time.Time {
	nowMu.RLock()
	fn := nowFunc
	nowMu.RUnlock()
	return fn()
}

// SetNowForTesting overrides the package clock until the returned restore
// function is called. Tests should always defer or register the restore
// callback with t.Cleanup.
func SetNowForTesting(fn func() time.Time) func() {
	nowMu.Lock()
	previous := nowFunc
	nowFunc = fn
	nowMu.Unlock()

	return func() {
		nowMu.Lock()
		nowFunc = previous
		nowMu.Unlock()
	}
}
