package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	log.Println("Starting Betting Engine Service")

	// Initialize router
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// Internal API
	api := r.PathPrefix("/internal/v1").Subrouter()
	api.HandleFunc("/odds/live", getLiveOddsHandler).Methods("GET")
	api.HandleFunc("/odds/event/{eventId}", getEventOddsHandler).Methods("GET")
	api.HandleFunc("/bets/validate", validateBetHandler).Methods("POST")
	api.HandleFunc("/bets/calculate", calculateOddsHandler).Methods("POST")

	// Start odds sync from Sportradar
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go syncOddsFromProvider(ctx)

	// Start server
	port := os.Getenv("ENGINE_PORT")
	if port == "" {
		port = "8082"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Engine Service listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Engine Service...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Engine Service exited cleanly")
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"engine"}`))
}

func getLiveOddsHandler(w http.ResponseWriter, r *http.Request) {
	// Return live odds for in-play matches
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
		"matches": [
			{
				"event_id": "match_12345",
				"home_team": "Arsenal",
				"away_team": "Chelsea",
				"markets": {
					"1X2": {"home": 2.10, "draw": 3.40, "away": 3.20}
				}
			}
		]
	}`))
}

func getEventOddsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"odds":2.50,"available":true}`))
}

func validateBetHandler(w http.ResponseWriter, r *http.Request) {
	// Validate bet selections against current odds
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"valid":true,"total_odds":5.25}`))
}

func calculateOddsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"total_odds":8.50,"potential_win":850.00}`))
}

func syncOddsFromProvider(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("Starting odds sync from Sportradar...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Odds sync stopped")
			return
		case <-ticker.C:
			// In production:
			// 1. Fetch odds from Sportradar API
			// 2. Update Redis cache
			// 3. Publish odds update event via NATS
			// 4. Broadcast to WebSocket clients
			log.Println("Syncing odds... (mock)")
		}
	}
}
