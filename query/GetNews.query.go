package query

import (
	"fmt"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type GetNewsRequest struct {
	ID *int `json:"id"`
}

func GetNews(dbc db.Queryable, req GetNewsRequest) (*structs.News, error) {
	if req.ID == nil {
		return nil, fmt.Errorf("news ID is required")
	}

	news, err := ListNews(dbc, ListNewsRequest{
		ID: req.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting news: %w", err)
	}

	if len(news) == 0 {
		return nil, fmt.Errorf("no news found with ID: %d", *req.ID)
	}

	return &news[0], nil
}
