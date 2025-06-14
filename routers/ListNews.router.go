package routers

import (
	"net/http"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/query"
	"github.com/aidenappl/SentimentScraperAPI/responder"
	"github.com/aidenappl/SentimentScraperAPI/structs"
	"github.com/aidenappl/SentimentScraperAPI/tools"
)

type ListNewsRequest struct {
	structs.BaseListRequest
}

func ListNews(w http.ResponseWriter, r *http.Request) {
	var req ListNewsRequest
	if err := tools.ParseURLParams(r, &req); err != nil {
		responder.SendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	news, err := query.ListNews(db.DB, query.ListNewsRequest{
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		responder.SendError(w, http.StatusInternalServerError, "Failed to list news", err)
		return
	}
	if len(news) == 0 {
		responder.SendErrorWithParams(w, "No news found", http.StatusNotFound, nil, nil)
		return
	}
	responder.New(w, news)
}
