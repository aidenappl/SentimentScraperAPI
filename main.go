package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/aidenappl/SentimentScraperAPI/background"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/env"
	"github.com/aidenappl/SentimentScraperAPI/middleware"
	"github.com/aidenappl/SentimentScraperAPI/routers"
	"github.com/aidenappl/SentimentScraperAPI/sentiment"
	"github.com/aidenappl/SentimentScraperAPI/state"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Ping DB
	if err := db.PingDB(); err != nil {
		log.Fatalf("❌ Failed to connect to the database: %v", err)
	} else {
		log.Println("✅ Connected to the database successfully")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sentiment.StartSentimentWorker(ctx)

	// Hydrate News Cache
	err := state.HydrateNewsCache()
	if err != nil {
		log.Fatalf("❌ Failed to hydrate news cache: %v", err)
	} else {
		log.Println("✅ News cache hydrated successfully")
	}

	r := mux.NewRouter()

	// Request logger
	r.Use(middleware.LoggingMiddleware)

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
	core.HandleFunc("/hydrateTickers", routers.HydrateTickers).Methods(http.MethodPost)
	core.HandleFunc("/news", routers.ListNews).Methods(http.MethodGet)
	core.HandleFunc("/news/{id}", routers.GetNews).Methods(http.MethodGet)

	// Background Handlers
	go func() {
		for {
			log.Println("📰 Fetching feeds...")
			state.HydrateNewsCache()
			background.NewsFilter()
			background.CheckCrawlers()
			time.Sleep(1 * time.Minute)
		}
	}()

	// CORS Middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"https://sentimentscraper.com",
		},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	})

	log.Printf("✅ SentimentScraper API running on port %s\n", env.Port)
	log.Fatal(http.ListenAndServe(":"+env.Port, corsMiddleware.Handler(r)))
}
