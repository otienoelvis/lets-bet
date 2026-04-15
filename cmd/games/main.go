package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/betting-platform/internal/core/usecase"
	"github.com/betting-platform/internal/games"
)

func main() {
	log.Println("Starting Crash Games Service")

	// Initialize context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize WebSocket hub
	hub := games.NewHub()
	go hub.Run()
	log.Println("WebSocket Hub started")

	// Initialize Provably Fair service
	fairService := usecase.NewProvablyFairService()

	// Initialize repositories (mock for now)
	// In production, connect to PostgreSQL
	gameRepo := &MockGameRepository{}
	betRepo := &MockGameBetRepository{}

	// Initialize Crash Game Engine
	engine := games.NewCrashGameEngine(hub, fairService, gameRepo, betRepo)

	// Start the game loop
	go engine.Start(ctx)

	log.Println("Crash Game Engine running")
	log.Println("Players can connect via WebSocket to receive real-time updates")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Games Service...")
	cancel()

	log.Println("Games Service exited cleanly")
}

type MockGameRepository struct{}

func (r *MockGameRepository) Create(ctx context.Context, game interface{}) error {
	log.Printf("[Mock] Game created: Round %v", game)
	return nil
}

func (r *MockGameRepository) UpdateStatus(ctx context.Context, id interface{}, status interface{}) error {
	log.Printf("[Mock] Game status updated: %v -> %v", id, status)
	return nil
}

type MockGameBetRepository struct{}

func (r *MockGameBetRepository) Create(ctx context.Context, bet interface{}) error {
	log.Printf("[Mock] Game bet created: %v", bet)
	return nil
}

func (r *MockGameBetRepository) GetActiveByGame(ctx context.Context, gameID interface{}) ([]interface{}, error) {
	log.Printf("[Mock] Getting active bets for game: %v", gameID)
	return []interface{}{}, nil
}

func (r *MockGameBetRepository) UpdateCashout(ctx context.Context, id interface{}, cashoutAt interface{}, payout interface{}) error {
	log.Printf("[Mock] Cashout updated: %v at %v = %v", id, cashoutAt, payout)
	return nil
}
