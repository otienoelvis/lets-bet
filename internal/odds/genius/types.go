// Package genius provides Genius Sports odds feed integration
package genius

import (
	"context"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// GeniusConfig provides configuration for Genius Sports client
type GeniusConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	Environment string        `json:"environment"` // "trial", "production"
	Timeout     time.Duration `json:"timeout"`
	RateLimit   int           `json:"rate_limit"` // requests per minute
}

// DefaultGeniusConfig returns default configuration
func DefaultGeniusConfig() *GeniusConfig {
	return &GeniusConfig{
		Environment: "trial",
		BaseURL:     "https://api.geniussports.com",
		Timeout:     30 * time.Second,
		RateLimit:   60, // 60 requests per minute
	}
}

// GeniusClient provides Genius Sports odds feed integration
type GeniusClient struct {
	config      *GeniusConfig
	httpClient  *http.Client
	rateLimiter *RateLimiter
}

// RateLimiter implements rate limiting for API requests
type RateLimiter struct {
	tokens     int
	maxTokens  int
	interval   time.Duration
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		maxTokens:  maxTokens,
		tokens:     maxTokens,
		interval:   interval,
		lastRefill: time.Now(),
	}
}

// Wait waits until a token is available
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.tryConsume() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.interval / time.Duration(r.maxTokens)):
			// Wait for a short period before retrying
		}
	}
}

// tryConsume tries to consume a token
func (r *RateLimiter) tryConsume() bool {
	now := time.Now()
	// Refill tokens based on time elapsed
	elapsed := now.Sub(r.lastRefill)
	tokensToAdd := int(elapsed / r.interval)
	if tokensToAdd > 0 {
		r.tokens = min(r.maxTokens, r.tokens+tokensToAdd)
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

// OddsRequest represents an odds request
type OddsRequest struct {
	SportID  string    `json:"sport_id"`
	LeagueID string    `json:"league_id,omitempty"`
	MatchID  string    `json:"match_id,omitempty"`
	Market   string    `json:"market,omitempty"`
	From     time.Time `json:"from,omitempty"`
	To       time.Time `json:"to,omitempty"`
	Live     bool      `json:"live,omitempty"`
}

// OddsResponse represents an odds response
type OddsResponse struct {
	Success bool       `json:"success"`
	Data    []OddsData `json:"data"`
	Message string     `json:"message,omitempty"`
}

// OddsData contains odds information
type OddsData struct {
	MatchID   string       `json:"match_id"`
	SportID   string       `json:"sport_id"`
	LeagueID  string       `json:"league_id"`
	HomeTeam  string       `json:"home_team"`
	AwayTeam  string       `json:"away_team"`
	StartTime time.Time    `json:"start_time"`
	IsLive    bool         `json:"is_live"`
	Score     *MatchScore  `json:"score,omitempty"`
	Markets   []MarketData `json:"markets"`
}

// MatchScore represents match score
type MatchScore struct {
	HomeScore int `json:"home_score"`
	AwayScore int `json:"away_score"`
	Minute    int `json:"minute,omitempty"`
}

// MarketData represents betting market data
type MarketData struct {
	MarketID   string        `json:"market_id"`
	MarketName string        `json:"market_name"`
	Outcomes   []OutcomeData `json:"outcomes"`
}

// OutcomeData represents betting outcome data
type OutcomeData struct {
	OutcomeID   string          `json:"outcome_id"`
	OutcomeName string          `json:"outcome_name"`
	Odds        decimal.Decimal `json:"odds"`
	IsActive    bool            `json:"is_active"`
}

// MatchRequest represents a match request
type MatchRequest struct {
	SportID  string    `json:"sport_id"`
	LeagueID string    `json:"league_id,omitempty"`
	From     time.Time `json:"from,omitempty"`
	To       time.Time `json:"to,omitempty"`
	Live     bool      `json:"live,omitempty"`
}

// MatchResponse represents a match response
type MatchResponse struct {
	Success bool        `json:"success"`
	Data    []MatchData `json:"data"`
	Message string      `json:"message,omitempty"`
}

// MatchData contains match information
type MatchData struct {
	MatchID   string      `json:"match_id"`
	SportID   string      `json:"sport_id"`
	LeagueID  string      `json:"league_id"`
	HomeTeam  string      `json:"home_team"`
	AwayTeam  string      `json:"away_team"`
	StartTime time.Time   `json:"start_time"`
	IsLive    bool        `json:"is_live"`
	Status    string      `json:"status"`
	Score     *MatchScore `json:"score,omitempty"`
}

// SportRequest represents a sport request
type SportRequest struct {
	ActiveOnly bool `json:"active_only,omitempty"`
}

// SportResponse represents a sport response
type SportResponse struct {
	Success bool        `json:"success"`
	Data    []SportData `json:"data"`
	Message string      `json:"message,omitempty"`
}

// SportData contains sport information
type SportData struct {
	SportID   string `json:"sport_id"`
	SportName string `json:"sport_name"`
	IsActive  bool   `json:"is_active"`
}

// LeagueRequest represents a league request
type LeagueRequest struct {
	SportID    string `json:"sport_id,omitempty"`
	ActiveOnly bool   `json:"active_only,omitempty"`
}

// LeagueResponse represents a league response
type LeagueResponse struct {
	Success bool         `json:"success"`
	Data    []LeagueData `json:"data"`
	Message string       `json:"message,omitempty"`
}

// LeagueData contains league information
type LeagueData struct {
	LeagueID   string `json:"league_id"`
	LeagueName string `json:"league_name"`
	SportID    string `json:"sport_id"`
	Country    string `json:"country"`
	IsActive   bool   `json:"is_active"`
}

// GeniusMetrics represents client metrics
type GeniusMetrics struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	LastRequestTime     time.Time     `json:"last_request_time"`
	AverageResponseTime time.Duration `json:"average_response_time"`
}

// GeniusError represents an API error
type GeniusError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
