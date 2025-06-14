package query

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type GetNewsSourceRequest struct {
	ID   *int    `json:"id"`
	Name *string `json:"name"`
}

func GetNewsSource(dbc db.Queryable, req GetNewsSourceRequest) (*structs.Outlet, error) {
	if req.ID == nil && req.Name == nil {
		return nil, fmt.Errorf("either ID or Name must be provided")
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	q := psql.Select(
		"id",
		"name",
		"website",
		"logo",
		"inserted_at",
	).From("website.outlets")

	if req.ID != nil {
		q = q.From("website.outlets").Where(sq.Eq{"id": *req.ID})
	}
	if req.Name != nil {
		q = q.From("website.outlets").Where(sq.Eq{"name": *req.Name})
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error building SQL query: %w", err)
	}

	var outlet structs.Outlet
	err = dbc.QueryRow(query, args...).Scan(
		&outlet.ID,
		&outlet.Name,
		&outlet.Website,
		&outlet.Logo,
		&outlet.InsertedAt,
	)
	if err != nil {
		if err == db.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error executing SQL query: %w", err)
	}

	return &outlet, nil

}
