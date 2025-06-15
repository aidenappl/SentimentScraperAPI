package routers

import (
	"net/http"
	"strconv"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/responder"
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

	responder.New(w, news)
}
