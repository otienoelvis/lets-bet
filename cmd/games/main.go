package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/betting-platform/internal/core/usecase"
	"github.com/betting-platform/internal/core/usecase/games"
	"github.com/betting-platform/internal/infrastructure/database"
	gameshttp "github.com/betting-platform/internal/infrastructure/http"
	"github.com/betting-platform/internal/infrastructure/repository/postgres"
	"github.com/betting-platform/internal/infrastructure/websocket"
	"github.com/gorilla/mux"
)

func main() {
	log.Println("Starting Crash Games Service")

	// Initialize context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()
	log.Println("WebSocket Hub started")

	// Initialize Provably Fair service
	fairService := usecase.NewProvablyFairService()

	// Initialize database connection
	dbConfig := database.GetDefaultConfig()
	db, err := database.NewPostgresConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize PostgreSQL repositories
	gameRepo := postgres.NewGameRepository(db)
	betRepo := postgres.NewGameBetRepository(db)

	// Initialize Crash Game Engine
	engine := games.NewCrashGameEngine(hub, fairService, gameRepo, betRepo)

	// Initialize router
	r := mux.NewRouter()

	// Initialize handlers with engine
	handler := gameshttp.NewGamesHandler(engine)

	// Register routes
	handler.RegisterRoutes(r)

	// Start the game loop
	go engine.Start(ctx)

	log.Println("Crash Game Engine running")
	log.Println("Players can connect via WebSocket to receive real-time updates")

	// Start server
	port := os.Getenv("GAMES_PORT")
	if port == "" {
		port = "8082"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Games Service listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Games Service...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	cancel()

	log.Println("Games Service exited cleanly")
}
