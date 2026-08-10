package scraper

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aidenappl/SentimentScraperAPI/logging"
	"github.com/aidenappl/SentimentScraperAPI/tools"
	"github.com/gocolly/colly"
)

type ScrapedArticle struct {
	Title       string
	AuthorName  string
	ArticleBody string
	Category    *string
}

// ErrEmptyBody means the page was fetched but no usable article text came out
// of it. Callers must not persist the result: writing an empty body would mark
// the article as crawled and it would never be retried.
var ErrEmptyBody = errors.New("no article body extracted")

// requestDelay spaces out requests to the same host. It is a variable so tests
// can drop it to zero.
var requestDelay = 5 * time.Second

// Scrape fetches url and extracts the article.
//
// It returns an error for every outcome that did not produce a usable article,
// so callers can tell "fetched and parsed" from "fetched and empty" — a
// distinction the previous always-non-nil return could not express.
func Scrape(url string) (*ScrapedArticle, error) {
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.UserAgent(tools.UserAgent()),
	)

	c.SetRequestTimeout(30 * time.Second)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		RandomDelay: requestDelay,
	})

	article := &ScrapedArticle{}
	var fetchErr error

	// Rooted at <html>, not <body>: the generic parser reads JSON-LD and the
	// author/title meta tags, all of which live in <head>.
	c.OnHTML("html", func(e *colly.HTMLElement) {
		parse, named := parserFor(e.Request.URL.Hostname())
		parse(e, article)

		// Always run the generic extractor behind a named parser. It only
		// fills fields left empty, so the named parser keeps what it got right
		// while its gaps — a byline selector that has drifted, a body the
		// outlet re-templated — are covered rather than lost.
		if named {
			parseGeneric(e, article)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", tools.UserAgent())
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Referer", "https://www.google.com/")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "cross-site")
	})

	c.OnError(func(r *colly.Response, err error) {
		reason := classifyError(r, err)
		fetchErr = fmt.Errorf("%s: %w", reason, err)

		// Count before logging: the summary must not depend on whether this
		// particular line survived deduplication.
		logging.Crawl.IncFailure(reason)

		slog.Error("scrape request failed",
			"reason", reason,
			"domain", requestHost(r),
			"status", r.StatusCode,
			"url", requestURL(r),
			"err", err,
		)
	})

	if err := c.Visit(url); err != nil {
		// A request that reached the network and failed has already been
		// classified, counted and logged by OnError; Visit just surfaces the
		// same error again. Only a pre-flight rejection (robots.txt, a URL
		// filter, a bad scheme) is new information here.
		if fetchErr != nil {
			return nil, fetchErr
		}

		logging.Crawl.IncFailure("rejected")

		slog.Error("scrape visit rejected",
			"reason", "rejected",
			"domain", hostOf(url),
			"url", url,
			"err", err,
		)

		return nil, fmt.Errorf("visiting %s: %w", url, err)
	}

	c.Wait()

	if fetchErr != nil {
		return nil, fetchErr
	}

	if len(strings.TrimSpace(article.ArticleBody)) < MinBodyLength {
		return nil, ErrEmptyBody
	}

	return article, nil
}

// classifyError maps a failure onto a small closed set of reasons. The set
// must stay small: it keys both the summary breakdown and the error dedup
// map, and a reason derived from raw error text would make both unbounded.
func classifyError(r *colly.Response, err error) string {
	// Colly synthesises a Response with a zero StatusCode when the request
	// never completed, which is the reliable way to tell a transport failure
	// from an HTTP error.
	if r == nil || r.StatusCode == 0 {
		if err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
				return "timeout"
			}
		}
		return "transport"
	}

	switch {
	case r.StatusCode == http.StatusForbidden:
		return "forbidden"
	case r.StatusCode == http.StatusNotFound:
		return "not_found"
	case r.StatusCode == http.StatusTooManyRequests:
		return "rate_limited"
	case r.StatusCode >= 500:
		return "server_error"
	default:
		return "http_" + strconv.Itoa(r.StatusCode)
	}
}

func requestHost(r *colly.Response) string {
	if r == nil || r.Request == nil || r.Request.URL == nil {
		return ""
	}

	return r.Request.URL.Hostname()
}

func requestURL(r *colly.Response) string {
	if r == nil || r.Request == nil || r.Request.URL == nil {
		return ""
	}

	return r.Request.URL.String()
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	return u.Hostname()
}
