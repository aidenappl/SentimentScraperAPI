package env

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

var (
	CoreDB    = getEnvOrPanic("CORE_DB")
	OpenAIKey = getEnvOrPanic("OPENAI_KEY")
	Port      = getEnv("PORT", "8000")

	// LogLevel is DEBUG, INFO, WARN or ERROR. Per-item crawl lines are logged
	// at DEBUG, so INFO leaves only summaries, errors and lifecycle events.
	LogLevel = getEnv("LOG_LEVEL", "INFO")

	// LogSummaryInterval is how often the crawl summary line is emitted, and
	// also the window over which repeated errors are deduplicated.
	LogSummaryInterval = getEnvDuration("LOG_SUMMARY_INTERVAL", 5*time.Minute, 10*time.Second)

	// CrawlInterval is how long the crawl loop sleeps between cycles.
	CrawlInterval = getEnvDuration("CRAWL_INTERVAL", time.Minute, 10*time.Second)

	// CrawlBatchLimit caps how many outstanding articles one cycle attempts.
	CrawlBatchLimit = getEnvInt("CRAWL_BATCH_LIMIT", 50)

	// CrawlMaxAttempts bounds the retry backoff exponent: after this many
	// consecutive failures the delay stops doubling and sits at
	// CrawlRetryBackoffMax. Articles are never abandoned outright — a crawler
	// that writes off every article has no work left to do.
	CrawlMaxAttempts = getEnvInt("CRAWL_MAX_ATTEMPTS", 5)

	// CrawlRetryBackoff is the delay before a failed article is retried; it
	// doubles with each consecutive failure.
	CrawlRetryBackoff = getEnvDuration("CRAWL_RETRY_BACKOFF", 15*time.Minute, time.Second)

	// CrawlRetryBackoffMax caps that delay.
	CrawlRetryBackoffMax = getEnvDuration("CRAWL_RETRY_BACKOFF_MAX", 6*time.Hour, time.Second)
)

func getEnv(key string, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}

// getEnvDuration parses a Go duration string ("30s", "5m"). An unparseable or
// too-small value falls back to the default rather than stopping the service.
func getEnvDuration(key string, fallback, min time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	d, err := time.ParseDuration(v)
	if err != nil || d < min {
		return fallback
	}

	return d
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}

func getEnvOrPanic(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("❌ missing required environment variable: '%v'\n", key))
	}
	return value
}
