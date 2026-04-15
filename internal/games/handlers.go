package games

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct {
	// Add dependencies here (games service, repositories, etc.)
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Health check
	r.HandleFunc("/health", h.healthCheckHandler).Methods("GET")

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Games routes
	api.HandleFunc("/games/crash/current", h.getCurrentGameHandler).Methods("GET")
	api.HandleFunc("/games/crash/history", h.getGameHistoryHandler).Methods("GET")
	api.HandleFunc("/games/crash/bet", h.placeGameBetHandler).Methods("POST")
	api.HandleFunc("/games/crash/cashout", h.cashoutHandler).Methods("POST")

	// WebSocket endpoint for live games
	r.HandleFunc("/ws/games/{gameId}", h.handleWebSocket)
}

func (h *Handler) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"games"}`))
}

func (h *Handler) getCurrentGameHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"game_id":"123","round_number":42,"status":"RUNNING","current_multiplier":2.45}`))
}

func (h *Handler) getGameHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"games":[{"round":41,"crash_point":3.52},{"round":40,"crash_point":1.23}]}`))
}

func (h *Handler) placeGameBetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message":"Game bet placed - connect to WebSocket for live updates"}`))
}

func (h *Handler) cashoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Cashout processed","payout":245.00}`))
}

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// This will be implemented with the actual WebSocket hub
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"WebSocket endpoint - integrate with crash engine"}`))
}
