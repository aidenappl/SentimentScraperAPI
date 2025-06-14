package background

import (
	"log"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/scraper"
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
		log.Printf("📰 %s: %s\n", item.Article.Title, item.Article.Source.Name)
		err := query.InsertNews(db.DB, item)
		if err != nil {
			log.Printf("❌ Error inserting news item: %v\n", err)
			continue
		}
		log.Printf("✅ Successfully inserted news item: %s\n", item.Article.Title)
	}

}
