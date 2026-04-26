// Package main is the bot entrypoint.
// Startup: config → logger → DB → rate limiter → client → login → state manager → wait for shutdown.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/defender"
	"github.com/user/ogame-bot/internal/ogamed"
	"github.com/user/ogame-bot/internal/state"
)

func main() {
	// 1. Bootstrap logger for config loading
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// 2. Load config per D-04, D-07
	cfg, err := config.Load("config.yaml", log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 3. Setup structured logging per D-18 (JSON in production)
	level := parseLogLevel(cfg.LogLevel)
	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	log.Info("Starting OGame Bot")

	// 4. Create data directory per D-17
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Error("Failed to create data directory", "error", err)
		os.Exit(1)
	}

	// 5. Open SQLite database per D-08, D-10
	dbPath := filepath.Join(dataDir, "bot.db")
	db, err := state.OpenDB(dbPath, log)
	if err != nil {
		log.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 6. Create ogamed client with rate limiter per D-14, D-11
	rateLimiter := ogamed.NewRateLimiter(cfg.RateLimit)
	client := ogamed.NewClient(cfg.Ogamed.URL, rateLimiter, log)

	// 7. Login to ogamed per INFRA-01
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := client.Login(ctx); err != nil {
		log.Error("Failed to login to ogamed", "error", err)
		os.Exit(1)
	}
	log.Info("Connected to ogamed")

	// 8. Start game state manager per D-09
	stateMgr := state.NewManager(db, client, log)
	go stateMgr.Run(ctx)

	// 8.5. Start defender if enabled
	if cfg.Features.Defender.Enabled {
		def := defender.NewDefender(client, stateMgr, db, cfg.Features.Defender, log)
		go def.Run(ctx)
		log.Info("Defender started", "pollInterval", time.Duration(cfg.Features.Defender.PollIntervalMs)*time.Millisecond)
	}

	log.Info("Bot started successfully")

	// 9. Wait for shutdown signal per RESEARCH.md Pitfall 4
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Info("Shutting down", "signal", sig)
	cancel()
}

// parseLogLevel converts a config log level string to slog.Level.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug", "trace":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
