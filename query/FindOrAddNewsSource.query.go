package query

import (
	"fmt"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type FindOrAddNewsSourceRequest struct {
	Name string `json:"name"`
}

var outletCache = make(map[string]*structs.Outlet)

func FindOrAddNewsSource(db db.Queryable, req FindOrAddNewsSourceRequest) (*structs.Outlet, error) {

	// Check if the outlet is already cached
	if outlet, exists := outletCache[req.Name]; exists {
		return outlet, nil
	}

	// If not cached, fetch from the database
	outlet, err := GetNewsSource(db, GetNewsSourceRequest{Name: &req.Name})
	if err != nil {
		return nil, fmt.Errorf("error getting outlet: %w", err)
	}

	if outlet != nil {
		outletCache[req.Name] = outlet
		return outlet, nil
	}

	outlet, err = CreateNewsSource(db, CreateNewsSourceRequest{
		Name:    req.Name,
		Website: nil,
		Logo:    nil,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating outlet: %w", err)
	}
	outletCache[req.Name] = outlet
	return outlet, nil
}
