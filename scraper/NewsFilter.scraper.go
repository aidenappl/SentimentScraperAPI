// This package provides functionality to scrape news briefs from various APIs.
package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NewsItem represents a single news item with its details.
type NewsItem struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	Article   Article   `json:"article"`
}

// Article represents the details of an article associated with a news item.
type Article struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Symbols     []string `json:"symbols"`
	Source      Source   `json:"source"`
	PublishedAt string   `json:"publishedAt"`
}

// Source represents the source of the article.
type Source struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NewsFilterBriefs fetches news briefs from the NewsFilter API.
func NewsFilterBriefs() ([]NewsItem, error) {
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

	var newsItems []NewsItem
	if err := json.NewDecoder(response.Body).Decode(&newsItems); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return newsItems, nil
}
