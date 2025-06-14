package structs

import "time"

type News struct {
	ID               *int       `json:"id"`
	Title            *string    `json:"title"`
	SummaryText      *string    `json:"summary_text"`
	PostedAt         *time.Time `json:"posted_at"`
	ArticleSource    *Outlet    `json:"article_source"`
	DataPipelineID   *string    `json:"data_pipeline_id"`
	UniquePipelineID *string    `json:"unique_pipeline_id"`
	ArticleURL       *string    `json:"article_url"`
	Companies        *[]Company `json:"companies"`
	Tickers          *[]string  `json:"tickers"`
	Sentiment        *Sentiment `json:"sentiment"`
	InsertedAt       *time.Time `json:"inserted_at"`
	BodyContent      *string    `json:"body_content"`
	Authors          *string    `json:"authors"`
}
