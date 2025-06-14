package routers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
)

type SECCompany struct {
	CIK    int    `json:"cik_str"`
	Ticker string `json:"ticker"`
	Title  string `json:"title"`
}

type SECCompanies map[string]SECCompany

func HydrateTickers(w http.ResponseWriter, r *http.Request) {
	companies, err := fetchSECJSON()
	if err != nil {
		log.Fatalf("❌ Failed to fetch/parse SEC data: %v", err)
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	db := db.DB

	for _, c := range companies {
		if c.Ticker == "" || c.Title == "" {
			continue
		}
		query, args, err := psql.
			Insert("website.companies").
			Columns("name", "ticker", "cik").
			Values(c.Title, c.Ticker, c.CIK).
			ToSql()
		if err != nil {
			log.Printf("❌ Failed to build SQL query for %s (%s): %v", c.Title, c.Ticker, err)
			continue
		}
		if _, err := db.Exec(query, args...); err != nil {
			log.Printf("❌ Failed to insert %s (%s): %v", c.Title, c.Ticker, err)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Tickers hydrated successfully"))
}

func fetchSECJSON() (SECCompanies, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://www.sec.gov/files/company_tickers.json", nil)
	if err != nil {
		return nil, err
	}

	// Add required User-Agent header
	req.Header.Set("User-Agent", "MyTickerBot/1.0 (contact: aiden@trailblaze.to)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var companies SECCompanies
	if err := json.Unmarshal(body, &companies); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return companies, nil
}
