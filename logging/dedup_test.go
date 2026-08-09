package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func newTestDedupLogger(window time.Duration, maxKeys int) (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	return slog.New(NewDedupHandler(inner, window, maxKeys)), buf
}

func countLines(buf *bytes.Buffer) int {
	trimmed := strings.TrimSpace(buf.String())
	if trimmed == "" {
		return 0
	}

	return len(strings.Split(trimmed, "\n"))
}

func TestDedupSuppressesRepeatsWithinWindow(t *testing.T) {
	log, buf := newTestDedupLogger(time.Hour, 16)

	for range 50 {
		log.Error("scrape request failed", "reason", "forbidden", "domain", "mckinsey.com", "url", "https://mckinsey.com/a")
	}

	if got := countLines(buf); got != 1 {
		t.Fatalf("got %d lines, want 1 — repeats inside the window must be suppressed", got)
	}
}

func TestDedupSeparatesDistinctKeys(t *testing.T) {
	log, buf := newTestDedupLogger(time.Hour, 16)

	log.Error("scrape request failed", "reason", "forbidden", "domain", "mckinsey.com")
	log.Error("scrape request failed", "reason", "forbidden", "domain", "reuters.com")
	log.Error("scrape request failed", "reason", "timeout", "domain", "mckinsey.com")

	// Same message, but three distinct (reason, domain) pairs.
	if got := countLines(buf); got != 3 {
		t.Fatalf("got %d lines, want 3", got)
	}
}

func TestDedupIgnoresHighCardinalityAttrs(t *testing.T) {
	log, buf := newTestDedupLogger(time.Hour, 16)

	// The URL differs every time; it must not create a new key, or the
	// suppression map would grow without bound.
	for i := range 20 {
		log.Error("scrape request failed",
			"reason", "forbidden",
			"domain", "mckinsey.com",
			"url", "https://mckinsey.com/article-"+string(rune('a'+i)),
		)
	}

	if got := countLines(buf); got != 1 {
		t.Fatalf("got %d lines, want 1 — url must not be part of the dedup key", got)
	}
}

func TestDedupEmitsAgainAfterWindow(t *testing.T) {
	log, buf := newTestDedupLogger(10*time.Millisecond, 16)

	log.Error("scrape request failed", "reason", "forbidden", "domain", "mckinsey.com")
	log.Error("scrape request failed", "reason", "forbidden", "domain", "mckinsey.com")
	time.Sleep(20 * time.Millisecond)
	log.Error("scrape request failed", "reason", "forbidden", "domain", "mckinsey.com")

	if got := countLines(buf); got != 2 {
		t.Fatalf("got %d lines, want 2 — a new window must emit again", got)
	}
	if !strings.Contains(buf.String(), `"suppressed":1`) {
		t.Fatalf("second line should report the suppressed count, got: %s", buf.String())
	}
}

func TestDedupPassesThroughBelowWarn(t *testing.T) {
	log, buf := newTestDedupLogger(time.Hour, 16)

	for range 5 {
		log.Info("crawl summary", "kind", "interval")
		log.Debug("crawling news item", "news_id", 1)
	}

	// Summaries and debug lines must never be deduplicated.
	if got := countLines(buf); got != 10 {
		t.Fatalf("got %d lines, want 10", got)
	}
}

func TestDedupBoundsKeyCount(t *testing.T) {
	const maxKeys = 8
	h := NewDedupHandler(slog.NewJSONHandler(&bytes.Buffer{}, nil), time.Hour, maxKeys)
	log := slog.New(h)

	// Far more distinct keys than the cap allows.
	for i := range 200 {
		log.Error("scrape request failed", "reason", "forbidden", "domain", string(rune('a'+i%200))+".example.com")
	}

	h.state.mu.Lock()
	size := len(h.state.seen)
	h.state.mu.Unlock()

	if size > maxKeys {
		t.Fatalf("suppression map grew to %d entries, want at most %d", size, maxKeys)
	}
}

func TestDedupWithAttrsSharesState(t *testing.T) {
	log, buf := newTestDedupLogger(time.Hour, 16)

	// Attributes carried on a derived logger must still key the dedup.
	derived := log.With("reason", "forbidden", "domain", "mckinsey.com")
	derived.Error("scrape request failed")
	derived.Error("scrape request failed")

	if got := countLines(buf); got != 1 {
		t.Fatalf("got %d lines, want 1 — WithAttrs must share suppression state", got)
	}
}

func TestDedupHandlerEnabledDelegates(t *testing.T) {
	inner := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := NewDedupHandler(inner, time.Hour, 16)

	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("Debug should be disabled when the inner handler is at Warn")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("Error should be enabled")
	}
}
