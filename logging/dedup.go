package logging

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

// dedupMaxKeys bounds the suppression map. A crawler generates unbounded
// distinct URLs, so the key is built only from the low-cardinality attributes
// below; the cap is a backstop against that assumption being broken later.
const dedupMaxKeys = 1024

// dedupKeyAttrs are the attribute keys folded into the dedup key, in addition
// to level and message. Anything not listed here (url, err, status) is carried
// on the line but does not create a new key.
var dedupKeyAttrs = []string{"reason", "domain"}

// DedupHandler collapses repeated Warn/Error records within a time window
// down to a single line per key. The first record in each window passes
// through carrying a "suppressed" count of how many duplicates were dropped
// during the previous window.
//
// Records below Warn are never deduplicated — summary and debug output must
// pass through untouched.
type DedupHandler struct {
	inner slog.Handler
	state *dedupState
	attrs []slog.Attr
}

type dedupState struct {
	window  time.Duration
	maxKeys int

	mu   sync.Mutex
	seen map[string]*dedupEntry
}

type dedupEntry struct {
	windowStart time.Time
	suppressed  int64
}

func NewDedupHandler(inner slog.Handler, window time.Duration, maxKeys int) *DedupHandler {
	if window <= 0 {
		window = time.Minute
	}
	if maxKeys <= 0 {
		maxKeys = dedupMaxKeys
	}
	return &DedupHandler{
		inner: inner,
		state: &dedupState{
			window:  window,
			maxKeys: maxKeys,
			seen:    make(map[string]*dedupEntry),
		},
	}
}

func (h *DedupHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

// WithAttrs returns a handler sharing the same suppression state, so
// dedup works across derived loggers.
func (h *DedupHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	merged = append(merged, h.attrs...)
	merged = append(merged, attrs...)
	return &DedupHandler{inner: h.inner.WithAttrs(attrs), state: h.state, attrs: merged}
}

func (h *DedupHandler) WithGroup(name string) slog.Handler {
	return &DedupHandler{inner: h.inner.WithGroup(name), state: h.state, attrs: h.attrs}
}

func (h *DedupHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < slog.LevelWarn {
		return h.inner.Handle(ctx, r)
	}

	key := h.key(r)
	now := r.Time
	if now.IsZero() {
		now = time.Now()
	}

	suppressed, pass := h.state.admit(key, now)
	if !pass {
		return nil
	}
	if suppressed > 0 {
		r = r.Clone()
		r.AddAttrs(slog.Int64("suppressed", suppressed))
	}

	return h.inner.Handle(ctx, r)
}

// key builds the suppression key from level, message and the low-cardinality
// attributes in dedupKeyAttrs. \x00 separates parts so distinct fields cannot
// collide by concatenation.
func (h *DedupHandler) key(r slog.Record) string {
	var b strings.Builder
	b.WriteString(r.Level.String())
	b.WriteByte(0)
	b.WriteString(r.Message)

	want := make(map[string]string, len(dedupKeyAttrs))
	for _, a := range h.attrs {
		if isDedupKeyAttr(a.Key) {
			want[a.Key] = a.Value.String()
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		if isDedupKeyAttr(a.Key) {
			want[a.Key] = a.Value.String()
		}
		return true
	})

	for _, k := range dedupKeyAttrs {
		b.WriteByte(0)
		b.WriteString(want[k])
	}

	return b.String()
}

func isDedupKeyAttr(k string) bool {
	return slices.Contains(dedupKeyAttrs, k)
}

// admit reports whether a record with this key should be emitted, and how many
// duplicates were suppressed since the last emission for that key.
func (s *dedupState) admit(key string, now time.Time) (suppressed int64, pass bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e, ok := s.seen[key]; ok {
		if now.Sub(e.windowStart) < s.window {
			e.suppressed++
			return 0, false
		}
		suppressed = e.suppressed
		e.windowStart = now
		e.suppressed = 0
		return suppressed, true
	}

	s.evictLocked(now)
	s.seen[key] = &dedupEntry{windowStart: now}

	return 0, true
}

// evictLocked keeps the map bounded: expired entries first, then arbitrary
// entries if the cap is still exceeded. Dropping an entry only costs one
// duplicate line, never correctness.
func (s *dedupState) evictLocked(now time.Time) {
	if len(s.seen) < s.maxKeys {
		return
	}
	for k, e := range s.seen {
		if now.Sub(e.windowStart) >= s.window {
			delete(s.seen, k)
		}
	}
	for k := range s.seen {
		if len(s.seen) < s.maxKeys {
			break
		}
		delete(s.seen, k)
	}
}
