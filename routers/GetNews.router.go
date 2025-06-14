package routers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/gpt"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/responder"
	"github.com/aidenappl/SentimentScraperAPI/sentiment"
	"github.com/gorilla/mux"
)

func GetNews(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	newsID := vars["id"]

	if newsID == "" {
		responder.SendError(w, http.StatusBadRequest, "News ID is required", nil)
		return
	}

	idInt, err := strconv.Atoi(newsID)
	if err != nil {
		responder.SendError(w, http.StatusBadRequest, "Invalid news ID format", err)
		return
	}

	news, err := query.GetNews(db.DB, query.GetNewsRequest{
		ID: &idInt,
	})
	if err != nil {
		responder.SendError(w, http.StatusConflict, "Failed to get news", err)
		return
	}

	if news == nil {
		responder.SendErrorWithParams(w, "News not found", http.StatusNotFound, nil, nil)
		return
	}

	gptSentiment, err := gpt.FetchSentimentFromChatGPT(*news)
	if err != nil {
		log.Println("❌ Failed to fetch sentiment from ChatGPT:", err)
	}

	senti, err := sentiment.GenerateSentiment(*news.SummaryText)
	if err != nil {
		log.Println("❌ Failed to generate sentiment:", err)
	}

	vsenti, err := sentiment.GenerateVaderSentiment(*news.SummaryText)
	if err != nil {
		log.Println("❌ Failed to generate VADER sentiment:", err)
	}

	if gptSentiment == nil {
		log.Println("❌ GPT sentiment analysis returned nil")
	} else {
		news.Sentiment = gptSentiment
	}

	if vsenti == nil {
		log.Println("❌ VADER sentiment analysis returned nil")
	} else {
		comp := (*vsenti)["compound"]
		pos := (*vsenti)["pos"]
		neg := (*vsenti)["neg"]
		neu := (*vsenti)["neu"]

		news.Sentiment.VaderPos = &pos
		news.Sentiment.VaderNeg = &neg
		news.Sentiment.VaderNeu = &neu
		news.Sentiment.VaderComp = &comp
	}

	if senti == nil {
		log.Println("❌ Sentiment analysis returned nil")
	} else {
		news.Sentiment.MultitextClass = senti
	}

	responder.New(w, news)
}
