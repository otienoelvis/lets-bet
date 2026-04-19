package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/betting-platform/internal/jackpots"
	"github.com/shopspring/decimal"
)

// Jackpot types to avoid import cycle
type JackpotType string
type JackpotStatus string
type TicketStatus string

const (
	JackpotTypeDaily       JackpotType = "DAILY"
	JackpotTypeWeekly      JackpotType = "WEEKLY"
	JackpotTypeMonthly     JackpotType = "MONTHLY"
	JackpotTypeProgressive JackpotType = "PROGRESSIVE"
	JackpotTypeMystery     JackpotType = "MYSTERY"
)

const (
	JackpotStatusActive  JackpotStatus = "ACTIVE"
	JackpotStatusPaused  JackpotStatus = "PAUSED"
	JackpotStatusSettled JackpotStatus = "SETTLED"
	JackpotStatusExpired JackpotStatus = "EXPIRED"
)

const (
	TicketStatusActive  TicketStatus = "ACTIVE"
	TicketStatusDrawn   TicketStatus = "DRAWN"
	TicketStatusWon     TicketStatus = "WON"
	TicketStatusExpired TicketStatus = "EXPIRED"
)

// Jackpot represents a jackpot game
type Jackpot struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Type             JackpotType     `json:"type"`
	CurrentAmount    decimal.Decimal `json:"current_amount"`
	SeedAmount       decimal.Decimal `json:"seed_amount"`
	ContributionRate decimal.Decimal `json:"contribution_rate"`
	MinBet           decimal.Decimal `json:"min_bet"`
	MaxBet           decimal.Decimal `json:"max_bet"`
	Status           JackpotStatus   `json:"status"`
	CreatedAt        time.Time       `json:"created_at"`
	LastWonAt        *time.Time      `json:"last_won_at,omitempty"`
	LastWonBy        string          `json:"last_won_by,omitempty"`
	NextDrawAt       *time.Time      `json:"next_draw_at,omitempty"`
}

// JackpotTicket represents a jackpot ticket
type JackpotTicket struct {
	ID        string          `json:"id"`
	JackpotID string          `json:"jackpot_id"`
	UserID    string          `json:"user_id"`
	BetAmount decimal.Decimal `json:"bet_amount"`
	Numbers   []int           `json:"numbers"`
	Status    TicketStatus    `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	DrawnAt   *time.Time      `json:"drawn_at,omitempty"`
	Won       bool            `json:"won"`
	Prize     decimal.Decimal `json:"prize"`
}

// JackpotResult represents a jackpot draw result
type JackpotResult struct {
	JackpotID      string    `json:"jackpot_id"`
	WinningNumbers []int     `json:"winning_numbers"`
	TotalTickets   int       `json:"total_tickets"`
	DrawnAt        time.Time `json:"drawn_at"`
}

// JackpotRepository implements jackpot repository using PostgreSQL
type JackpotRepository struct {
	db *sql.DB
}

// NewJackpotRepository creates a new jackpot repository
func NewJackpotRepository(db *sql.DB) *JackpotRepository {
	return &JackpotRepository{db: db}
}

// Create creates a new jackpot
func (r *JackpotRepository) Create(ctx context.Context, jackpot *Jackpot) error {
	query := `
		INSERT INTO jackpots (
			id, name, type, current_amount, seed_amount, contribution_rate,
			min_bet, max_bet, status, created_at, last_won_at, last_won_by, next_draw_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.ExecContext(ctx, query,
		jackpot.ID, jackpot.Name, string(jackpot.Type), jackpot.CurrentAmount,
		jackpot.SeedAmount, jackpot.ContributionRate, jackpot.MinBet, jackpot.MaxBet,
		string(jackpot.Status), jackpot.CreatedAt, jackpot.LastWonAt,
		jackpot.LastWonBy, jackpot.NextDrawAt,
	)

	return err
}

// GetByID retrieves a jackpot by ID
func (r *JackpotRepository) GetByID(ctx context.Context, id string) (*Jackpot, error) {
	query := `
		SELECT id, name, type, current_amount, seed_amount, contribution_rate,
			   min_bet, max_bet, status, created_at, last_won_at, last_won_by, next_draw_at
		FROM jackpots
		WHERE id = $1
	`

	var jackpot Jackpot
	var jackpotType, status string
	var lastWonAt, nextDrawAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&jackpot.ID, &jackpot.Name, &jackpotType, &jackpot.CurrentAmount,
		&jackpot.SeedAmount, &jackpot.ContributionRate, &jackpot.MinBet, &jackpot.MaxBet,
		&status, &jackpot.CreatedAt, &lastWonAt, &jackpot.LastWonBy, &nextDrawAt,
	)

	if err != nil {
		return nil, err
	}

	jackpot.Type = JackpotType(jackpotType)
	jackpot.Status = JackpotStatus(status)

	if lastWonAt.Valid {
		jackpot.LastWonAt = &lastWonAt.Time
	}

	if nextDrawAt.Valid {
		jackpot.NextDrawAt = &nextDrawAt.Time
	}

	return &jackpot, nil
}

// Update updates a jackpot
func (r *JackpotRepository) Update(ctx context.Context, jackpot *Jackpot) error {
	query := `
		UPDATE jackpots
		SET current_amount = $2, status = $3, last_won_at = $4, last_won_by = $5, next_draw_at = $6
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		jackpot.ID, jackpot.CurrentAmount, string(jackpot.Status),
		jackpot.LastWonAt, jackpot.LastWonBy, jackpot.NextDrawAt,
	)

	return err
}

// GetActive retrieves all active jackpots
func (r *JackpotRepository) GetActive(ctx context.Context) ([]*Jackpot, error) {
	query := `
		SELECT id, name, type, current_amount, seed_amount, contribution_rate,
			   min_bet, max_bet, status, created_at, last_won_at, last_won_by, next_draw_at
		FROM jackpots
		WHERE status = 'ACTIVE'
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jackpots []*Jackpot
	for rows.Next() {
		var jackpot Jackpot
		var jackpotType, status string
		var lastWonAt, nextDrawAt sql.NullTime

		err := rows.Scan(
			&jackpot.ID, &jackpot.Name, &jackpotType, &jackpot.CurrentAmount,
			&jackpot.SeedAmount, &jackpot.ContributionRate, &jackpot.MinBet, &jackpot.MaxBet,
			&status, &jackpot.CreatedAt, &lastWonAt, &jackpot.LastWonBy, &nextDrawAt,
		)

		if err != nil {
			return nil, err
		}

		jackpot.Type = JackpotType(jackpotType)
		jackpot.Status = JackpotStatus(status)

		if lastWonAt.Valid {
			jackpot.LastWonAt = &lastWonAt.Time
		}

		if nextDrawAt.Valid {
			jackpot.NextDrawAt = &nextDrawAt.Time
		}

		jackpots = append(jackpots, &jackpot)
	}

	return jackpots, nil
}

// CreateTicket creates a new jackpot ticket
func (r *JackpotRepository) CreateTicket(ctx context.Context, ticket *JackpotTicket) error {
	query := `
		INSERT INTO jackpot_tickets (
			id, jackpot_id, user_id, bet_amount, numbers, status, created_at, drawn_at, won, prize
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		ticket.ID, ticket.JackpotID, ticket.UserID, ticket.BetAmount,
		ticket.Numbers, string(ticket.Status), ticket.CreatedAt,
		ticket.DrawnAt, ticket.Won, ticket.Prize,
	)

	return err
}

// DeleteTicket deletes a jackpot ticket
func (r *JackpotRepository) DeleteTicket(ctx context.Context, ticketID string) error {
	query := `DELETE FROM jackpot_tickets WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, ticketID)
	return err
}

// GetActiveTickets retrieves all active tickets for a jackpot
func (r *JackpotRepository) GetActiveTickets(ctx context.Context, jackpotID string) ([]*jackpots.JackpotTicket, error) {
	query := `
		SELECT id, jackpot_id, user_id, bet_amount, numbers, status, created_at, drawn_at, won, prize
		FROM jackpot_tickets
		WHERE jackpot_id = $1 AND status = 'ACTIVE'
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, jackpotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*jackpots.JackpotTicket
	for rows.Next() {
		var ticket jackpots.JackpotTicket
		var status string
		var drawnAt sql.NullTime

		err := rows.Scan(
			&ticket.ID, &ticket.JackpotID, &ticket.UserID, &ticket.BetAmount,
			&ticket.Numbers, &status, &ticket.CreatedAt, &drawnAt,
			&ticket.Won, &ticket.Prize,
		)

		if err != nil {
			return nil, err
		}

		ticket.Status = jackpots.TicketStatus(status)

		if drawnAt.Valid {
			ticket.DrawnAt = &drawnAt.Time
		}

		tickets = append(tickets, &ticket)
	}

	return tickets, nil
}

// GetUserTickets retrieves all tickets for a user
func (r *JackpotRepository) GetUserTickets(ctx context.Context, userID string) ([]*jackpots.JackpotTicket, error) {
	query := `
		SELECT id, jackpot_id, user_id, bet_amount, numbers, status, created_at, drawn_at, won, prize
		FROM jackpot_tickets
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*jackpots.JackpotTicket
	for rows.Next() {
		var ticket jackpots.JackpotTicket
		var status string
		var drawnAt sql.NullTime

		err := rows.Scan(
			&ticket.ID, &ticket.JackpotID, &ticket.UserID, &ticket.BetAmount,
			&ticket.Numbers, &status, &ticket.CreatedAt, &drawnAt,
			&ticket.Won, &ticket.Prize,
		)

		if err != nil {
			return nil, err
		}

		ticket.Status = jackpots.TicketStatus(status)

		if drawnAt.Valid {
			ticket.DrawnAt = &drawnAt.Time
		}

		tickets = append(tickets, &ticket)
	}

	return tickets, nil
}

// UpdateTicketStatus updates the status of a jackpot ticket
func (r *JackpotRepository) UpdateTicketStatus(ctx context.Context, ticketID string, status jackpots.TicketStatus, prize decimal.Decimal) error {
	query := `
		UPDATE jackpot_tickets
		SET status = $2, drawn_at = $3, won = $4, prize = $5
		WHERE id = $1
	`

	now := time.Now()
	won := status == jackpots.TicketStatusWon

	_, err := r.db.ExecContext(ctx, query, ticketID, string(status), now, won, prize)
	return err
}

// GetJackpotHistory retrieves the history of jackpot draws
func (r *JackpotRepository) GetJackpotHistory(ctx context.Context, jackpotID string, limit int) ([]*jackpots.JackpotResult, error) {
	query := `
		SELECT id, jackpot_id, winning_numbers, total_tickets, drawn_at
		FROM jackpot_results
		WHERE jackpot_id = $1
		ORDER BY drawn_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, jackpotID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*jackpots.JackpotResult
	for rows.Next() {
		var result jackpots.JackpotResult
		err := rows.Scan(
			&result.JackpotID, &result.JackpotID, &result.WinningNumbers,
			&result.TotalTickets, &result.DrawnAt,
		)

		if err != nil {
			return nil, err
		}

		results = append(results, &result)
	}

	return results, nil
}

// SaveResult saves a jackpot draw result
func (r *JackpotRepository) SaveResult(ctx context.Context, result *jackpots.JackpotResult) error {
	query := `
		INSERT INTO jackpot_results (
			id, jackpot_id, winning_numbers, total_tickets, drawn_at
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		result.JackpotID, result.JackpotID, result.WinningNumbers,
		result.TotalTickets, result.DrawnAt,
	)

	return err
}
