package background

import (
	"log/slog"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/scraper"
	"github.com/aidenappl/SentimentScraperAPI/state"
)

func Google() {
	fetchGoogleRSS()
}

// NewsFilter pulls the brief feed and inserts anything not already known.
//
// A first scrape is attempted inline, but a failure is not fatal: the item is
// inserted with an empty body and CheckCrawlers retries it on a later cycle,
// under the same attempt limit.
func NewsFilter() {
	news, err := scraper.NewsFilterBriefs()
	if err != nil {
		slog.Error("failed to fetch news briefs", "reason", "feed", "err", err)
		return
	}

	if len(news) == 0 {
		slog.Debug("news brief feed returned no items")
		return
	}

	for _, item := range news {
		// Check if the news item already exists.
		if _, exists := state.GetFromNewsCache(item.Article.URL); exists {
			continue
		}

		outlet, err := query.FindOrAddNewsSource(db.DB, query.FindOrAddNewsSourceRequest{Name: item.Article.Source.Name})
		if err != nil {
			slog.Error("failed to resolve news source", "reason", "query", "outlet", item.Article.Source.Name, "err", err)
			continue
		}

		// A failed first scrape leaves body and author empty; the item still
		// gets inserted so CheckCrawlers can pick it up.
		var body, authors string
		if article, err := scraper.Scrape(item.Article.URL); err == nil {
			body, authors = article.ArticleBody, article.AuthorName
		}

		if err := query.InsertNews(db.DB, item, query.InsertNewsRequest{
			ArticleSourceID:  outlet.ID,
			UniquePipelineID: item.ID,
			DataPipelineID:   1,
			BodyContent:      body,
			Authors:          authors,
		}); err != nil {
			slog.Error("failed to insert news item", "reason", "insert", "url", item.Article.URL, "err", err)
			continue
		}
	}

	slog.Debug("news brief feed processed", "items", len(news))
}
