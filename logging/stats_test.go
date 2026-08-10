package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// captureLogs redirects the default logger to a buffer for the duration of a
// test and returns the buffer.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return buf
}

func TestStatsSnapshotDrainsCounters(t *testing.T) {
	s := NewStats()

	s.IncFound()
	s.IncFound()
	s.IncScraped()
	s.IncEmpty()
	s.IncSkipped()
	s.IncFailure("forbidden")
	s.IncFailure("forbidden")
	s.IncFailure("timeout")
	s.SetBacklog(7)

	found, scraped, empty, skipped, backlog, _, failures := s.snapshot()

	if found != 2 || scraped != 1 || empty != 1 || skipped != 1 {
		t.Fatalf("got found=%d scraped=%d empty=%d skipped=%d, want 2/1/1/1", found, scraped, empty, skipped)
	}
	if backlog != 7 {
		t.Fatalf("got backlog=%d, want 7", backlog)
	}
	if failures["forbidden"] != 2 || failures["timeout"] != 1 {
		t.Fatalf("got failures=%v, want forbidden=2 timeout=1", failures)
	}

	// A second snapshot must come back empty; counters reset on read.
	found, scraped, empty, skipped, backlog, _, failures = s.snapshot()
	if found != 0 || scraped != 0 || empty != 0 || skipped != 0 || len(failures) != 0 {
		t.Fatalf("counters not reset: found=%d scraped=%d empty=%d skipped=%d failures=%v",
			found, scraped, empty, skipped, failures)
	}

	// The backlog is a gauge and must survive the snapshot.
	if backlog != 7 {
		t.Fatalf("got backlog=%d after reset, want it to persist as 7", backlog)
	}
}

func TestStatsEmitIncludesFailureBreakdown(t *testing.T) {
	buf := captureLogs(t)

	s := NewStats()
	s.IncFound()
	s.IncFailure("forbidden")
	s.emit(time.Minute, "interval")

	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
		t.Fatalf("summary is not valid JSON: %v", err)
	}

	if line["msg"] != "crawl summary" {
		t.Fatalf("got msg=%v, want 'crawl summary'", line["msg"])
	}
	if line["kind"] != "interval" {
		t.Fatalf("got kind=%v, want 'interval'", line["kind"])
	}
	if line["items_failed"] != float64(1) {
		t.Fatalf("got items_failed=%v, want 1", line["items_failed"])
	}
	if line["fail_forbidden"] != float64(1) {
		t.Fatalf("got fail_forbidden=%v, want 1", line["fail_forbidden"])
	}
}

func TestStatsEmitsWhenIdle(t *testing.T) {
	buf := captureLogs(t)

	// An all-zero interval must still produce a line: a missing line cannot be
	// distinguished from a dead process.
	NewStats().emit(time.Minute, "interval")

	if !strings.Contains(buf.String(), "crawl summary") {
		t.Fatal("expected a summary line for an idle interval, got none")
	}
}

func TestStatsRunEmitsFinalOnCancel(t *testing.T) {
	buf := captureLogs(t)

	ctx, cancel := context.WithCancel(context.Background())

	s := NewStats()
	s.IncScraped()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// A long interval guarantees the only line comes from the final flush.
		s.Run(ctx, time.Hour)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}

	out := buf.String()
	if !strings.Contains(out, `"kind":"final"`) {
		t.Fatalf("expected a final summary line, got: %s", out)
	}
	if !strings.Contains(out, `"items_scraped":1`) {
		t.Fatalf("final line lost the partial interval's counts, got: %s", out)
	}
}
