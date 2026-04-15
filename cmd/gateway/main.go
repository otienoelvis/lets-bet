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
	"github.com/gorilla/websocket"
)

const (
	defaultPort = "8080"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, validate origin
	},
}

func main() {
	log.Println("Starting Betting Platform Gateway")

	// Initialize router
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Auth routes
	api.HandleFunc("/auth/register", registerHandler).Methods("POST")
	api.HandleFunc("/auth/login", loginHandler).Methods("POST")

	// User routes (protected)
	api.HandleFunc("/users/me", getUserHandler).Methods("GET")
	api.HandleFunc("/users/me/wallet", getWalletHandler).Methods("GET")

	// Betting routes
	api.HandleFunc("/bets", placeBetHandler).Methods("POST")
	api.HandleFunc("/bets/{id}", getBetHandler).Methods("GET")
	api.HandleFunc("/bets/history", getBetHistoryHandler).Methods("GET")

	// Payment routes
	api.HandleFunc("/payments/deposit", depositHandler).Methods("POST")
	api.HandleFunc("/payments/withdraw", withdrawHandler).Methods("POST")
	api.HandleFunc("/payments/mpesa/callback", mpesaCallbackHandler).Methods("POST")

	// Games routes
	api.HandleFunc("/games/crash/current", getCurrentGameHandler).Methods("GET")
	api.HandleFunc("/games/crash/history", getGameHistoryHandler).Methods("GET")
	api.HandleFunc("/games/crash/bet", placeGameBetHandler).Methods("POST")

	// WebSocket endpoint for live games
	r.HandleFunc("/ws/games/{gameId}", handleWebSocket)

	// CORS middleware
	r.Use(corsMiddleware)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Gateway listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"gateway"}`))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message":"Registration endpoint - implement with KYC validation"}`))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Login endpoint - implement with JWT token generation"}`))
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Get user profile - implement with auth middleware"}`))
}

func getWalletHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"balance":1000.00,"currency":"KES","bonus_balance":50.00}`))
}

func placeBetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message":"Place bet endpoint - implement with PlaceBetUseCase"}`))
}

func getBetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Get bet details"}`))
}

func getBetHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Get bet history"}`))
}

func depositHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"message":"Deposit initiated - check M-Pesa prompt on phone"}`))
}

func withdrawHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"message":"Withdrawal initiated - M-Pesa payout in progress"}`))
}

func mpesaCallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Handle M-Pesa callbacks (STK Push result, B2C result)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
}

func getCurrentGameHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"game_id":"123","round_number":42,"status":"RUNNING","current_multiplier":2.45}`))
}

func getGameHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"games":[{"round":41,"crash_point":3.52},{"round":40,"crash_point":1.23}]}`))
}

func placeGameBetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message":"Game bet placed - connect to WebSocket for live updates"}`))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	log.Println("WebSocket client connected")

	// In production, integrate with the Hub from crash_engine.go
	defer conn.Close()

	// Send welcome message
	conn.WriteJSON(map[string]any{
		"type":    "connected",
		"message": "Connected to crash game",
	})

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
