package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/env"
	"github.com/aidenappl/SentimentScraperAPI/routers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Ping DB
	if err := db.PingDB(); err != nil {
		log.Fatalf("❌ Failed to connect to the database: %v", err)
	}

	r := mux.NewRouter()

	// Base API Endpoint
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Welcome to the SentimentScraper API!"))
	}).Methods(http.MethodGet)

	// Health Check Endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods(http.MethodGet)

	// Core V1 API Endpoint
	core := r.PathPrefix("/core/v1/").Subrouter()

	// Get All News
	core.HandleFunc("/trending", routers.GetTrendingNews).Methods(http.MethodGet)

	// CORS Middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"https://newsfilter.io",
		},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	})

	fmt.Printf("✅ SentimentScraper API running on port %s\n", env.Port)
	log.Fatal(http.ListenAndServe(":"+env.Port, corsMiddleware.Handler(r)))
}
