package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/betting-platform/internal/admin"
)

// AdminHandler handles admin dashboard HTTP requests
type AdminHandler struct {
	adminService *admin.AdminService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(adminService *admin.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// GetDashboardData returns comprehensive dashboard data
func (h *AdminHandler) GetDashboardData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dashboardData, err := h.adminService.GetDashboardData(ctx)
	if err != nil {
		WriteError(w, err, "Failed to get dashboard data", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, dashboardData, http.StatusOK)
}

// GetUserManagementData returns user management data with pagination
func (h *AdminHandler) GetUserManagementData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	offset := 0 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	users, err := h.adminService.GetUserManagementData(ctx, limit, offset)
	if err != nil {
		WriteError(w, err, "Failed to get user management data", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, users, http.StatusOK)
}

// SuspendUser suspends a user account
func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from URL path
	userID := extractUserIDFromPath(r.URL.Path)
	if userID == "" {
		WriteError(w, fmt.Errorf("user ID is required"), "User ID is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, err, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.adminService.SuspendUser(ctx, userID, req.Reason)
	if err != nil {
		WriteError(w, err, "Failed to suspend user", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, map[string]string{"message": "User suspended successfully"}, http.StatusOK)
}

// UnsuspendUser unsuspends a user account
func (h *AdminHandler) UnsuspendUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from URL path
	userID := extractUserIDFromPath(r.URL.Path)
	if userID == "" {
		WriteError(w, fmt.Errorf("user ID is required"), "User ID is required", http.StatusBadRequest)
		return
	}

	err := h.adminService.UnsuspendUser(ctx, userID)
	if err != nil {
		WriteError(w, err, "Failed to unsuspend user", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, map[string]string{"message": "User unsuspended successfully"}, http.StatusOK)
}

// GetRevenueAnalytics returns detailed revenue analytics
func (h *AdminHandler) GetRevenueAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d" // default to 30 days
	}

	// Get dashboard data and extract revenue metrics
	dashboardData, err := h.adminService.GetDashboardData(ctx)
	if err != nil {
		WriteError(w, err, "Failed to get revenue analytics", http.StatusInternalServerError)
		return
	}

	// Filter revenue data based on period
	revenueData := filterRevenueByPeriod(dashboardData.RevenueMetrics, period)

	WriteJSON(w, revenueData, http.StatusOK)
}

// GetBettingAnalytics returns detailed betting analytics
func (h *AdminHandler) GetBettingAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	sport := r.URL.Query().Get("sport")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h" // default to 24 hours
	}

	// Get dashboard data and extract betting metrics
	dashboardData, err := h.adminService.GetDashboardData(ctx)
	if err != nil {
		WriteError(w, err, "Failed to get betting analytics", http.StatusInternalServerError)
		return
	}

	// Filter betting data based on sport and period
	bettingData := filterBettingData(dashboardData.BettingMetrics, sport, period)

	WriteJSON(w, bettingData, http.StatusOK)
}

// GetFinancialAnalytics returns detailed financial analytics
func (h *AdminHandler) GetFinancialAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = "KES" // default currency
	}

	// Get dashboard data and extract financial metrics
	dashboardData, err := h.adminService.GetDashboardData(ctx)
	if err != nil {
		WriteError(w, err, "Failed to get financial analytics", http.StatusInternalServerError)
		return
	}

	// Filter financial data based on currency
	financialData := filterFinancialData(dashboardData.FinancialStats, currency)

	WriteJSON(w, financialData, http.StatusOK)
}

// GetSystemHealth returns system health status
func (h *AdminHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get dashboard data and extract system health
	dashboardData, err := h.adminService.GetDashboardData(ctx)
	if err != nil {
		WriteError(w, err, "Failed to get system health", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, dashboardData.SystemHealth, http.StatusOK)
}

// GetActivityLog returns recent activity log
func (h *AdminHandler) GetActivityLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	typeFilter := r.URL.Query().Get("type")
	statusFilter := r.URL.Query().Get("status")

	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get dashboard data and extract recent activity
	dashboardData, err := h.adminService.GetDashboardData(ctx)
	if err != nil {
		WriteError(w, err, "Failed to get activity log", http.StatusInternalServerError)
		return
	}

	// Filter activity based on parameters
	activityData := filterActivityData(dashboardData.RecentActivity, limit, typeFilter, statusFilter)

	WriteJSON(w, activityData, http.StatusOK)
}

// GetTopUsers returns top users by various metrics
func (h *AdminHandler) GetTopUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	metric := r.URL.Query().Get("metric")
	limitStr := r.URL.Query().Get("limit")

	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get dashboard data and extract top users
	dashboardData, err := h.adminService.GetDashboardData(ctx)
	if err != nil {
		WriteError(w, err, "Failed to get top users", http.StatusInternalServerError)
		return
	}

	// Sort users based on metric
	topUsers := sortUsersByMetric(dashboardData.TopUsers, metric, limit)

	WriteJSON(w, topUsers, http.StatusOK)
}

// Helper functions

// extractUserIDFromPath extracts user ID from URL path
func extractUserIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "users" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// filterRevenueByPeriod filters revenue data based on period
func filterRevenueByPeriod(revenue admin.RevenueMetrics, period string) admin.RevenueMetrics {
	// In a real implementation, this would filter based on the period parameter
	// For now, return the original data
	return revenue
}

// filterBettingData filters betting data based on sport and period
func filterBettingData(betting admin.BettingMetrics, sport, period string) admin.BettingMetrics {
	// In a real implementation, this would filter based on sport and period
	// For now, return the original data
	return betting
}

// filterFinancialData filters financial data based on currency
func filterFinancialData(financial admin.FinancialStats, currency string) admin.FinancialStats {
	// In a real implementation, this would filter based on currency
	// For now, return the original data
	return financial
}

// filterActivityData filters activity data based on parameters
func filterActivityData(activities []admin.ActivityItem, limit int, typeFilter, statusFilter string) []admin.ActivityItem {
	var filtered []admin.ActivityItem

	for _, activity := range activities {
		// Apply type filter
		if typeFilter != "" && activity.Type != typeFilter {
			continue
		}

		// Apply status filter
		if statusFilter != "" && activity.Status != statusFilter {
			continue
		}

		filtered = append(filtered, activity)

		// Apply limit
		if len(filtered) >= limit {
			break
		}
	}

	return filtered
}

// sortUsersByMetric sorts users based on the specified metric
func sortUsersByMetric(users []admin.UserStats, metric string, limit int) []admin.UserStats {
	// In a real implementation, this would sort users based on the metric
	// For now, return the top N users
	if len(users) > limit {
		return users[:limit]
	}
	return users
}

// RegisterAdminRoutes registers admin routes
func RegisterAdminRoutes(mux *http.ServeMux, adminHandler *AdminHandler) {
	// Dashboard routes
	mux.HandleFunc("/GET/admin/dashboard", adminHandler.GetDashboardData)
	mux.HandleFunc("/GET/admin/dashboard/overview", adminHandler.GetDashboardData)

	// User management routes
	mux.HandleFunc("/GET/admin/users", adminHandler.GetUserManagementData)
	mux.HandleFunc("/POST/admin/users/{userID}/suspend", adminHandler.SuspendUser)
	mux.HandleFunc("/POST/admin/users/{userID}/unsuspend", adminHandler.UnsuspendUser)

	// Analytics routes
	mux.HandleFunc("/GET/admin/analytics/revenue", adminHandler.GetRevenueAnalytics)
	mux.HandleFunc("/GET/admin/analytics/betting", adminHandler.GetBettingAnalytics)
	mux.HandleFunc("/GET/admin/analytics/financial", adminHandler.GetFinancialAnalytics)

	// System routes
	mux.HandleFunc("/GET/admin/system/health", adminHandler.GetSystemHealth)
	mux.HandleFunc("/GET/admin/system/activity", adminHandler.GetActivityLog)

	// Reports routes
	mux.HandleFunc("/GET/admin/reports/users/top", adminHandler.GetTopUsers)
}
