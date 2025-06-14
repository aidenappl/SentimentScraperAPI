package query

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type UpdateNewsRequest struct {
	ID int `json:"id"`
	structs.News
}

func UpdateNews(db db.Queryable, req UpdateNewsRequest) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	if req.ID == 0 {
		return fmt.Errorf("ID is required for updating news")
	}
	setMap := map[string]interface{}{}
	if req.Title != nil {
		setMap["title"] = *req.Title
	}
	if req.SummaryText != nil {
		setMap["summary_text"] = *req.SummaryText
	}
	if req.PostedAt != nil {
		setMap["posted_at"] = *req.PostedAt
	}
	if req.ArticleSource != nil {
		setMap["article_source"] = req.ArticleSource.ID
	}
	if req.DataPipelineID != nil {
		setMap["data_pipeline_id"] = *req.DataPipelineID
	}
	if req.UniquePipelineID != nil {
		setMap["unique_pipeline_id"] = *req.UniquePipelineID
	}
	if req.ArticleURL != nil {
		setMap["article_url"] = *req.ArticleURL
	}
	if req.BodyContent != nil {
		setMap["body_content"] = *req.BodyContent
	}
	if req.Authors != nil {
		setMap["authors"] = *req.Authors
	}
	q := psql.Update("website.news").
		SetMap(setMap).
		Where(sq.Eq{"id": req.ID})

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("error building SQL query: %w", err)
	}

	_, err = db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error executing SQL query: %w", err)
	}
	return nil

}
