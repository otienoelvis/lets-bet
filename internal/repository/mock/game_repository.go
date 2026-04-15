package mock

import (
	"context"
	"log"

	"github.com/betting-platform/internal/core/domain"
	"github.com/google/uuid"
)

// MockGameRepository implements a mock game repository for testing
type MockGameRepository struct{}

func NewMockGameRepository() *MockGameRepository {
	return &MockGameRepository{}
}

func (r *MockGameRepository) Create(ctx context.Context, game *domain.Game) error {
	log.Printf("[Mock] Game created: Round %v", game.RoundNumber)
	return nil
}

func (r *MockGameRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.GameStatus) error {
	log.Printf("[Mock] Game status updated: %v -> %v", id, status)
	return nil
}
