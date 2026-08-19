// Package retry schedules re-attempts for work that failed.
//
// It deliberately depends on nothing else in the service so its scheduling
// rules can be tested in isolation.
package retry

import (
	"sync"
	"time"
)

// MaxDeferredIDs caps how many IDs are excluded from a crawl query. Beyond
// this the exclusion list stops growing and the oldest-scheduled entries are
// allowed through again; a few wasted attempts beat an unbounded SQL clause.
const MaxDeferredIDs = 2000

// Tracker schedules when a failed article may be attempted again.
//
// Failures back off exponentially instead of being skipped permanently. A
// permanent skip sounds tidy but ends with the crawler doing nothing at all:
// once every article in the batch has been written off, there is no work left
// and no way back without a restart.
type Tracker struct {
	base        time.Duration
	max         time.Duration
	maxAttempts int

	mu      sync.Mutex
	entries map[int]*retryEntry
	dead    map[int]struct{}
}

type retryEntry struct {
	failures    int
	nextAttempt time.Time
}

func New(base, max time.Duration, maxAttempts int) *Tracker {
	if base <= 0 {
		base = 15 * time.Minute
	}
	if max < base {
		max = base
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	return &Tracker{
		base:        base,
		max:         max,
		maxAttempts: maxAttempts,
		entries:     make(map[int]*retryEntry),
		dead:        make(map[int]struct{}),
	}
}

// Ready reports whether an article may be attempted now.
func (t *Tracker) Ready(id int, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.entries[id]

	return !ok || !now.Before(e.nextAttempt)
}

// Fail records a failure and schedules the next attempt.
func (t *Tracker) Fail(id int, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.entries[id]
	if !ok {
		e = &retryEntry{}
		t.entries[id] = e
	}
	e.failures++
	e.nextAttempt = now.Add(t.delayFor(e.failures))
}

// Succeed clears an article's history so a later failure starts from the
// shortest delay again.
func (t *Tracker) Succeed(id int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.entries, id)
	delete(t.dead, id)
}

// MarkDead records that an item can never succeed — a deleted article, say —
// so it is dropped from future work entirely rather than retried on a cycle
// forever. Unlike a backoff this is not time-based, but it is still in-memory:
// a restart gives every item another chance, which is the right behaviour when
// the cause might have been a bad deploy rather than a dead URL.
func (t *Tracker) MarkDead(id int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.entries, id)
	t.dead[id] = struct{}{}
}

// Dead returns the items marked permanently unavailable.
func (t *Tracker) Dead() []int {
	t.mu.Lock()
	defer t.mu.Unlock()

	ids := make([]int, 0, len(t.dead))
	for id := range t.dead {
		ids = append(ids, id)
	}

	return ids
}

// delayFor doubles the base delay per failure, capped both by maxAttempts
// (which bounds the exponent) and by the max delay.
func (t *Tracker) delayFor(failures int) time.Duration {
	if failures > t.maxAttempts {
		failures = t.maxAttempts
	}

	delay := t.base
	for range failures - 1 {
		delay *= 2
		if delay >= t.max {
			return t.max
		}
	}

	return delay
}

// Deferred returns the IDs that are not yet eligible, for exclusion from the
// next crawl query. Without this the batch refills with the same failing rows
// every cycle and never reaches the rest of the backlog.
func (t *Tracker) Deferred(now time.Time) []int {
	t.mu.Lock()
	defer t.mu.Unlock()

	ids := make([]int, 0, len(t.entries))
	for id, e := range t.entries {
		if now.Before(e.nextAttempt) {
			ids = append(ids, id)
		}
		if len(ids) >= MaxDeferredIDs {
			break
		}
	}

	return ids
}

// Pending reports how many articles are currently waiting out a backoff.
func (t *Tracker) Pending(now time.Time) int {
	return len(t.Deferred(now))
}
