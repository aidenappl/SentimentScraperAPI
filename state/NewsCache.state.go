package state

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

var newsCache = make(map[string]string)

func AddToNewsCache(articleID string, newsItem string) {
	if _, exists := newsCache[articleID]; !exists {
		newsCache[articleID] = newsItem
	}
}

func GetFromNewsCache(articleID string) (string, bool) {
	if newsItem, exists := newsCache[articleID]; exists {
		return newsItem, true
	}
	return "", false
}

func HydrateNewsCache() error {
	news, err := minifiedListNews(db.DB, minifiedListNewsRequest{
		Limit:  nil,
		Offset: nil,
		Sort:   nil,
	})
	if err != nil {
		return fmt.Errorf("error hydrating news cache: %w", err)
	}
	// Clear the existing cache
	newsCache = make(map[string]string)
	for _, item := range news {
		if _, exists := newsCache[*item.ArticleURL]; !exists {
			newsCache[*item.ArticleURL] = *item.UniquePipelineID
		}
	}
	return nil
}

type minifiedListNewsRequest struct {
	Limit  *int    `json:"limit"`
	Offset *int    `json:"offset"`
	Sort   *string `json:"sort"`
}

func minifiedListNews(dbc db.Queryable, req minifiedListNewsRequest) ([]structs.News, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	q := psql.Select(
		"id",
		"unique_pipeline_id",
		"article_url",
	).From("website.news")

	if req.Limit != nil {
		q = q.Limit(uint64(*req.Limit))
	}

	if req.Offset != nil {
		q = q.Offset(uint64(*req.Offset))
	}

	if req.Sort != nil {
		q = q.OrderBy(*req.Sort)
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error building SQL query: %w", err)
	}

	rows, err := dbc.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing SQL query: %w", err)
	}
	defer rows.Close()
	var newsItems []structs.News
	for rows.Next() {
		var newsItem structs.News
		err := rows.Scan(
			&newsItem.ID,
			&newsItem.UniquePipelineID,
			&newsItem.ArticleURL,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning rows: %w", err)
		}
		newsItems = append(newsItems, newsItem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return newsItems, nil
}
