# AGENTS.md — SentimentScraperAPI

## 1. What this repo is

SentimentScraperAPI is the Go service behind Sentiment Scraper: it ingests a
financial-news brief feed, crawls the linked articles for their full text,
scores them for sentiment, and serves the result over a small read-only HTTP
API.

It owns the news pipeline end to end — feed polling, article crawling, body
extraction, sentiment scoring, and the `/core/v1` read endpoints.

It does **not** own the frontend (`sentiment-scraper-web`), and it is not a
general-purpose crawler: it only fetches URLs that arrived via the brief feed.

## 2. Stack & dependencies

- **Go 1.23.4**, module `github.com/aidenappl/SentimentScraperAPI`
- **PostgreSQL** via `lib/pq` — note this service is Postgres, not MariaDB;
  all queries use `$1` placeholders (`sq.Dollar`)
- **`gorilla/mux`** router, **`rs/cors`** for CORS
- **`Masterminds/squirrel`** for query building — no ORM
- **`gocolly/colly` v1** for crawling, **`PuerkitoBio/goquery`** for the
  generic article extractor
- **`log/slog`** (stdlib) for structured logging — no logrus, no zap
- Sentiment: **`drankou/go-vader`** and **`cdipaolo/sentiment`**, backed by
  `vader_lexicon.txt` and `emoji_utf8_lexicon.txt` shipped alongside the binary

## 3. Project structure

```
main.go          Wiring: logger, DB ping, goroutines, routes, graceful shutdown
background/      Long-running work: feed polling, crawl loop, health pings
db/              Connection, Queryable interface, pagination constants
env/             Environment variable parsing
gpt/             DEAD CODE — nothing imports it (see Rules)
logging/         slog setup, crawl summary counters, error dedup handler
middleware/      Request logging
query/           One file per query, squirrel-built
responder/       Standard success/error JSON envelopes
routers/         HTTP handlers
scraper/         Feed client, Colly crawler, per-domain + generic parsers
sentiment/       Sentiment worker and queue
state/           In-memory news cache (URL -> pipeline ID)
structs/         Domain types
tools/           Small helpers (pointer conversion, user agents)
```

## 4. Running, building & testing

Prerequisites: Go 1.23+, a reachable Postgres, and a `.env` with `CORE_DB`,
`OPENAI_KEY` and `PORT`.

```bash
dev          # go run .
dev build    # go build -o bin/app .
dev test     # go test ./...
dev fmt      # gofmt
dev vet      # go vet
dev check    # fmt + vet
dev tidy     # go mod tidy
```

Tests are hermetic — the crawler tests spin up `httptest` servers and need no
network or database.

### Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `CORE_DB` | — (required) | Postgres DSN |
| `OPENAI_KEY` | — (required) | Unused; see Rules |
| `PORT` | `8000` | Listen port |
| `LOG_LEVEL` | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `LOG_SUMMARY_INTERVAL` | `5m` | Crawl summary cadence and error dedup window |
| `CRAWL_INTERVAL` | `1m` | Time between crawl cycles |
| `CRAWL_BATCH_LIMIT` | `50` | Max articles attempted per cycle |
| `CRAWL_MAX_ATTEMPTS` | `3` | Failures before an article is skipped |

Bad values fall back to the default rather than stopping the service — a typo
in `LOG_LEVEL` must never prevent a boot.

## 5. How code is written here

Standard conventions apply (`Handle{Verb}{Entity}` handlers, `{entity}.query.go`
files, squirrel not ORM, `Queryable` as the first query argument). Repo-specific
rules:

**Logging is leveled and summarised.** The crawl loop touches every backlogged
article each cycle, so per-item logging floods the log store. The rules:

- **Per-item lines are `slog.Debug`.** They are invisible at the default level
  and recoverable by setting `LOG_LEVEL=DEBUG`.
- **Aggregate counts, don't log them.** Call `logging.Crawl.Inc*` on every
  outcome; one `crawl summary` line per `LOG_SUMMARY_INTERVAL` reports them.
- **Increment before logging.** Counters must not depend on whether a line
  survived level filtering or deduplication.
- **Errors log immediately at `slog.Error`**, with a constant message and the
  detail in attributes — never `fmt.Sprintf`-ed into the message, which would
  defeat deduplication.
- **`reason` and `domain` are the dedup key** and must stay low-cardinality.
  `reason` comes from the closed set in `scraper.classifyError`. Never put a
  URL, an ID, or a raw error string in either.

**Never persist an empty scrape.** `body_content = ''` still matches the
"needs crawling" filter, so writing one puts the article on a permanent
re-crawl treadmill. `scraper.Scrape` returns `ErrEmptyBody` for this; callers
must skip the update.

**Every domain has a parser.** `scraper.parserFor` returns a named parser for
known outlets and `parseGeneric` for everything else, so no URL is ever
unhandled. Adding an outlet means adding to `domainParsers` — never adding a
branch that leaves other domains unparsed.

## 6. Domain & architecture

Three goroutines run alongside the HTTP server, all cancelled by the same
signal-derived context:

1. **Crawl loop** (`runCrawlLoop` in `main.go`) — every `CRAWL_INTERVAL`:
   hydrate the news cache, poll the brief feed (`background.NewsFilter`), then
   crawl outstanding articles (`background.CheckCrawlers`).
2. **Sentiment worker** (`sentiment.StartSentimentWorker`) — drains a buffered
   channel of news items and scores them.
3. **Summary emitter** (`logging.Crawl.Run`) — one summary line per interval,
   plus a final line on shutdown.

**Crawl flow.** `CheckCrawlers` lists up to `CRAWL_BATCH_LIMIT` articles whose
`body_content` is null or empty, and for each: skip if it has already failed
`CRAWL_MAX_ATTEMPTS` times; otherwise `scraper.Scrape` it. Success writes the
body and author back and clears the attempt count. Any failure increments it.
Attempt tracking is in memory, so a restart gives every article a fresh set of
attempts — usually what you want, since a deploy is what fixes a bad parser.

**Extraction.** `Scrape` builds a Colly collector rooted at `<html>` and
dispatches to the parser for the host. `parseGeneric` prefers schema.org
JSON-LD (`articleBody`, `headline`, `author`), falling back to picking the
densest block of `<p>` text while excluding nav, header, footer, aside and
figure. A result under `MinBodyLength` is treated as no article at all.

**Auth:** none. The API is read-only and public.

## 7. Ecosystem & related repos

- `sentiment-scraper-web` — the frontend
- Feed source: `static.newsfilter.io` brief feed (external)
- Liveness: pings healthchecks.io on a timer
- No Forta, Keyring, or Monitor integration today; logs go to stdout/stderr and
  are collected by Lattice

## 8. Operations

- Deployed via **Lattice stack 11** (`sentimentscraper`), single container on
  worker 2, port 8001
- CI (`.github/workflows/push-to-registry.yml`) builds linux/amd64, pushes
  `registry.appleby.cloud/sentimentscraper:latest`, then calls the Lattice
  deploy URL
- Container healthcheck hits `/health`, which is excluded from request logging

**Reading the logs.** At the default level you should see startup lines, one
`crawl summary` per interval, and deduplicated errors. The summary is the
health signal:

```json
{"msg":"crawl summary","kind":"interval","interval_s":300,"backlog":42,
 "items_found":40,"items_scraped":31,"items_empty":4,"items_skipped":2,
 "items_failed":5,"fail_forbidden":4,"fail_timeout":1}
```

- `backlog` climbing over hours means articles are not being drained.
- `backlog` steady with `items_scraped: 0` means the crawler is wedged, not idle
  — the reason a summary is emitted even when every counter is zero.
- A large `items_skipped` means many articles have hit `CRAWL_MAX_ATTEMPTS`; a
  restart retries them.
- `fail_forbidden` concentrated on one domain means that outlet is blocking us.

To investigate a specific article, set `LOG_LEVEL=DEBUG` — every per-item line
carries `news_id`.

## 9. Rules & guardrails

- **Do not log per item in any loop.** Count it and let the summary report it.
- **Do not build log messages with `fmt.Sprintf`.** Constant message, variable
  attributes, or deduplication silently stops working.
- **Do not add high-cardinality values to `reason` or `domain`.**
- **Do not write an empty `body_content`.**
- **Do not use `log.Fatal` outside startup.** `slog.SetLogLoggerLevel` routes
  stdlib `log` to Debug, so a `log.Fatal` would exit the process while printing
  nothing. Use `logging.Fatal`, and never from inside an HTTP handler.
- `gpt/` is dead code and `OPENAI_KEY` is still required by `env`, so the
  service will not boot without a key it never uses. Removing both is safe but
  has not been done.
- `state.NewsCache` is not mutex-protected; it is currently safe only because a
  single goroutine touches it. Add a `sync.RWMutex` before sharing it.

## 10. Verification

```bash
gofmt -w . && go vet ./... && go build ./... && go test -race ./...
```

Behavioural checks:

```bash
LOG_LEVEL=INFO  dev    # summaries only, no per-item lines
LOG_LEVEL=DEBUG dev    # per-item lines return
```

## 11. Keeping this file updated

Any change to structure, stack, commands, conventions, endpoints, schema, or a
service contract updates this file **in the same change** — and `README.md` too
if setup, commands, or the repo's role changed. Stale docs mislead every future
reader, which is worse than no docs.
