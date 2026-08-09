package background

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/env"
	"github.com/aidenappl/SentimentScraperAPI/logging"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/scraper"
	"github.com/aidenappl/SentimentScraperAPI/structs"
	"github.com/aidenappl/SentimentScraperAPI/tools"
)

// attempts tracks consecutive crawl failures per news item so a permanently
// unreachable article stops being retried every cycle.
//
// This is deliberately in memory: it needs no migration, and a restart giving
// every article a fresh set of attempts is the desired behaviour — a deploy
// is usually what fixes a broken parser.
var attempts sync.Map // news item ID -> int

// CheckCrawlers fetches articles that have no body content yet and attempts to
// scrape each one.
//
// Per-item lines are logged at Debug and counted into the crawl summary. An
// article that cannot be scraped is never written back with an empty body:
// doing so would leave it matching the "needs crawling" filter forever, which
// is what made this loop re-fetch the same articles every cycle indefinitely.
func CheckCrawlers() {
	slog.Debug("checking for news items that need crawling")

	news, err := query.ListNews(db.DB, query.ListNewsRequest{
		HasBodyContent: tools.BoolP(false),
		Limit:          tools.IntP(env.CrawlBatchLimit),
	})
	if err != nil {
		slog.Error("failed to list news items for crawling", "reason", "query", "err", err)
		return
	}

	logging.Crawl.SetBacklog(len(news))

	if len(news) == 0 {
		slog.Debug("no news items require crawling")
		return
	}

	for _, item := range news {
		if item.ID == nil || item.ArticleURL == nil {
			continue
		}

		id, articleURL := *item.ID, *item.ArticleURL

		if failed, _ := attempts.Load(id); failed != nil && failed.(int) >= env.CrawlMaxAttempts {
			logging.Crawl.IncSkipped()
			slog.Debug("skipping news item past its attempt limit", "news_id", id, "url", articleURL)
			continue
		}

		logging.Crawl.IncFound()
		slog.Debug("crawling news item", "news_id", id, "url", articleURL)

		article, err := scraper.Scrape(articleURL)
		if err != nil {
			recordFailure(id)

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
			recordFailure(id)
			logging.Crawl.IncFailure("update")
			slog.Error("failed to update news item", "reason", "update", "news_id", id, "err", err)
			continue
		}

		attempts.Delete(id)
		logging.Crawl.IncScraped()
		slog.Debug("scraped news item", "news_id", id, "chars", len(article.ArticleBody))
	}
}

func recordFailure(id int) {
	prior, _ := attempts.Load(id)
	count := 1
	if prior != nil {
		count = prior.(int) + 1
	}
	attempts.Store(id, count)
}
