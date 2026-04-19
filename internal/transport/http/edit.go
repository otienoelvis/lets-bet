package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/betting-platform/internal/sports/edit"
)

// EditHandler handles bet editing HTTP requests
type EditHandler struct {
	editService *edit.EditBetService
}

// NewEditHandler creates a new edit handler
func NewEditHandler(editService *edit.EditBetService) *EditHandler {
	return &EditHandler{
		editService: editService,
	}
}

// RegisterRoutes registers edit betting routes
func (h *EditHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/edit/bets", h.GetEditableBets)
	mux.HandleFunc("/api/edit/bets/", h.EditBet)
}

// EditBetRequest represents the HTTP request for editing a bet
type EditBetRequest struct {
	BetID        string `json:"bet_id"`
	NewAmount    string `json:"new_amount"`
	NewOdds      string `json:"new_odds"`
	NewOutcomeID string `json:"new_outcome_id,omitempty"`
	Reason       string `json:"reason"`
}

// EditBet handles bet editing requests
func (h *EditHandler) EditBet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract bet ID from URL path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	var betID string
	for i, part := range parts {
		if part == "bets" && i+1 < len(parts) {
			betID = parts[i+1]
			break
		}
	}

	if betID == "" {
		WriteError(w, nil, "Bet ID is required", http.StatusBadRequest)
		return
	}

	var req EditBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, err, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse amounts
	newAmount, err := decimal.NewFromString(req.NewAmount)
	if err != nil {
		WriteError(w, err, "Invalid new amount", http.StatusBadRequest)
		return
	}

	newOdds, err := decimal.NewFromString(req.NewOdds)
	if err != nil {
		WriteError(w, err, "Invalid new odds", http.StatusBadRequest)
		return
	}

	// Validate amounts
	if newAmount.LessThanOrEqual(decimal.Zero) {
		WriteError(w, nil, "New amount must be greater than zero", http.StatusBadRequest)
		return
	}

	if newOdds.LessThanOrEqual(decimal.Zero) {
		WriteError(w, nil, "New odds must be greater than zero", http.StatusBadRequest)
		return
	}

	// Get user ID from context
	userID := getUserID(ctx)

	// Create edit request
	editReq := &edit.EditBetRequest{
		BetID:        betID,
		UserID:       userID,
		NewAmount:    newAmount,
		NewOdds:      newOdds,
		NewOutcomeID: req.NewOutcomeID,
		Reason:       req.Reason,
	}

	// Validate reason
	if editReq.Reason == "" {
		editReq.Reason = "User requested edit"
	}

	// Process the edit
	response, err := h.editService.EditBet(ctx, editReq)
	if err != nil {
		WriteError(w, err, "Failed to edit bet", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, response, http.StatusOK)
}

// GetEditableBets returns bets that can be edited for the current user
func (h *EditHandler) GetEditableBets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID := getUserID(ctx)

	// Get editable bets
	bets, err := h.editService.GetEditableBets(ctx, userID)
	if err != nil {
		WriteError(w, err, "Failed to get editable bets", http.StatusInternalServerError)
		return
	}

	// Create response
	response := map[string]any{
		"bets":                    bets,
		"count":                   len(bets),
		"editable_window_minutes": 5, // Configurable edit window
		"rules": []string{
			"Only pending bets can be edited",
			"Bets can only be edited within 5 minutes of placement",
			"Bets cannot be edited after the match starts",
			"Amount can be increased or decreased",
			"Odds and outcome can be changed",
		},
	}

	WriteJSON(w, response, http.StatusOK)
}

// GetEditHistory retrieves edit history for a bet (placeholder for future implementation)
func (h *EditHandler) GetEditHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract bet ID from URL path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	var betID string
	for i, part := range parts {
		if part == "bets" && i+1 < len(parts) {
			betID = parts[i+1]
			break
		}
	}

	if betID == "" {
		WriteError(w, nil, "Bet ID is required", http.StatusBadRequest)
		return
	}

	// Get user ID from context
	userID := getUserID(ctx)

	// TODO: Implement edit history tracking
	// For now, return empty history
	response := map[string]any{
		"bet_id":       betID,
		"user_id":      userID,
		"edit_history": []any{},
		"message":      "Edit history tracking not yet implemented",
	}

	WriteJSON(w, response, http.StatusOK)
}
