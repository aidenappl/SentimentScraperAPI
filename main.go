package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aidenappl/SentimentScraperAPI/background"
	"github.com/aidenappl/SentimentScraperAPI/db"
	"github.com/aidenappl/SentimentScraperAPI/env"
	"github.com/aidenappl/SentimentScraperAPI/logging"
	"github.com/aidenappl/SentimentScraperAPI/middleware"
	"github.com/aidenappl/SentimentScraperAPI/routers"
	"github.com/aidenappl/SentimentScraperAPI/sentiment"
	"github.com/aidenappl/SentimentScraperAPI/state"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	logging.Init(env.LogLevel, env.LogSummaryInterval)

	// Ping DB
	if err := db.PingDB(); err != nil {
		logging.Fatal("failed to connect to the database", "err", err)
	}
	slog.Info("connected to the database")

	// SIGTERM is what Docker sends on stop, and it arrives roughly ten seconds
	// before SIGKILL — everything below the cancel must finish inside that.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	go sentiment.StartSentimentWorker(ctx)

	// Emit one crawl summary per interval, plus a final line on shutdown.
	wg.Add(1)
	go func() {
		defer wg.Done()
		logging.Crawl.Run(ctx, env.LogSummaryInterval)
	}()

	// Hydrate News Cache
	if err := state.HydrateNewsCache(); err != nil {
		logging.Fatal("failed to hydrate news cache", "err", err)
	}
	slog.Info("news cache hydrated")

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
	wg.Add(1)
	go func() {
		defer wg.Done()
		runCrawlLoop(ctx)
	}()

	// CORS Middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8001",
			"https://sentimentscraper.com",
		},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	})

	// Start Healthcheck Polling
	go background.StartHealthCheckPolling(ctx)

	server := &http.Server{
		Addr:         ":" + env.Port,
		Handler:      corsMiddleware.Handler(r),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("SentimentScraper API listening",
			"port", env.Port,
			"log_level", env.LogLevel,
			"summary_interval", env.LogSummaryInterval.String(),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logging.Fatal("http server failed", "err", err)
		}
	}()

	<-ctx.Done()
	stop()
	slog.Info("shutdown signal received, draining")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown failed", "err", err)
	}

	// Wait for the crawl loop and the summary emitter: without this the
	// process exits before the final summary line is written, losing exactly
	// the interval that explains why the service stopped.
	wg.Wait()
	slog.Info("shutdown complete")
}

// runCrawlLoop polls the feed and crawls outstanding articles until ctx is
// cancelled. It uses a ticker rather than sleeping at the end of the body, so
// a slow cycle does not push every later cycle further out of step.
func runCrawlLoop(ctx context.Context) {
	cycle := func() {
		slog.Debug("fetching feeds")

		if err := state.HydrateNewsCache(); err != nil {
			slog.Error("failed to hydrate news cache", "reason", "query", "err", err)
			return
		}

		background.NewsFilter()
		background.CheckCrawlers()
	}

	cycle()

	t := time.NewTicker(env.CrawlInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			cycle()
		case <-ctx.Done():
			return
		}
	}
}
