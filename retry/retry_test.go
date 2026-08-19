package retry

import (
	"testing"
	"time"
)

var base = time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

func TestRetryReadyByDefault(t *testing.T) {
	tr := New(time.Minute, time.Hour, 5)

	if !tr.Ready(1, base) {
		t.Fatal("an article with no history must be eligible")
	}
}

func TestRetryDefersAfterFailure(t *testing.T) {
	tr := New(time.Minute, time.Hour, 5)

	tr.Fail(1, base)

	if tr.Ready(1, base.Add(30*time.Second)) {
		t.Fatal("article should still be in backoff")
	}
	if !tr.Ready(1, base.Add(time.Minute)) {
		t.Fatal("article should be eligible once the delay has elapsed")
	}
}

func TestRetryBackoffDoublesAndCaps(t *testing.T) {
	tr := New(time.Minute, 8*time.Minute, 10)

	tests := []struct {
		failures int
		want     time.Duration
	}{
		{1, time.Minute},
		{2, 2 * time.Minute},
		{3, 4 * time.Minute},
		{4, 8 * time.Minute},
		{5, 8 * time.Minute},  // capped
		{50, 8 * time.Minute}, // still capped, no overflow
	}

	for _, tt := range tests {
		if got := tr.delayFor(tt.failures); got != tt.want {
			t.Errorf("delayFor(%d) = %v, want %v", tt.failures, got, tt.want)
		}
	}
}

// The whole point of backoff over a permanent skip: an article that keeps
// failing must still come back around, or the crawler eventually has nothing
// left it is willing to attempt.
func TestRetryNeverAbandonsPermanently(t *testing.T) {
	tr := New(time.Minute, 10*time.Minute, 3)

	now := base
	for range 20 {
		tr.Fail(1, now)
		now = now.Add(24 * time.Hour)
	}

	if !tr.Ready(1, now) {
		t.Fatal("article must become eligible again after the backoff elapses")
	}
}

func TestRetrySucceedClearsHistory(t *testing.T) {
	tr := New(time.Minute, time.Hour, 5)

	tr.Fail(1, base)
	tr.Fail(1, base)
	tr.Succeed(1)

	if !tr.Ready(1, base) {
		t.Fatal("a successful scrape must clear the backoff")
	}

	// The next failure should start from the shortest delay again.
	tr.Fail(1, base)
	if !tr.Ready(1, base.Add(time.Minute)) {
		t.Fatal("failure count should have reset on success")
	}
}

func TestRetryDeferredListsOnlyWaitingIDs(t *testing.T) {
	tr := New(time.Minute, time.Hour, 5)

	tr.Fail(1, base)
	tr.Fail(2, base)
	tr.Fail(3, base.Add(-time.Hour)) // long since elapsed

	deferred := tr.Deferred(base.Add(30 * time.Second))

	if len(deferred) != 2 {
		t.Fatalf("got %d deferred IDs, want 2: %v", len(deferred), deferred)
	}
	for _, id := range deferred {
		if id == 3 {
			t.Fatal("an article whose backoff has elapsed must not be excluded")
		}
	}
}

func TestRetryDeferredIsBounded(t *testing.T) {
	tr := New(time.Hour, time.Hour, 5)

	for id := range MaxDeferredIDs * 2 {
		tr.Fail(id, base)
	}

	if got := len(tr.Deferred(base)); got > MaxDeferredIDs {
		t.Fatalf("got %d deferred IDs, want at most %d — the SQL exclusion list must stay bounded", got, MaxDeferredIDs)
	}
}

func TestMarkDeadRemovesFromRotation(t *testing.T) {
	tr := New(time.Minute, time.Hour, 5)

	tr.Fail(1, base)
	tr.MarkDead(1)

	if got := tr.Dead(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("got dead=%v, want [1]", got)
	}

	// A dead item must not also appear as merely deferred, or it would be
	// double-counted against the backlog.
	if got := tr.Deferred(base); len(got) != 0 {
		t.Fatalf("got deferred=%v, want empty once marked dead", got)
	}
}

func TestSucceedRevivesDeadItem(t *testing.T) {
	tr := New(time.Minute, time.Hour, 5)

	tr.MarkDead(1)
	tr.Succeed(1)

	if got := tr.Dead(); len(got) != 0 {
		t.Fatalf("got dead=%v, want empty after a success", got)
	}
}

func TestRetryTrackerDefaults(t *testing.T) {
	// Nonsense configuration must not produce a zero or negative delay, which
	// would make every failure retry immediately and spin the crawler.
	tr := New(0, 0, 0)

	if got := tr.delayFor(1); got <= 0 {
		t.Fatalf("got delay %v, want a positive fallback", got)
	}
}
