package sportradar

import (
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// SportradarConfig provides configuration for Sportradar client
type SportradarConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	Environment string        `json:"environment"` // "trial", "production"
	Timeout     time.Duration `json:"timeout"`
	RateLimit   int           `json:"rate_limit"` // requests per minute
}

// SportradarClient provides Sportradar odds feed integration
type SportradarClient struct {
	config      *SportradarConfig
	httpClient  *http.Client
	rateLimiter *RateLimiter
}

type Sport struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Tournament represents a tournament in Sportradar
type Tournament struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Sport    Sport  `json:"sport"`
	Category string `json:"category"`
}

// Match represents a match/event in Sportradar
type Match struct {
	ID          string     `json:"id"`
	Sport       Sport      `json:"sport"`
	Tournament  Tournament `json:"tournament"`
	HomeTeam    Team       `json:"home_team"`
	AwayTeam    Team       `json:"away_team"`
	Status      string     `json:"status"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Score       Score      `json:"score"`
	Odds        []Odds     `json:"odds"`
}

// Team represents a team in Sportradar
type Team struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

// Score represents match score
type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

// Odds represents betting odds for a match
type Odds struct {
	ID          string          `json:"id"`
	Market      string          `json:"market"` // "moneyline", "spread", "total"
	Outcome     string          `json:"outcome"`
	Price       decimal.Decimal `json:"price"`
	UpdatedAt   time.Time       `json:"updated_at"`
	IsAvailable bool            `json:"is_available"`
}

// SportradarOddsResponse represents Sportradar API response
type SportradarOddsResponse struct {
	Success bool    `json:"success"`
	Data    []Match `json:"data"`
	Error   string  `json:"error,omitempty"`
}
