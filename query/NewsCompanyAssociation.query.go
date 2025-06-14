package query

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
)

type NewsCompanyAssociationRequest struct {
	NewsID    *int    `json:"news_id"`
	CompanyID *int    `json:"company_id"`
	Ticker    *string `json:"ticker"`
}

func NewsCompanyAssociation(db db.Queryable, req NewsCompanyAssociationRequest) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	company, err := GetCompany(db, GetCompanyRequest{
		ID:     req.CompanyID,
		Ticker: req.Ticker,
	})
	if err != nil {
		return fmt.Errorf("error fetching company: %w", err)
	}

	if company == nil {
		return fmt.Errorf("company not found for ID: %v or Ticker: %v", req.CompanyID, req.Ticker)
	}

	q := psql.Insert("website.company_associations").
		Columns("news_id", "company_id").
		Values(req.NewsID, company.ID)
	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("error building SQL query: %w", err)
	}
	_, err = db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error executing SQL query: %w", err)
	}
	fmt.Println("Inserted news-company association for News ID:", req.NewsID, "and Company ID:", company.ID)
	return nil

}
