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
		outlet, err := query.FindOrAddNewsSource(db.DB, query.FindOrAddNewsSourceRequest{Name: item.Article.Source.Name})
		if err != nil {
			log.Printf("❌ Error fetching news source: %v\n", err)
			continue
		}
		err = query.InsertNews(db.DB, item, query.InsertNewsRequest{
			ArticleSourceID:  outlet.ID,
			UniquePipelineID: item.ID,
			DataPipelineID:   1,
		})
		if err != nil {
			log.Printf("❌ Error inserting news item: %v\n", err)
			continue
		}
	}
	log.Println("✅ News items processed or inserted successfully")

}
