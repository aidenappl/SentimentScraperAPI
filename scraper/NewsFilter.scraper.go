// This package provides functionality to scrape news briefs from various APIs.
package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aidenappl/SentimentScraperAPI/structs"
)

// NewsFilterBriefs fetches news briefs from the NewsFilter API.
func NewsFilterBriefs() ([]structs.NewsItem, error) {
	request, err := http.NewRequest(http.MethodGet, "https://static.newsfilter.io/landing-page/briefs.json", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var newsItems []structs.NewsItem
	if err := json.NewDecoder(response.Body).Decode(&newsItems); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return newsItems, nil
}
