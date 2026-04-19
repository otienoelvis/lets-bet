package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/betting-platform/internal/core/domain"
	"github.com/shopspring/decimal"
)

// SportBetRepository implements sport bet repository using PostgreSQL
type SportBetRepository struct {
	db *sql.DB
}

// NewSportBetRepository creates a new sport bet repository
func NewSportBetRepository(db *sql.DB) *SportBetRepository {
	return &SportBetRepository{db: db}
}

// Create creates a new sport bet
func (r *SportBetRepository) Create(ctx context.Context, bet *domain.SportBet) error {
	query := `
		INSERT INTO sport_bets (
			id, user_id, event_id, market_id, outcome_id, amount, odds,
			currency, status, payout, net_payout, settled_at,
			settlement_reason, placed_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.ExecContext(ctx, query,
		bet.ID, bet.UserID, bet.EventID, bet.MarketID, bet.OutcomeID,
		bet.Amount, bet.Odds, "KES", string(bet.Status), bet.Payout,
		bet.NetPayout, bet.SettledAt, "", bet.PlacedAt, time.Now(),
	)

	return err
}

// GetByID retrieves a sport bet by ID
func (r *SportBetRepository) GetByID(ctx context.Context, id string) (*domain.SportBet, error) {
	query := `
		SELECT id, user_id, event_id, market_id, outcome_id, amount, odds,
			   currency, status, payout, net_payout, settled_at,
			   settlement_reason, placed_at, updated_at
		FROM sport_bets
		WHERE id = $1
	`

	var bet domain.SportBet
	var status, settlementReason string
	var settledAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&bet.ID, &bet.UserID, &bet.EventID, &bet.MarketID, &bet.OutcomeID,
		&bet.Amount, &bet.Odds, &bet.Currency, &status, &bet.Payout,
		&bet.NetPayout, &settledAt, &settlementReason, &bet.PlacedAt, &bet.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	bet.Status = domain.BetStatus(status)
	bet.SettlementReason = settlementReason

	if settledAt.Valid {
		bet.SettledAt = &settledAt.Time
	}

	return &bet, nil
}

// GetByOutcome retrieves sport bets by outcome ID
func (r *SportBetRepository) GetByOutcome(ctx context.Context, outcomeID string) ([]*domain.SportBet, error) {
	query := `
		SELECT id, user_id, event_id, market_id, outcome_id, amount, odds,
			   currency, status, payout, net_payout, settled_at,
			   settlement_reason, placed_at, updated_at
		FROM sport_bets
		WHERE outcome_id = $1
		ORDER BY placed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, outcomeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []*domain.SportBet
	for rows.Next() {
		var bet domain.SportBet
		var status, settlementReason string
		var settledAt sql.NullTime

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.EventID, &bet.MarketID, &bet.OutcomeID,
			&bet.Amount, &bet.Odds, &bet.Currency, &status, &bet.Payout,
			&bet.NetPayout, &settledAt, &settlementReason, &bet.PlacedAt, &bet.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		bet.Status = domain.BetStatus(status)
		bet.SettlementReason = settlementReason

		if settledAt.Valid {
			bet.SettledAt = &settledAt.Time
		}

		bets = append(bets, &bet)
	}

	return bets, nil
}

// GetByUser retrieves sport bets by user ID
func (r *SportBetRepository) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.SportBet, error) {
	query := `
		SELECT id, user_id, event_id, market_id, outcome_id, amount, odds,
			   currency, status, payout, net_payout, settled_at,
			   settlement_reason, placed_at, updated_at
		FROM sport_bets
		WHERE user_id = $1
		ORDER BY placed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []*domain.SportBet
	for rows.Next() {
		var bet domain.SportBet
		var status, settlementReason string
		var settledAt sql.NullTime

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.EventID, &bet.MarketID, &bet.OutcomeID,
			&bet.Amount, &bet.Odds, &bet.Currency, &status, &bet.Payout,
			&bet.NetPayout, &settledAt, &settlementReason, &bet.PlacedAt, &bet.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		bet.Status = domain.BetStatus(status)
		bet.SettlementReason = settlementReason

		if settledAt.Valid {
			bet.SettledAt = &settledAt.Time
		}

		bets = append(bets, &bet)
	}

	return bets, nil
}

// GetByEvent retrieves sport bets by event ID
func (r *SportBetRepository) GetByEvent(ctx context.Context, eventID string) ([]*domain.SportBet, error) {
	query := `
		SELECT id, user_id, event_id, market_id, outcome_id, amount, odds,
			   currency, status, payout, net_payout, settled_at,
			   settlement_reason, placed_at, updated_at
		FROM sport_bets
		WHERE event_id = $1
		ORDER BY placed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []*domain.SportBet
	for rows.Next() {
		var bet domain.SportBet
		var status, settlementReason string
		var settledAt sql.NullTime

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.EventID, &bet.MarketID, &bet.OutcomeID,
			&bet.Amount, &bet.Odds, &bet.Currency, &status, &bet.Payout,
			&bet.NetPayout, &settledAt, &settlementReason, &bet.PlacedAt, &bet.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		bet.Status = domain.BetStatus(status)
		bet.SettlementReason = settlementReason

		if settledAt.Valid {
			bet.SettledAt = &settledAt.Time
		}

		bets = append(bets, &bet)
	}

	return bets, nil
}

// Update updates a sport bet
func (r *SportBetRepository) Update(ctx context.Context, bet *domain.SportBet) error {
	query := `
		UPDATE sport_bets
		SET user_id = $2, event_id = $3, market_id = $4, outcome_id = $5,
			amount = $6, odds = $7, status = $8, payout = $9, net_payout = $10,
			settled_at = $11, settlement_reason = $12, updated_at = $13
		WHERE id = $1
	`

	var settledAt sql.NullTime
	if bet.SettledAt != nil {
		settledAt = sql.NullTime{Time: *bet.SettledAt, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		bet.ID, bet.UserID, bet.EventID, bet.MarketID, bet.OutcomeID,
		bet.Amount, bet.Odds, string(bet.Status), bet.Payout, bet.NetPayout,
		settledAt, bet.SettlementReason, time.Now(),
	)

	return err
}

// UpdateStatus updates sport bet status
func (r *SportBetRepository) UpdateStatus(ctx context.Context, id string, status domain.BetStatus) error {
	query := `
		UPDATE sport_bets
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, string(status), time.Now())
	return err
}

// UpdatePayout updates sport bet payout
func (r *SportBetRepository) UpdatePayout(ctx context.Context, id string, payout, netPayout decimal.Decimal) error {
	query := `
		UPDATE sport_bets
		SET payout = $2, net_payout = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, payout, netPayout, time.Now())
	return err
}

// Delete deletes a sport bet
func (r *SportBetRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sport_bets WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetPendingBets retrieves all pending sport bets
func (r *SportBetRepository) GetPendingBets(ctx context.Context) ([]*domain.SportBet, error) {
	query := `
		SELECT id, user_id, event_id, market_id, outcome_id, amount, odds,
			   currency, status, payout, net_payout, settled_at,
			   settlement_reason, placed_at, updated_at
		FROM sport_bets
		WHERE status = 'PENDING'
		ORDER BY placed_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []*domain.SportBet
	for rows.Next() {
		var bet domain.SportBet
		var status, settlementReason string
		var settledAt sql.NullTime

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.EventID, &bet.MarketID, &bet.OutcomeID,
			&bet.Amount, &bet.Odds, &bet.Currency, &status, &bet.Payout,
			&bet.NetPayout, &settledAt, &settlementReason, &bet.PlacedAt, &bet.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		bet.Status = domain.BetStatus(status)

		if settledAt.Valid {
			bet.SettledAt = &settledAt.Time
		}

		bet.SettlementReason = settlementReason
		bets = append(bets, &bet)
	}

	return bets, nil
}

// GetByUserAndStatus retrieves sport bets by user ID and status
func (r *SportBetRepository) GetByUserAndStatus(ctx context.Context, userID string, status domain.BetStatus) ([]*domain.SportBet, error) {
	query := `
		SELECT id, user_id, event_id, market_id, outcome_id, amount, odds,
			   currency, status, payout, net_payout, settled_at,
			   settlement_reason, placed_at, updated_at
		FROM sport_bets
		WHERE user_id = $1 AND status = $2
		ORDER BY placed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []*domain.SportBet
	for rows.Next() {
		var bet domain.SportBet
		var betStatus, settlementReason string
		var settledAt sql.NullTime

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.EventID, &bet.MarketID, &bet.OutcomeID,
			&bet.Amount, &bet.Odds, &bet.Currency, &betStatus, &bet.Payout,
			&bet.NetPayout, &settledAt, &settlementReason, &bet.PlacedAt, &bet.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		bet.Status = domain.BetStatus(betStatus)

		if settledAt.Valid {
			bet.SettledAt = &settledAt.Time
		}

		bet.SettlementReason = settlementReason
		bets = append(bets, &bet)
	}

	return bets, nil
}
