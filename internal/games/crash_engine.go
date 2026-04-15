package games

import (
	"context"
	"log"
	"time"

	"github.com/betting-platform/internal/core/domain"
	"github.com/betting-platform/internal/core/usecase"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CrashGameEngine manages the game loop for Aviator-style crash games
type CrashGameEngine struct {
	hub          *Hub
	fairService  *usecase.ProvablyFairService
	gameRepo     GameRepository
	betRepo      GameBetRepository
	currentGame  *domain.Game
	roundNumber  int64
	tickInterval time.Duration
}

type GameRepository interface {
	Create(ctx context.Context, game *domain.Game) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.GameStatus) error
}

type GameBetRepository interface {
	Create(ctx context.Context, bet *domain.GameBet) error
	GetActiveByGame(ctx context.Context, gameID uuid.UUID) ([]*domain.GameBet, error)
	UpdateCashout(ctx context.Context, id uuid.UUID, cashoutAt decimal.Decimal, payout decimal.Decimal) error
}

func NewCrashGameEngine(
	hub *Hub,
	fairService *usecase.ProvablyFairService,
	gameRepo GameRepository,
	betRepo GameBetRepository,
) *CrashGameEngine {
	return &CrashGameEngine{
		hub:          hub,
		fairService:  fairService,
		gameRepo:     gameRepo,
		betRepo:      betRepo,
		roundNumber:  1,
		tickInterval: 100 * time.Millisecond, // Update every 100ms
	}
}

// Start begins the infinite game loop
func (e *CrashGameEngine) Start(ctx context.Context) {
	log.Println("Crash Game Engine started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Crash Game Engine stopped")
			return
		default:
			e.runRound(ctx)
		}
	}
}

// runRound executes a single game round
func (e *CrashGameEngine) runRound(ctx context.Context) {
	// 1. PREPARATION PHASE
	game := e.prepareGame(ctx)
	e.currentGame = game

	// 2. BETTING PHASE (5 seconds)
	e.bettingPhase(ctx, game)

	// 3. FLIGHT PHASE (crash game)
	e.flightPhase(ctx, game)

	// 4. SETTLEMENT PHASE
	e.settlementPhase(ctx, game)

	// 5. Increment round
	e.roundNumber++

	// Small pause before next round
	time.Sleep(2 * time.Second)
}

func (e *CrashGameEngine) prepareGame(ctx context.Context) *domain.Game {
	// Generate provably fair seeds
	serverSeed := e.fairService.GenerateServerSeed()
	serverSeedHash := e.fairService.HashServerSeed(serverSeed)
	clientSeed := "player_combined_seed_" + time.Now().Format("20060102150405")

	// Calculate crash point (hidden from players)
	crashPoint := e.fairService.CalculateCrashPoint(serverSeed, clientSeed, e.roundNumber)

	game := &domain.Game{
		ID:             uuid.New(),
		GameType:       domain.GameTypeCrash,
		RoundNumber:    e.roundNumber,
		ServerSeed:     serverSeed,
		ServerSeedHash: serverSeedHash,
		ClientSeed:     clientSeed,
		CrashPoint:     crashPoint,
		Status:         domain.GameStatusWaiting,
		StartedAt:      time.Now(),
		CountryCode:    "GLOBAL", // Can be region-specific
		MinBet:         decimal.NewFromInt(10),
		MaxBet:         decimal.NewFromInt(10000),
		MaxMultiplier:  decimal.NewFromInt(100),
	}

	// Persist game
	if err := e.gameRepo.Create(ctx, game); err != nil {
		log.Printf("Error creating game: %v", err)
	}

	log.Printf("Round %d prepared. Crash point: %.2fx (hidden)", e.roundNumber, crashPoint)

	return game
}

func (e *CrashGameEngine) bettingPhase(ctx context.Context, game *domain.Game) {
	log.Printf("Betting phase started for round %d", game.RoundNumber)

	bettingDuration := 5 * time.Second
	startTime := time.Now()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for time.Since(startTime) < bettingDuration {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			remaining := int(bettingDuration.Seconds() - time.Since(startTime).Seconds())

			state := &GameState{
				GameID:            game.ID,
				RoundNumber:       game.RoundNumber,
				Status:            domain.GameStatusWaiting,
				CurrentMultiplier: decimal.NewFromInt(1),
				TimeRemaining:     remaining,
				ActivePlayers:     e.hub.GetActivePlayerCount(game.ID),
			}

			e.hub.BroadcastGameState(state)
		}
	}
}

func (e *CrashGameEngine) flightPhase(ctx context.Context, game *domain.Game) {
	log.Printf("Flight started for round %d. Will crash at %.2fx", game.RoundNumber, game.CrashPoint)

	// Update game status
	game.Status = domain.GameStatusRunning
	e.gameRepo.UpdateStatus(ctx, game.ID, domain.GameStatusRunning)

	// Start from 1.00x
	currentMultiplier := decimal.NewFromFloat(1.00)
	increment := decimal.NewFromFloat(0.01) // Increase by 0.01x every tick

	ticker := time.NewTicker(e.tickInterval)
	defer ticker.Stop()

	for currentMultiplier.LessThan(game.CrashPoint) {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentMultiplier = currentMultiplier.Add(increment)

			// Broadcast current multiplier
			state := &GameState{
				GameID:            game.ID,
				RoundNumber:       game.RoundNumber,
				Status:            domain.GameStatusRunning,
				CurrentMultiplier: currentMultiplier,
				ActivePlayers:     e.hub.GetActivePlayerCount(game.ID),
			}

			e.hub.BroadcastGameState(state)

			// Check if we've reached the crash point
			if currentMultiplier.GreaterThanOrEqual(game.CrashPoint) {
				break
			}
		}
	}

	// CRASH!
	game.Status = domain.GameStatusCrashed
	now := time.Now()
	game.CrashedAt = &now
	e.gameRepo.UpdateStatus(ctx, game.ID, domain.GameStatusCrashed)

	log.Printf("CRASHED at %.2fx", game.CrashPoint)

	// Broadcast crash event
	state := &GameState{
		GameID:            game.ID,
		RoundNumber:       game.RoundNumber,
		Status:            domain.GameStatusCrashed,
		CurrentMultiplier: game.CrashPoint,
		CrashPoint:        &game.CrashPoint,
		ActivePlayers:     e.hub.GetActivePlayerCount(game.ID),
	}

	e.hub.BroadcastGameState(state)
}

func (e *CrashGameEngine) settlementPhase(ctx context.Context, game *domain.Game) {
	log.Printf("Settling round %d", game.RoundNumber)

	// Get all active bets for this game
	bets, err := e.betRepo.GetActiveByGame(ctx, game.ID)
	if err != nil {
		log.Printf("Error getting active bets: %v", err)
		return
	}

	winnersCount := 0
	losersCount := 0

	for _, bet := range bets {
		if bet.CashedOut {
			// Player cashed out before crash - they won
			winnersCount++
			bet.Status = domain.GameBetStatusWon
		} else {
			// Player didn't cash out - they lost
			losersCount++
			bet.Status = domain.GameBetStatusLost
			bet.Payout = decimal.Zero
		}

		// In production, update wallet balances here
	}

	log.Printf("✅ Round %d settled: %d winners, %d losers", game.RoundNumber, winnersCount, losersCount)
}

// HandleCashout processes a player's cashout request
func (e *CrashGameEngine) HandleCashout(ctx context.Context, betID uuid.UUID, currentMultiplier decimal.Decimal) error {
	// This would be called from the WebSocket handler when a player clicks "Cashout"

	// 1. Validate the bet exists and is active
	// 2. Calculate payout: bet.Amount * currentMultiplier
	// 3. Update bet status to CASHED_OUT
	// 4. Credit wallet immediately

	payout := decimal.NewFromInt(100).Mul(currentMultiplier) // Example

	if err := e.betRepo.UpdateCashout(ctx, betID, currentMultiplier, payout); err != nil {
		return err
	}

	log.Printf("Cashout processed: Bet %s at %.2fx = %.2f", betID, currentMultiplier, payout)

	return nil
}
