package background

import (
	"errors"
	"log/slog"
	"time"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/env"
	"github.com/aidenappl/SentimentScraperAPI/logging"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/retry"
	"github.com/aidenappl/SentimentScraperAPI/scraper"
	"github.com/aidenappl/SentimentScraperAPI/structs"
	"github.com/aidenappl/SentimentScraperAPI/tools"
)

// retries schedules re-attempts for articles that failed to scrape.
//
// State is in memory by design: it needs no migration, and a restart giving
// every article a fresh start is usually what you want, since a deploy is
// what fixes a broken parser.
var retries = retry.New(env.CrawlRetryBackoff, env.CrawlRetryBackoffMax, env.CrawlMaxAttempts)

// CheckCrawlers fetches articles that have no body content yet and attempts to
// scrape each one.
//
// Articles in retry backoff are excluded from the query rather than filtered
// after it. The listing is ordered newest-first and the batch is capped, so
// filtering afterwards would hand back the same failing rows every cycle and
// the crawler would never reach the rest of the backlog.
func CheckCrawlers() {
	slog.Debug("checking for news items that need crawling")

	now := time.Now()
	deferred := retries.Deferred(now)

	total, err := query.CountNewsNeedingCrawl(db.DB)
	if err != nil {
		slog.Error("failed to count crawl backlog", "reason", "query", "err", err)
	} else {
		logging.Crawl.SetBacklog(total)
	}
	logging.Crawl.SetDeferred(len(deferred))

	news, err := query.ListNews(db.DB, query.ListNewsRequest{
		HasBodyContent: tools.BoolP(false),
		Limit:          tools.IntP(env.CrawlBatchLimit),
		ExcludeIDs:     deferred,
	})
	if err != nil {
		slog.Error("failed to list news items for crawling", "reason", "query", "err", err)
		return
	}

	if len(news) == 0 {
		slog.Debug("no news items are due for crawling", "backlog", total, "deferred", len(deferred))
		return
	}

	for _, item := range news {
		if item.ID == nil || item.ArticleURL == nil {
			continue
		}

		id, articleURL := *item.ID, *item.ArticleURL

		// Belt and braces: the query already excluded these, but the tracker
		// is the authority on eligibility.
		if !retries.Ready(id, time.Now()) {
			logging.Crawl.IncSkipped()
			continue
		}

		logging.Crawl.IncFound()
		slog.Debug("crawling news item", "news_id", id, "url", articleURL)

		article, err := scraper.Scrape(articleURL)
		if err != nil {
			retries.Fail(id, time.Now())

			// Scrape already logged and counted the fetch failure; an empty
			// body is the one outcome it cannot count, since it is not a
			// request error.
			if errors.Is(err, scraper.ErrEmptyBody) {
				logging.Crawl.IncEmpty()
				slog.Debug("no article body extracted", "news_id", id, "url", articleURL)
			}

			continue
		}

		if err := query.UpdateNews(db.DB, query.UpdateNewsRequest{
			ID: id,
			News: structs.News{
				BodyContent: &article.ArticleBody,
				Authors:     &article.AuthorName,
			},
		}); err != nil {
			retries.Fail(id, time.Now())
			logging.Crawl.IncFailure("update")
			slog.Error("failed to update news item", "reason", "update", "news_id", id, "err", err)
			continue
		}

		retries.Succeed(id)
		logging.Crawl.IncScraped()
		slog.Debug("scraped news item", "news_id", id, "chars", len(article.ArticleBody))
	}
}
