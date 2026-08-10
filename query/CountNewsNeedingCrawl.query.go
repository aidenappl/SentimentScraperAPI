package query

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
)

// CountNewsNeedingCrawl returns how many articles still have no body content.
//
// This is the true backlog. The crawl batch is capped, so counting the rows a
// batch returned would just report the cap back and hide a growing queue.
func CountNewsNeedingCrawl(dbc db.Queryable) (int, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	q := psql.Select("COUNT(*)").
		From("website.news n").
		Where(sq.Or{sq.Eq{"n.body_content": nil}, sq.Eq{"n.body_content": ""}})

	query, args, err := q.ToSql()
	if err != nil {
		return 0, fmt.Errorf("error building SQL query: %w", err)
	}

	var count int
	if err := dbc.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("error executing SQL query: %w", err)
	}

	return count, nil
}
