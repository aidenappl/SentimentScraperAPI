# SentimentScraperAPI

News ingestion, article crawling and sentiment scoring for Sentiment Scraper.

> **Sentiment Scraper** · Go API · `sentimentscraper` (Lattice stack 11, port 8001)

---

## Overview

SentimentScraperAPI polls a financial-news brief feed, crawls each linked
article for its full text, scores it for sentiment, and serves the results over
a small read-only HTTP API.

Article extraction is domain-aware: known outlets get purpose-built parsers,
and everything else falls through to a generic extractor that reads schema.org
JSON-LD or picks the densest block of paragraph text on the page — so no URL is
ever left unparsed. Outlets behind hard paywalls are ingested for their
headline and symbols but never crawled.

## Role in the Sentiment Scraper ecosystem

This is the only backend. `sentiment-scraper-web` renders what it serves. The
brief feed is external (`static.newsfilter.io`); liveness is reported to
healthchecks.io.

## Tech stack

Go 1.23 · PostgreSQL (`lib/pq`) · gorilla/mux · squirrel · gocolly/colly ·
goquery · log/slog · VADER + a custom classifier for sentiment

## Getting started

### Prerequisites

- Go 1.23+
- A reachable PostgreSQL instance
- A `.env` file with `CORE_DB`, `OPENAI_KEY` and `PORT`

### Setup

```bash
git clone git@github.com:aidenappl/SentimentScraperAPI.git
cd SentimentScraperAPI
dev tidy
dev
```

Tests need neither network nor database — the crawler tests run against local
`httptest` servers.

## Development

| Command | What it does |
|---|---|
| `dev` | Run the service |
| `dev build` | Build to `bin/app` |
| `dev test` | Run the test suite |
| `dev fmt` | Format |
| `dev vet` | Vet |
| `dev check` | Format + vet |
| `dev tidy` | `go mod tidy` |

### Configuration

| Variable | Default | Purpose |
|---|---|---|
| `CORE_DB` | — (required) | Postgres DSN |
| `OPENAI_KEY` | — (required) | Currently unused |
| `PORT` | `8000` | Listen port |
| `LOG_LEVEL` | `INFO` | `DEBUG` restores per-item crawl lines |
| `LOG_SUMMARY_INTERVAL` | `5m` | Crawl summary cadence and error dedup window |
| `CRAWL_INTERVAL` | `1m` | Time between crawl cycles |
| `CRAWL_BATCH_LIMIT` | `50` | Max articles attempted per cycle |
| `CRAWL_MAX_ATTEMPTS` | `5` | Consecutive failures before the retry delay stops doubling |
| `CRAWL_RETRY_BACKOFF` | `15m` | First retry delay after a failure; doubles each time |
| `CRAWL_RETRY_BACKOFF_MAX` | `6h` | Ceiling on that delay |
| `CRAWL_BLOCKED_DOMAINS` | built-in list | Comma-separated outlets to ingest but never crawl; `none` disables blocking |

### Logging

Output is structured JSON on stderr. The crawl loop does not log per article;
it emits one summary line per `LOG_SUMMARY_INTERVAL`:

```json
{"msg":"crawl summary","kind":"interval","interval_s":300,"backlog":1284,
 "deferred":37,"items_found":40,"items_scraped":31,"items_failed":5,
 "fail_forbidden":4}
```

`backlog` is the true number of articles awaiting a body; `deferred` is how
many of those are waiting out a retry backoff. Failed articles retry with
exponential backoff rather than being abandoned.

Errors are logged immediately and deduplicated by `(reason, domain)` within the
interval, so one blocking outlet produces one line rather than thousands. Set
`LOG_LEVEL=DEBUG` to get per-article detail back.

## Project structure

```
main.go        Wiring and graceful shutdown
background/    Feed polling, crawl loop, health pings
db/            Connection and Queryable interface
env/           Environment configuration
logging/       slog setup, crawl counters, error dedup
middleware/    Request logging
query/         Squirrel-built queries
responder/     JSON response envelopes
retry/         Exponential-backoff scheduler
routers/       HTTP handlers
scraper/       Feed client, crawler, domain + generic parsers
sentiment/     Sentiment worker and queue
state/         In-memory news cache
structs/       Domain types
tools/         Small helpers
```

## Deployment

CI builds and pushes `registry.appleby.cloud/sentimentscraper:latest`, then
triggers the Lattice deploy. The container healthcheck hits `/health`.

## Contributing & further reading

See [AGENTS.md](AGENTS.md) for architecture, conventions, operational runbook,
and the rules that keep log volume under control.
