package background

import (
	"log"
	"strings"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/scraper"
	"github.com/aidenappl/SentimentScraperAPI/structs"
	"github.com/aidenappl/SentimentScraperAPI/tools"
)

func CheckCrawlers() {
	log.Println("🔍 Checking for news items that need crawling...")
	news, err := query.ListNews(db.DB, query.ListNewsRequest{
		HasBodyContent: tools.BoolP(false),
	})
	if err != nil {
		log.Printf("❌ Error fetching news: %v\n", err)
		return
	}
	if len(news) == 0 {
		log.Println("No news items found that require crawling")
		return
	}
	for _, item := range news {
		// skip reuters articles silently
		if strings.Contains(*item.ArticleURL, "reuters.com") {
			continue
		}

		log.Printf("📰 Found news item for crawling: %v\n", *item.ID)
		article := scraper.Scrape(*item.ArticleURL)
		if article == nil {
			log.Printf("❌ Error scraping article for news item %d: article is nil\n", *item.ID)
			continue
		}
		log.Printf("✅ Successfully scraped article for news item %d\n", *item.ID)
		err := query.UpdateNews(db.DB, query.UpdateNewsRequest{
			ID: *item.ID,
			News: structs.News{
				BodyContent: &article.ArticleBody,
				Authors:     &article.AuthorName,
			},
		})
		if err != nil {
			log.Printf("❌ Error updating news item %d: %v\n", *item.ID, err)
		}
	}
}
