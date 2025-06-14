package structs

import "time"

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
