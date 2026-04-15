package engine

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct {
	// Add dependencies here (engine service, repositories, etc.)
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Health check
	r.HandleFunc("/health", h.healthCheckHandler).Methods("GET")

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Odds routes
	api.HandleFunc("/odds/live", h.getLiveOddsHandler).Methods("GET")
	api.HandleFunc("/odds/calculate", h.calculateOddsHandler).Methods("POST")
	api.HandleFunc("/odds/sync", h.syncOddsHandler).Methods("POST")
}

func (h *Handler) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"engine"}`))
}

func (h *Handler) getLiveOddsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"matches":[{"id":"1","home":"Arsenal","away":"Chelsea","odds":{"home":2.50,"draw":3.20,"away":2.80}}]}`))
}

func (h *Handler) calculateOddsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"total_odds":8.50,"potential_win":850.00}`))
}

func (h *Handler) syncOddsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"message":"Odds sync initiated"}`))
}
