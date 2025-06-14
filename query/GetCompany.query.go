package query

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type GetCompanyRequest struct {
	ID     *int    `json:"id"`
	Ticker *string `json:"ticker"`
	Name   *string `json:"name"`
}

func GetCompany(dbc db.Queryable, req GetCompanyRequest) (*structs.Company, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	q := psql.Select(
		"id",
		"name",
		"ticker",
		"website",
		"logo",
		"inserted_at",
		"cik",
	).From("website.companies")

	if req.ID != nil {
		q = q.Where(sq.Eq{"id": *req.ID})
	}
	if req.Ticker != nil {
		//  replace . with % for ILike
		ticker := strings.Replace(*req.Ticker, ".", "%", -1)

		q = q.Where(sq.ILike{"ticker": ticker})
	}
	if req.Name != nil {
		q = q.Where(sq.Eq{"name": *req.Name})
	}
	if req.ID == nil && req.Ticker == nil && req.Name == nil {
		return nil, fmt.Errorf("at least one of ID, Ticker, or Name must be provided")
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error building SQL query: %w", err)
	}

	var company structs.Company
	err = dbc.QueryRow(query, args...).Scan(
		&company.ID,
		&company.Name,
		&company.Ticker,
		&company.Website,
		&company.Logo,
		&company.InsertedAt,
		&company.CIK,
	)
	if err != nil {
		if err == db.ErrNoRows {

		}
		return nil, fmt.Errorf("error fetching company: %w", err)
	}

	return &company, nil
}
