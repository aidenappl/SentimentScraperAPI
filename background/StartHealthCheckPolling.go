package background

import (
	"context"
	"log"
	"net/http"
	"time"
)

func StartHealthCheckPolling(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	healthCheckURL := "https://hc-ping.com/150e1ab6-6ca1-41c1-a4dd-8d0ef0e91765"

	for {
		select {
		case <-ticker.C:
			// Create HTTP client with timeout
			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			// Make GET request to health check endpoint
			resp, err := client.Get(healthCheckURL)
			if err != nil {
				log.Printf("❌ Health check failed: %v", err)
				continue
			}

			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				// log.Printf("✅ Health check successful: %d", resp.StatusCode) --- IGNORE ---
			} else {
				log.Printf("⚠️ Health check returned status: %d", resp.StatusCode)
			}

		case <-ctx.Done():
			log.Println("🚦 Stopping health check polling...")
			return
		}
	}
}
