package query

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type CreateNewsSourceRequest struct {
	Name    string  `json:"name"`
	Website *string `json:"website"`
	Logo    *string `json:"logo"`
}

func CreateNewsSource(dbc db.Queryable, req CreateNewsSourceRequest) (*structs.Outlet, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	if req.Name == "" {
		return nil, fmt.Errorf("outlet name cannot be empty")
	}

	q := psql.Insert("website.outlets").
		Columns("name", "website", "logo").
		Values(req.Name, req.Website, req.Logo).
		Suffix("RETURNING id, name, website, logo, inserted_at")

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
		return nil, fmt.Errorf("error executing SQL query: %w", err)
	}

	return &outlet, nil
}
