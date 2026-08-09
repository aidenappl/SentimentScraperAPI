package routers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/responder"
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
		// This used to be log.Fatalf, which killed the whole service whenever
		// the SEC returned something unexpected.
		slog.Error("failed to fetch SEC ticker data", "reason", "upstream", "err", err)
		responder.SendError(w, http.StatusBadGateway, "failed to fetch SEC ticker data", err)
		return
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
			slog.Debug("failed to build ticker insert", "ticker", c.Ticker, "err", err)
			continue
		}
		if _, err := db.Exec(query, args...); err != nil {
			slog.Debug("failed to insert ticker", "ticker", c.Ticker, "err", err)
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
