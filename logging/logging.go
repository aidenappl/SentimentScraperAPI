// Package logging provides the service's structured logger, a periodic
// summary emitter for high-volume loops, and a dedup handler that collapses
// repeated errors.
//
// The crawl loop touches every backlogged article once per cycle. Logging each
// item individually floods the container log store, so per-item lines are
// emitted at Debug and rolled up into one summary line per interval. Errors
// stay at Error and are logged immediately, deduplicated by (reason, domain).
package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

// LevelVar backs the global logger's minimum level. It is exported so the
// level can be retuned at runtime without a redeploy.
var LevelVar = new(slog.LevelVar)

// Init configures the global slog logger. level is parsed leniently: an
// unrecognised value falls back to INFO rather than panicking, so a typo in
// LOG_LEVEL can never stop the service from booting.
func Init(level string, dedupWindow time.Duration) {
	if s := strings.TrimSpace(level); s != "" {
		// slog.LevelVar implements encoding.TextUnmarshaler, which accepts
		// "DEBUG"/"INFO"/"WARN"/"ERROR" case-insensitively plus numeric
		// offsets like "INFO+2".
		if err := LevelVar.UnmarshalText([]byte(s)); err != nil {
			LevelVar.Set(slog.LevelInfo)
		}
	}

	debug := LevelVar.Level() <= slog.LevelDebug

	var h slog.Handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: LevelVar,
		// AddSource calls runtime.CallersFrames per record, so only pay for
		// it when someone has actually asked for debug output.
		AddSource: debug,
	})
	h = NewDedupHandler(h, dedupWindow, dedupMaxKeys)

	slog.SetDefault(slog.New(h))

	// Route the ~60 remaining stdlib log.Print calls through slog at Debug so
	// they fall silent at the default level. Callers that must always be seen
	// (startup failures) use slog directly instead of log.Fatal.
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

// Fatal logs at Error and exits non-zero. It replaces log.Fatalf, which would
// otherwise be demoted to Debug by SetLogLoggerLevel and exit silently.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
