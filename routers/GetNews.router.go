package routers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/responder"
	"github.com/aidenappl/SentimentScraperAPI/sentiment"
	"github.com/aidenappl/SentimentScraperAPI/tools"
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

	if news.Sentiment == nil {
		if news, err = query.CreateEmptySentiment(db.DB, *news.ID); err != nil {
			log.Println("❌ Failed to create empty sentiment:", err)
			responder.SendError(w, http.StatusInternalServerError, "Failed to create empty sentiment", err)
			return
		}

		sentiment.QueueSentimentProcessing(news)
	} else {
		fmt.Println(news.Sentiment.Status.ID)
		if news.Sentiment.Status.ID != tools.IntP(4) {
			sentiment.QueueSentimentProcessing(news)
		}
	}

	responder.New(w, news)
}
