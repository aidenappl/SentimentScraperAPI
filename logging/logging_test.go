package logging

import (
	"bytes"
	"log"
	"log/slog"
	"strings"
	"testing"
)

func TestLevelParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{"debug", "DEBUG", slog.LevelDebug},
		{"lowercase", "debug", slog.LevelDebug},
		{"padded", "  WARN  ", slog.LevelWarn},
		{"error", "ERROR", slog.LevelError},
		{"garbage falls back to info", "verbose-please", slog.LevelInfo},
		{"empty keeps info", "", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := LevelVar.Level()
			t.Cleanup(func() { LevelVar.Set(prev) })

			LevelVar.Set(slog.LevelInfo)
			if s := strings.TrimSpace(tt.input); s != "" {
				if err := LevelVar.UnmarshalText([]byte(s)); err != nil {
					LevelVar.Set(slog.LevelInfo)
				}
			}

			if got := LevelVar.Level(); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// The service still has ~60 stdlib log.Print calls in packages this change did
// not touch. They stay quiet only because SetLogLoggerLevel routes them
// through slog at Debug — if that ever stops working, the flood returns.
func TestLegacyLogPackageIsDemotedToDebug(t *testing.T) {
	buf := &bytes.Buffer{}

	prevLogger := slog.Default()
	prevFlags, prevPrefix := log.Flags(), log.Prefix()
	t.Cleanup(func() {
		slog.SetDefault(prevLogger)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
		slog.SetLogLoggerLevel(slog.LevelInfo)
	})

	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.SetLogLoggerLevel(slog.LevelDebug)

	log.Printf("📰 Found news item for crawling: %d", 13105)

	if strings.Contains(buf.String(), "Found news item") {
		t.Fatalf("legacy log output was not demoted below the Info threshold: %s", buf.String())
	}
}

func TestLegacyLogPackageVisibleAtDebug(t *testing.T) {
	buf := &bytes.Buffer{}

	prevLogger := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(prevLogger)
		slog.SetLogLoggerLevel(slog.LevelInfo)
	})

	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	slog.SetLogLoggerLevel(slog.LevelDebug)

	log.Printf("📰 Found news item for crawling: %d", 13105)

	if !strings.Contains(buf.String(), "Found news item") {
		t.Fatalf("legacy log output should reappear at Debug, got: %s", buf.String())
	}
}
