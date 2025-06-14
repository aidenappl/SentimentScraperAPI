package query

import (
	"fmt"
	"log"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/state"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type InsertNewsRequest struct {
	ArticleSourceID  int    `json:"article_source_id"`
	UniquePipelineID string `json:"unique_pipeline_id"`
	DataPipelineID   int    `json:"data_pipeline_id"`
	BodyContent      string `json:"body_content"`
	Authors          string `json:"authors"`
}

func InsertNews(dbc db.Queryable, newsItem structs.NewsItem, newsMetadata InsertNewsRequest) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	// check if news item already exists
	if _, exists := state.GetFromNewsCache(newsItem.Article.URL); exists {
		return nil
	}

	log.Println("Inserting news item into database:", newsItem.Article.ID)

	q := psql.Insert("website.news").
		Columns(
			"title",
			"summary_text",
			"posted_at",
			"article_source",
			"data_pipeline_id",
			"unique_pipeline_id",
			"article_url",
			"body_content",
			"authors",
		).
		Values(
			newsItem.Article.Title,
			newsItem.Text,
			newsItem.Article.PublishedAt,
			newsMetadata.ArticleSourceID,
			newsMetadata.DataPipelineID,
			newsMetadata.UniquePipelineID,
			newsItem.Article.URL,
			newsMetadata.BodyContent,
			newsMetadata.Authors,
		).Suffix("RETURNING id")

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("error building SQL query: %w", err)
	}

	var newsItemID int
	err = dbc.QueryRow(query, args...).Scan(&newsItemID)
	if err != nil {
		return fmt.Errorf("error inserting and querying news item: %w", err)
	}

	// Cache the news item
	state.AddToNewsCache(newsItem.Article.URL, newsItem.Article.ID)
	log.Println("Inserted news item into database and cached:", newsItem.Article.ID)

	// Add Company Associations
	if len(newsItem.Article.Symbols) > 0 {
		for _, symbol := range newsItem.Article.Symbols {
			// Assuming there's a function to add company associations
			err := NewsCompanyAssociation(dbc, NewsCompanyAssociationRequest{
				NewsID:    &newsItemID,
				CompanyID: nil, // We will find the company by ticker
				Ticker:    &symbol,
			})
			if err != nil {
				log.Printf("❌ Error adding company association for %s: %v\n", symbol, err)
				continue
			}
		}
	}

	return nil
}
