package gpt

import (
	"fmt"

	"github.com/aidenappl/SentimentScraperAPI/structs"
)

func buildPrompt(article structs.News) string {
	return fmt.Sprintf(`
You are a sentiment analysis engine, you only respond in the given JSON structure do not respond in any other way. Use VADER, TextBlob or general analysis to determine your results. Given this article data, extract the following fields:
- sentiment_summary (concise overview of sentiment and why)
- score (range: -1 to 1) float64
- positive, negative, neutral (range: 0 to 1) float64
- confidence (range: 0 to 1) float64
- polarity (positive, negative, or neutral)
- subjectivity (range: 0 to 1) float64
- language (ISO code like "en")
- source (write "chatgpt")

ARTICLE:
Title: %s
Summary: %s
Publisher: %s

Return only a JSON object with keys matching exactly the struct fields.
`, *article.Title, *article.SummaryText, article.ArticleSource.Name)
}
