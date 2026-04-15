package wallet

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct {
	// Add dependencies here (wallet service, repositories, etc.)
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Health check
	r.HandleFunc("/health", h.healthCheckHandler).Methods("GET")

	// Internal API (called by other services)
	api := r.PathPrefix("/internal/v1").Subrouter()
	api.HandleFunc("/wallets/{userId}", h.getWalletHandler).Methods("GET")
	api.HandleFunc("/wallets/{userId}/balance", h.getBalanceHandler).Methods("GET")
	api.HandleFunc("/wallets/{userId}/debit", h.debitHandler).Methods("POST")
	api.HandleFunc("/wallets/{userId}/credit", h.creditHandler).Methods("POST")
	api.HandleFunc("/wallets/{userId}/transactions", h.getTransactionsHandler).Methods("GET")
}

func (h *Handler) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"wallet"}`))
}

func (h *Handler) getWalletHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"user_id":"` + userID + `","balance":1000.00,"currency":"KES","bonus_balance":50.00}`))
}

func (h *Handler) getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"user_id":"` + userID + `","balance":1000.00,"available":950.00}`))
}

func (h *Handler) debitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"user_id":"` + userID + `","status":"debited","amount":100.00}`))
}

func (h *Handler) creditHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"user_id":"` + userID + `","status":"credited","amount":100.00}`))
}

func (h *Handler) getTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"user_id":"` + userID + `","transactions":[{"type":"debit","amount":100.00,"status":"completed"}]}`))
}
