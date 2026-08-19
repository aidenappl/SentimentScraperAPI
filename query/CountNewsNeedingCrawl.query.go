package query

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
)

// CountNewsNeedingCrawl returns how many articles still have no body content,
// excluding any whose URL matches one of excludeURLPatterns.
//
// This is the true backlog. The crawl batch is capped, so counting the rows a
// batch returned would just report the cap back and hide a growing queue.
// Blocked outlets are excluded because they can never be crawled — counting
// them would hold the backlog permanently above zero and destroy its value as
// a health signal.
func CountNewsNeedingCrawl(dbc db.Queryable, excludeIDs []int, excludeURLPatterns []string) (int, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	q := psql.Select("COUNT(*)").
		From("website.news n").
		Where(sq.Or{sq.Eq{"n.body_content": nil}, sq.Eq{"n.body_content": ""}})

	if len(excludeIDs) > 0 {
		q = q.Where(sq.NotEq{"n.id": excludeIDs})
	}

	for _, pattern := range excludeURLPatterns {
		q = q.Where(sq.NotILike{"n.article_url": pattern})
	}

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
