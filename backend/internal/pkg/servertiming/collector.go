package servertiming

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const HeaderName = "Server-Timing"

type contextKey struct{}

type Collector struct {
	start   time.Time
	mu      sync.Mutex
	metrics map[string]metric
}

type metric struct {
	duration time.Duration
	count    int
}

func New(start time.Time) *Collector {
	if start.IsZero() {
		start = time.Now()
	}
	return &Collector{
		start:   start,
		metrics: make(map[string]metric),
	}
}

func WithCollector(ctx context.Context, c *Collector) context.Context {
	if ctx == nil || c == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, c)
}

func FromContext(ctx context.Context) (*Collector, bool) {
	if ctx == nil {
		return nil, false
	}
	c, ok := ctx.Value(contextKey{}).(*Collector)
	return c, ok && c != nil
}

func Active(ctx context.Context) bool {
	_, ok := FromContext(ctx)
	return ok
}

func Record(ctx context.Context, name string, duration time.Duration) {
	c, ok := FromContext(ctx)
	if !ok || duration < 0 {
		return
	}
	c.add(name, duration, 1)
}

func Observe(ctx context.Context, name string) func() {
	start := time.Now()
	return func() {
		Record(ctx, name, time.Since(start))
	}
}

func HeaderValue(ctx context.Context, now time.Time) string {
	c, ok := FromContext(ctx)
	if !ok {
		return ""
	}
	return c.headerValue(now)
}

func WrapRoundTripper(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return roundTripper{next: next}
}

func (c *Collector) add(name string, duration time.Duration, count int) {
	name = normalizeName(name)
	if name == "" {
		return
	}
	if count <= 0 {
		count = 1
	}
	c.mu.Lock()
	current := c.metrics[name]
	current.duration += duration
	current.count += count
	c.metrics[name] = current
	c.mu.Unlock()
}

func (c *Collector) headerValue(now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	c.mu.Lock()
	snapshot := make(map[string]metric, len(c.metrics)+1)
	for name, item := range c.metrics {
		snapshot[name] = item
	}
	c.mu.Unlock()
	if elapsed := now.Sub(c.start); elapsed >= 0 {
		snapshot["app"] = metric{duration: elapsed, count: 1}
	}

	names := make([]string, 0, len(snapshot))
	for name := range snapshot {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		item := snapshot[name]
		if item.duration < 0 {
			continue
		}
		part := fmt.Sprintf("%s;dur=%.1f", name, durationMillis(item.duration))
		if item.count > 1 {
			part += fmt.Sprintf(`;desc="%d"`, item.count)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

func normalizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func durationMillis(d time.Duration) float64 {
	ms := float64(d) / float64(time.Millisecond)
	return math.Round(ms*10) / 10
}
