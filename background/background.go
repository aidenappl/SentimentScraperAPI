package background

import (
	"log"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/scraper"
	"github.com/aidenappl/SentimentScraperAPI/state"
)

func Google() {
	fetchGoogleRSS()
}

func NewsFilter() {
	news, err := scraper.NewsFilterBriefs()
	if err != nil {
		log.Printf("❌ Error fetching news: %v\n", err)
		return
	}
	if len(news) == 0 {
		log.Println("No news found")
	}

	for _, item := range news {
		// check if news item already exists
		if _, exists := state.GetFromNewsCache(item.Article.URL); exists {
			continue
		}

		outlet, err := query.FindOrAddNewsSource(db.DB, query.FindOrAddNewsSourceRequest{Name: item.Article.Source.Name})
		if err != nil {
			log.Printf("❌ Error fetching news source: %v\n", err)
			continue
		}

		scrapedArticle := scraper.Scrape(item.Article.URL)
		if scrapedArticle != nil {

		}
		err = query.InsertNews(db.DB, item, query.InsertNewsRequest{
			ArticleSourceID:  outlet.ID,
			UniquePipelineID: item.ID,
			DataPipelineID:   1,
			BodyContent:      scrapedArticle.ArticleBody,
			Authors:          scrapedArticle.AuthorName,
		})
		if err != nil {
			log.Printf("❌ Error inserting news item: %v\n", err)
			continue
		}
	}
	log.Println("✅ News items processed or inserted successfully")

}
