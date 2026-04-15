package mock

import (
	"context"
	"log"

	"github.com/betting-platform/internal/core/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MockGameBetRepository implements a mock game bet repository for testing
type MockGameBetRepository struct{}

func NewMockGameBetRepository() *MockGameBetRepository {
	return &MockGameBetRepository{}
}

func (r *MockGameBetRepository) Create(ctx context.Context, bet *domain.GameBet) error {
	log.Printf("[Mock] Game bet created: %v", bet.ID)
	return nil
}

func (r *MockGameBetRepository) GetActiveByGame(ctx context.Context, gameID uuid.UUID) ([]*domain.GameBet, error) {
	log.Printf("[Mock] Getting active bets for game: %v", gameID)
	return []*domain.GameBet{}, nil
}

func (r *MockGameBetRepository) UpdateCashout(ctx context.Context, id uuid.UUID, cashoutAt decimal.Decimal, payout decimal.Decimal) error {
	log.Printf("[Mock] Cashout updated: %v at %v = %v", id, cashoutAt, payout)
	return nil
}
