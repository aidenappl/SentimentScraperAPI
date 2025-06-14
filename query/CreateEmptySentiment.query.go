package query

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

func CreateEmptySentiment(dbc db.Queryable, news_id int) (*structs.News, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query := psql.Insert("website.sentiment").
		Columns(
			"news_id",
			"sentiment_summary",
			"score",
			"positive",
			"negative",
			"neutral",
			"confidence",
			"polarity",
			"subjectivity",
			"language",
			"source",
			"processed_at",
			"vader_comp",
			"vader_pos",
			"vader_neg",
			"vader_neu",
			"multitext_classification").
		Values(
			news_id,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL for creating empty sentiment: %w", err)
	}

	if _, err := dbc.Exec(sql, args...); err != nil {
		return nil, fmt.Errorf("failed to create empty sentiment: %w", err)
	}

	return GetNews(dbc, GetNewsRequest{
		ID: &news_id,
	})
}
