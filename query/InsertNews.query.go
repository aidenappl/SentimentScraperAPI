package query

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type InsertNewsRequest struct {
	ArticleSourceID  int    `json:"article_source_id"`
	UniquePipelineID string `json:"unique_pipeline_id"`
	DataPipelineID   int    `json:"data_pipeline_id"`
}

func InsertNews(db db.Queryable, newsItem structs.NewsItem, newsMetadata InsertNewsRequest) error {
	q := sq.Insert("website.news").
		Columns(
			"title",
			"summary_text",
			"posted_at",
			"article_source",
			"data_pipeline_id",
			"unique_pipeline_id",
			"article_url",
		).
		Values(
			newsItem.Article.Title,
			newsItem.Text,
			newsItem.Article.PublishedAt,
			newsMetadata.ArticleSourceID,
			newsItem.Article.Summary,
		)

	query, args, err := q.ToSql()
	if err != nil {
		return err
	}

	_, err = db.Exec(query, args...)
	return err
}
