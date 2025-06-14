package sentiment

import (
	"fmt"

	"github.com/cdipaolo/sentiment"
	"github.com/drankou/go-vader/vader"
)

// multiclass text classification
func GenerateSentiment(text string) (*uint8, error) {
	model, err := sentiment.Restore()
	if err != nil {
		return nil, err
	}
	analysis := model.SentimentAnalysis(text, sentiment.English)

	return &analysis.Score, nil
}

// VADER sentiment analysis
func GenerateVaderSentiment(text string) (*map[string]float64, error) {
	sia := vader.SentimentIntensityAnalyzer{}
	err := sia.Init("vader_lexicon.txt", "emoji_utf8_lexicon.txt")
	if err != nil {
		return nil, err
	}

	score := sia.PolarityScores(text)
	fmt.Println(score)
	if score == nil {
		return nil, fmt.Errorf("failed to generate sentiment score")
	}
	return &score, nil
}
