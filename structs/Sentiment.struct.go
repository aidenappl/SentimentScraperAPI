package structs

import "time"

type Sentiment struct {
	ID               *int        `json:"id"`                // Unique identifier for the sentiment analysis
	NewsID           *int        `json:"news_id"`           // ID of the associated news item
	SentimentSummary *string     `json:"sentiment_summary"` // Summary of the sentiment analysis
	Score            *float64    `json:"score"`             // Overall sentiment score
	Positive         *float64    `json:"positive"`          // Positive sentiment score
	Negative         *float64    `json:"negative"`          // Negative sentiment score
	Neutral          *float64    `json:"neutral"`           // Neutral sentiment score
	Confidence       *float64    `json:"confidence"`        // Confidence score of the sentiment analysis
	Polarity         *string     `json:"polarity"`          // Polarity of the sentiment (e.g., positive, negative, neutral)
	Subjectivity     *float64    `json:"subjectivity"`      // Subjectivity score of the sentiment analysis
	Language         *string     `json:"language"`          // Language of the sentiment analysis
	Source           *string     `json:"source"`
	ProcessedAt      *time.Time  `json:"processed_at"`
	InsertedAt       *time.Time  `json:"inserted_at"`     // Timestamp when the sentiment was inserted
	Status           *GeneralNSN `json:"status"`          // Status of the sentiment analysis (e.g., pending, completed, failed)
	VaderComp        *float64    `json:"vader_comp"`      // VADER compound score
	VaderPos         *float64    `json:"vader_pos"`       // VADER positive score
	VaderNeg         *float64    `json:"vader_neg"`       // VADER negative score
	VaderNeu         *float64    `json:"vader_neu"`       // VADER neutral score
	MultitextClass   *uint8      `json:"multitext_class"` // Multiclass classification score
}
