package query

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

func UpdateSentiment(dbc db.Queryable, sentiment *structs.Sentiment) error {
	if sentiment == nil {
		return fmt.Errorf("sentiment cannot be nil")
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.Update("website.sentiment").
		Set("sentiment_summary", sentiment.SentimentSummary).
		Set("score", sentiment.Score).
		Set("positive", sentiment.Positive).
		Set("negative", sentiment.Negative).
		Set("neutral", sentiment.Neutral).
		Set("confidence", sentiment.Confidence).
		Set("polarity", sentiment.Polarity).
		Set("subjectivity", sentiment.Subjectivity).
		Set("language", sentiment.Language).
		Set("source", sentiment.Source).
		Set("status", sentiment.StatusID).
		Set("processed_at", sentiment.ProcessedAt).
		Set("vader_comp", sentiment.VaderComp).
		Set("vader_pos", sentiment.VaderPos).
		Set("vader_neg", sentiment.VaderNeg).
		Set("vader_neu", sentiment.VaderNeu).
		Set("multitext_classification", sentiment.MultitextClass).
		Where(sq.Eq{"id": sentiment.ID})
	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for updating sentiment: %w", err)
	}
	if _, err := dbc.Exec(sql, args...); err != nil {
		return fmt.Errorf("failed to update sentiment: %w", err)
	}
	return nil
}
