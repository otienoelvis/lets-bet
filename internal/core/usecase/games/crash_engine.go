package games

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/betting-platform/internal/core/domain"
	"github.com/betting-platform/internal/core/usecase"
	"github.com/betting-platform/internal/core/usecase/tax"
	"github.com/betting-platform/internal/core/usecase/wallet"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CrashGameEngine manages the game loop for Aviator-style crash games
type CrashGameEngine struct {
	hub           WebSocketHub
	fairService   *usecase.ProvablyFairService
	gameRepo      GameRepository
	betRepo       GameBetRepository
	walletService *wallet.Service
	taxEngine     *tax.Engine
	currentGame   *domain.Game
	roundNumber   int64
	tickInterval  time.Duration
}

type GameRepository interface {
	Create(ctx context.Context, game *domain.Game) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Game, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.GameStatus) error
}

type GameBetRepository interface {
	Create(ctx context.Context, bet *domain.GameBet) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.GameBet, error)
	GetActiveByGame(ctx context.Context, gameID uuid.UUID) ([]*domain.GameBet, error)
	UpdateCashout(ctx context.Context, id uuid.UUID, cashoutAt decimal.Decimal, payout decimal.Decimal) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.GameBetStatus) error
}

type WebSocketHub interface {
	BroadcastGameState(state any)
	GetActivePlayerCount(gameID uuid.UUID) int
}

func NewCrashGameEngine(
	hub WebSocketHub,
	fairService *usecase.ProvablyFairService,
	gameRepo GameRepository,
	betRepo GameBetRepository,
	walletService *wallet.Service,
	taxEngine *tax.Engine,
) *CrashGameEngine {
	return &CrashGameEngine{
		hub:           hub,
		fairService:   fairService,
		gameRepo:      gameRepo,
		betRepo:       betRepo,
		walletService: walletService,
		taxEngine:     taxEngine,
		roundNumber:   1,
		tickInterval:  100 * time.Millisecond, // Update every 100ms
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

func (e *CrashGameEngine) runRound(ctx context.Context) {
	// 1. Prepare new round
	game := e.prepareNewRound(ctx)
	if game == nil {
		return
	}

	// 2. Betting phase
	e.bettingPhase(ctx, game)

	// 3. Flight phase
	e.flightPhase(ctx, game)

	// 4. Settlement phase
	e.settlementPhase(ctx, game)

	// Increment round number
	e.roundNumber++
}

func (e *CrashGameEngine) prepareNewRound(ctx context.Context) *domain.Game {
	// Generate provably fair crash point
	serverSeed, err := e.fairService.GenerateServerSeed()
	if err != nil {
		log.Printf("error generating server seed: %v", err)
		return nil
	}
	clientSeed := "default-client-seed" // In production, get from user
	crashPoint := e.fairService.CalculateCrashPoint(serverSeed, clientSeed, e.roundNumber)

	game := &domain.Game{
		ID:             uuid.New(),
		GameType:       domain.GameTypeCrash,
		RoundNumber:    e.roundNumber,
		ServerSeed:     serverSeed,
		ServerSeedHash: e.fairService.HashServerSeed(serverSeed),
		ClientSeed:     clientSeed,
		CrashPoint:     crashPoint,
		Status:         domain.GameStatusWaiting,
		StartedAt:      time.Now(),
		CountryCode:    "KE", // Default to Kenya
		MinBet:         decimal.NewFromInt(10),
		MaxBet:         decimal.NewFromInt(10000),
		MaxMultiplier:  decimal.NewFromInt(100),
	}

	// Persist game
	if err := e.gameRepo.Create(ctx, game); err != nil {
		log.Printf("Error creating game: %v", err)
	}

	log.Printf("Round %d prepared. Crash point: %s (hidden)", e.roundNumber, crashPoint.String())

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
			remaining := bettingDuration - time.Since(startTime)

			// Broadcast betting state
			state := &GameState{
				GameID:        game.ID,
				RoundNumber:   game.RoundNumber,
				Status:        domain.GameStatusWaiting,
				TimeRemaining: remaining,
				ActivePlayers: e.hub.GetActivePlayerCount(game.ID),
			}

			e.hub.BroadcastGameState(state)
		}
	}
}

func (e *CrashGameEngine) flightPhase(ctx context.Context, game *domain.Game) {
	log.Printf("Flight started for round %d. Will crash at %s", game.RoundNumber, game.CrashPoint.String())

	// Update game status
	game.Status = domain.GameStatusRunning
	e.gameRepo.UpdateStatus(ctx, game.ID, domain.GameStatusRunning)

	// Start from 1.00x
	currentMultiplier := decimal.NewFromFloat(1.00)
	increment := decimal.NewFromFloat(0.01) // Increase by 0.01x every tick

	ticker := time.NewTicker(e.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if we should crash
			if currentMultiplier.GreaterThanOrEqual(game.CrashPoint) {
				goto CRASH
			}

			// Increase multiplier
			currentMultiplier = currentMultiplier.Add(increment)

			// Broadcast current state
			state := &GameState{
				GameID:            game.ID,
				RoundNumber:       game.RoundNumber,
				Status:            domain.GameStatusRunning,
				CurrentMultiplier: currentMultiplier,
				ActivePlayers:     e.hub.GetActivePlayerCount(game.ID),
			}

			e.hub.BroadcastGameState(state)
		}
	}

CRASH:
	// CRASH!
	game.Status = domain.GameStatusCrashed
	now := time.Now()
	game.CrashedAt = &now
	e.gameRepo.UpdateStatus(ctx, game.ID, domain.GameStatusCrashed)

	log.Printf("CRASHED at %s", game.CrashPoint.String())

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
	totalPayouts := decimal.Zero

	for _, bet := range bets {
		if bet.CashoutAt != nil {
			// User cashed out - payout already processed during cashout
			winnersCount++
			totalPayouts = totalPayouts.Add(bet.Payout)
		} else {
			// User didn't cash out - they lose their stake
			// No payout needed, just mark as lost
			bet.Status = domain.GameBetStatusLost
			losersCount++

			// Update bet status in repository
			if err := e.betRepo.UpdateStatus(ctx, bet.ID, domain.GameBetStatusLost); err != nil {
				log.Printf("Error updating bet status to lost: %v", err)
				continue
			}
		}
	}

	log.Printf("Round %d settled: %d winners, %d losers, total payouts: %s",
		game.RoundNumber, winnersCount, losersCount, totalPayouts.String())
}

// ProcessCashout handles a user cashing out during the flight phase
func (e *CrashGameEngine) ProcessCashout(ctx context.Context, betID uuid.UUID, currentMultiplier decimal.Decimal) error {
	// Get the bet details first
	bet, err := e.betRepo.GetByID(ctx, betID)
	if err != nil {
		return fmt.Errorf("get bet: %w", err)
	}

	if bet.Status != domain.GameBetStatusActive {
		return fmt.Errorf("bet is not active: %s", bet.Status)
	}

	// Get the game to determine country code (since GameBet doesn't have CountryCode)
	game, err := e.gameRepo.GetByID(ctx, bet.GameID)
	if err != nil {
		return fmt.Errorf("get game: %w", err)
	}

	// Calculate gross payout: bet.Amount * currentMultiplier
	grossPayout := bet.Amount.Mul(currentMultiplier)

	// Apply winnings tax using tax engine
	payoutBreakdown := e.taxEngine.ApplyPayoutTax(game.CountryCode, grossPayout, bet.Amount)
	netPayout := payoutBreakdown.NetPayout

	// Credit wallet with net payout
	movement := wallet.Movement{
		UserID:        bet.UserID,
		Amount:        netPayout,
		Type:          domain.TransactionTypeBetWon,
		ReferenceID:   &betID,
		ReferenceType: "game_bet",
		Description:   fmt.Sprintf("Crash game cashout at %sx", currentMultiplier.String()),
		ProviderName:  "crash_game",
		ProviderTxnID: betID.String(),
		CountryCode:   game.CountryCode,
	}

	transaction, err := e.walletService.Credit(ctx, bet.UserID, netPayout, movement)
	if err != nil {
		return fmt.Errorf("credit wallet: %w", err)
	}

	// Update bet with cashout details
	if err := e.betRepo.UpdateCashout(ctx, betID, currentMultiplier, netPayout); err != nil {
		return fmt.Errorf("update bet cashout: %w", err)
	}

	// Update bet status to CASHED_OUT
	if err := e.betRepo.UpdateStatus(ctx, betID, domain.GameBetStatusCashedOut); err != nil {
		return fmt.Errorf("update bet status: %w", err)
	}

	log.Printf("Cashout processed: Bet %s at %s = %s (tax: %s, net: %s, txn: %s)",
		betID, currentMultiplier.String(), grossPayout.String(),
		payoutBreakdown.WinningsTax.String(), netPayout.String(), transaction.ID.String())

	return nil
}

// GameState represents the current state of a crash game
type GameState struct {
	GameID            uuid.UUID         `json:"game_id"`
	RoundNumber       int64             `json:"round_number"`
	Status            domain.GameStatus `json:"status"`
	CurrentMultiplier decimal.Decimal   `json:"current_multiplier"`
	CrashPoint        *decimal.Decimal  `json:"crash_point,omitempty"`
	TimeRemaining     time.Duration     `json:"time_remaining,omitempty"`
	ActivePlayers     int               `json:"active_players"`
}
