package routers

import (
	"net/http"

	"github.com/aidenappl/SentimentScraperAPI/responder"
	"github.com/aidenappl/SentimentScraperAPI/scraper"
)

// GetTrendingNews handles the request to fetch trending news.
func GetTrendingNews(w http.ResponseWriter, r *http.Request) {
	// Get most recent news
	news, err := scraper.NewsFilterBriefs()
	if err != nil {
		responder.SendError(w, http.StatusInternalServerError, "error fetching news", err)
		return
	}

	// Check if news is empty
	if len(news) == 0 {
		responder.SendError(w, http.StatusNotFound, "no news found")
		return
	}

	// Respond with the news items
	responder.New(w, news, "trending news fetched successfully")
}
