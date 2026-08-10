package logging

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Crawl is the process-wide crawl counter set. The crawl loop and the Colly
// callbacks both write to it; Colly runs callbacks on arbitrary goroutines,
// so every field here is concurrency-safe.
var Crawl = NewStats()

// Stats accumulates crawl outcomes between summary emissions.
//
// Fixed counters use atomic.Int64 so the hot path is lock-free. Failures are
// keyed by a low-cardinality reason string and guarded by a mutex; the whole
// map is swapped on snapshot so the reset is atomic with the read.
type Stats struct {
	found   atomic.Int64
	scraped atomic.Int64
	empty   atomic.Int64
	skipped atomic.Int64

	// backlog and deferred are gauges, not counters: they are not reset on
	// snapshot. Together they distinguish an idle crawler from a wedged one —
	// backlog is the true number of articles awaiting a body, deferred is how
	// many of those are currently waiting out a retry backoff.
	backlog  atomic.Int64
	deferred atomic.Int64

	mu       sync.Mutex
	failures map[string]int64
}

func NewStats() *Stats {
	return &Stats{failures: make(map[string]int64)}
}

func (s *Stats) IncFound()   { s.found.Add(1) }
func (s *Stats) IncScraped() { s.scraped.Add(1) }
func (s *Stats) IncEmpty()   { s.empty.Add(1) }
func (s *Stats) IncSkipped() { s.skipped.Add(1) }

// SetBacklog records the total number of articles still awaiting a body.
// This is a count over the whole table, not the size of the current batch —
// a batch is capped, so reporting its length would just echo the cap back and
// hide a growing queue.
func (s *Stats) SetBacklog(n int) { s.backlog.Store(int64(n)) }

// SetDeferred records how many articles are waiting out a retry backoff.
func (s *Stats) SetDeferred(n int) { s.deferred.Store(int64(n)) }

// IncFailure records one failure under a reason. reason must come from a
// closed set (see scraper.classifyError) — never a raw error string or a URL,
// or this map grows without bound.
func (s *Stats) IncFailure(reason string) {
	s.mu.Lock()
	s.failures[reason]++
	s.mu.Unlock()
}

// snapshot drains every counter and returns the values for the elapsed
// interval. The gap between the atomic swaps and the map swap means an event
// landing mid-snapshot is counted in the next interval; totals are still
// conserved, which is all a summary line needs.
func (s *Stats) snapshot() (found, scraped, empty, skipped, backlog, deferred int64, failures map[string]int64) {
	found = s.found.Swap(0)
	scraped = s.scraped.Swap(0)
	empty = s.empty.Swap(0)
	skipped = s.skipped.Swap(0)
	backlog = s.backlog.Load()
	deferred = s.deferred.Load()

	s.mu.Lock()
	failures = s.failures
	s.failures = make(map[string]int64, len(failures))
	s.mu.Unlock()

	return found, scraped, empty, skipped, backlog, deferred, failures
}

// Run emits a summary line every interval until ctx is cancelled, then emits
// one final line covering the partial interval.
//
// The final emit is deferred so it runs on any exit path. Callers must wait
// for Run to return before the process exits, or the final line is lost —
// which is exactly the interval containing whatever stopped the crawler.
func (s *Stats) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	defer s.emit(interval, "final")

	for {
		select {
		case <-t.C:
			s.emit(interval, "interval")
		case <-ctx.Done():
			return
		}
	}
}

// emit writes one summary line. It emits even when every counter is zero: a
// missing line is ambiguous between "idle" and "dead", and the backlog gauge
// is what tells those apart.
func (s *Stats) emit(interval time.Duration, kind string) {
	found, scraped, empty, skipped, backlog, deferred, failures := s.snapshot()

	var failed int64
	for _, n := range failures {
		failed += n
	}

	attrs := []any{
		"kind", kind,
		"interval_s", int(interval.Seconds()),
		"backlog", backlog,
		"deferred", deferred,
		"items_found", found,
		"items_scraped", scraped,
		"items_empty", empty,
		"items_skipped", skipped,
		"items_failed", failed,
	}

	// Sort so the field order is stable across lines.
	reasons := make([]string, 0, len(failures))
	for r := range failures {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons)
	for _, r := range reasons {
		attrs = append(attrs, "fail_"+r, failures[r])
	}

	slog.Info("crawl summary", attrs...)
}
