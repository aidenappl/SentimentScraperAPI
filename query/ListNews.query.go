package query

import (
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type ListNewsRequest struct {
	Limit  *int    `json:"limit"`
	Offset *int    `json:"offset"`
	Sort   *string `json:"sort"`

	HasBodyContent *bool `json:"has_body_content"`

	// Selectors
	ID *int `json:"id"`
}

func ListNews(dbc db.Queryable, req ListNewsRequest) ([]structs.News, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	// Build subquery for companies
	companiesSubquery := `
    SELECT ca.news_id, array_agg(c.ticker) AS companies
    FROM website.company_associations ca
    LEFT JOIN website.companies c ON c.id = ca.company_id
    GROUP BY ca.news_id
`

	sentimentSubquery := `
    SELECT DISTINCT ON (news_id) *
    FROM website.sentiment
    ORDER BY news_id, processed_at DESC
`

	q := psql.Select(
		// --- News fields ---
		"n.id",
		"n.title",
		"n.summary_text",
		"n.posted_at",
		"n.data_pipeline_id",
		"n.unique_pipeline_id",
		"n.article_url",
		"n.inserted_at",
		"n.body_content",
		"n.authors",

		// --- Article Source ---
		"s.id",
		"s.name",
		"s.website",
		"s.logo",
		"s.inserted_at",

		// --- Companies ---
		"co.companies",

		// --- Sentiment ---
		"snt.id",
		"snt.news_id",
		"snt.sentiment_summary",
		"snt.score",
		"snt.positive",
		"snt.negative",
		"snt.neutral",
		"snt.confidence",
		"snt.polarity",
		"snt.subjectivity",
		"snt.language",
		"snt.source",
		"snt.processed_at",
		"snt.inserted_at",
		"snt.vader_comp",
		"snt.vader_pos",
		"snt.vader_neg",
		"snt.vader_neu",
		"snt.multitext_classification",

		// --- Sentiment Status ---
		"snt_status.id",
		"snt_status.name",
		"snt_status.short_name",
	).
		From("website.news n").
		LeftJoin("website.outlets s ON s.id = n.article_source").
		LeftJoin(fmt.Sprintf("(%s) snt ON snt.news_id = n.id", sentimentSubquery)).
		LeftJoin("website.sentiment_statuses snt_status ON snt.status = snt_status.id").
		LeftJoin(fmt.Sprintf("(%s) co ON co.news_id = n.id", companiesSubquery)).
		OrderBy("n.posted_at DESC")

	if req.Limit != nil {
		q = q.Limit(uint64(*req.Limit))
	}

	if req.Offset != nil {
		q = q.Offset(uint64(*req.Offset))
	}

	if req.Sort != nil {
		q = q.OrderBy(*req.Sort)
	}

	if req.ID != nil {
		q = q.Where(sq.Eq{"n.id": *req.ID})
	}

	if req.HasBodyContent != nil && !*req.HasBodyContent {
		q = q.Where(sq.Or{sq.Eq{"n.body_content": nil}, sq.Eq{"n.body_content": ""}})
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
		var articleSource structs.Outlet
		var companies sql.NullString
		var sentiment structs.Sentiment
		var sentimentStatus structs.GeneralNSN
		err := rows.Scan(
			&newsItem.ID,
			&newsItem.Title,
			&newsItem.SummaryText,
			&newsItem.PostedAt,
			&newsItem.DataPipelineID,
			&newsItem.UniquePipelineID,
			&newsItem.ArticleURL,
			&newsItem.InsertedAt,
			&newsItem.BodyContent,
			&newsItem.Authors,

			&articleSource.ID,
			&articleSource.Name,
			&articleSource.Website,
			&articleSource.Logo,
			&articleSource.InsertedAt,

			&companies,

			&sentiment.ID,
			&sentiment.NewsID,
			&sentiment.SentimentSummary,
			&sentiment.Score,
			&sentiment.Positive,
			&sentiment.Negative,
			&sentiment.Neutral,
			&sentiment.Confidence,
			&sentiment.Polarity,
			&sentiment.Subjectivity,
			&sentiment.Language,
			&sentiment.Source,
			&sentiment.ProcessedAt,
			&sentiment.InsertedAt,
			&sentiment.VaderComp,
			&sentiment.VaderPos,
			&sentiment.VaderNeg,
			&sentiment.VaderNeu,
			&sentiment.MultitextClass,

			&sentimentStatus.ID,
			&sentimentStatus.Name,
			&sentimentStatus.ShortName,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning rows: %w", err)
		}
		newsItem.ArticleSource = &articleSource
		if sentiment.ID != nil {
			newsItem.Sentiment = &sentiment
			newsItem.Sentiment.Status = &sentimentStatus
		} else {
			newsItem.Sentiment = nil
		}
		if companies.Valid && companies.String != "" && companies.String != "{NULL}" {
			str := strings.TrimSpace(companies.String)
			str = strings.Trim(str, "{}")
			tickers := strings.Split(str, ",")
			newsItem.Tickers = &tickers
		}
		newsItems = append(newsItems, newsItem)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return newsItems, nil
}
