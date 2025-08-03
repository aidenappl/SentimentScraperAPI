package sentiment

import (
	"context"
	"log"
	"time"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/gpt"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/structs"
	"github.com/aidenappl/SentimentScraperAPI/tools"
)

var NewsQueue = make(chan *structs.News, 1000) // buffered queue

func QueueSentimentProcessing(news *structs.News) {
	if news == nil || news.ID == nil {
		log.Println("❌ Cannot queue sentiment processing: news or news ID is nil")
		return
	}

	select {
	case NewsQueue <- news:
		news.Sentiment.StatusID = tools.IntP(2) // Set status to "Queued"
		err := query.UpdateSentiment(db.DB, news.Sentiment)
		if err != nil {
			log.Println("❌ Failed to update sentiment in database:", err)
			return
		}
		log.Println("✅ Queued news item for sentiment processing:", *news.ID)
	default:
		log.Println("⚠️ News queue is full. Dropping news item:", *news.ID)
	}
}

func processNewsSentiment(news *structs.News) {
	if news == nil || news.Sentiment == nil || news.BodyContent == nil {
		log.Println("❌ News or its required fields are nil. Skipping.")
		return
	}
	news.Sentiment.StatusID = tools.IntP(3) // Set status to "Processing"
	err := query.UpdateSentiment(db.DB, news.Sentiment)
	if err != nil {
		log.Println("❌ Failed to update sentiment in database:", err)
		return
	}

	gptSentiment, err := gpt.FetchSentimentFromChatGPT(*news)
	if err != nil {
		log.Println("❌ Failed to fetch sentiment from ChatGPT:", err)
		return
	}

	senti, err := GenerateSentiment(*news.BodyContent)
	if err != nil {
		log.Println("❌ Failed to generate sentiment:", err)
		return
	}

	vsenti, err := GenerateVaderSentiment(*news.BodyContent)
	if err != nil {
		log.Println("❌ Failed to generate VADER sentiment:", err)
		return
	}

	if gptSentiment == nil {
		log.Println("❌ GPT sentiment analysis returned nil")
		return
	} else {
		news.Sentiment.SentimentSummary = gptSentiment.SentimentSummary
		news.Sentiment.Score = gptSentiment.Score
		news.Sentiment.Positive = gptSentiment.Positive
		news.Sentiment.Negative = gptSentiment.Negative
		news.Sentiment.Neutral = gptSentiment.Neutral
		news.Sentiment.Confidence = gptSentiment.Confidence
		news.Sentiment.Polarity = gptSentiment.Polarity
		news.Sentiment.Subjectivity = gptSentiment.Subjectivity
		news.Sentiment.Language = gptSentiment.Language
		news.Sentiment.Source = gptSentiment.Source
	}

	if vsenti != nil {
		comp := (*vsenti)["compound"]
		pos := (*vsenti)["pos"]
		neg := (*vsenti)["neg"]
		neu := (*vsenti)["neu"]

		news.Sentiment.VaderPos = &pos
		news.Sentiment.VaderNeg = &neg
		news.Sentiment.VaderNeu = &neu
		news.Sentiment.VaderComp = &comp
	} else {
		log.Println("❌ VADER sentiment analysis returned nil")
		return
	}

	if senti != nil {
		news.Sentiment.MultitextClass = senti
	} else {
		log.Println("❌ Sentiment analysis returned nil")
		return
	}

	nowTime := time.Now().UTC()
	news.Sentiment.ProcessedAt = &nowTime
	news.Sentiment.StatusID = tools.IntP(4)

	err = query.UpdateSentiment(db.DB, news.Sentiment)
	if err != nil {
		log.Println("❌ Failed to update sentiment in database:", err)
		return
	}

	log.Println("✅ Sentiment processing completed for news item:", *news.ID)
}

func StartSentimentWorker(ctx context.Context) {
	for {
		select {
		case news := <-NewsQueue:
			log.Printf("🔍 Processing sentiment for news item: %d. %d left in queue", *news.ID, len(NewsQueue))
			processNewsSentiment(news)
		case <-ctx.Done():
			log.Println("🛑 Sentiment worker stopped.")
			return
		}
	}
}
