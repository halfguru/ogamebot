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

	charmlog "github.com/charmbracelet/log"

	"github.com/user/ogame-bot/internal/builder"
	"github.com/user/ogame-bot/internal/colonizer"
	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/dashboard"
	"github.com/user/ogame-bot/internal/defender"
	"github.com/user/ogame-bot/internal/farmer"
	"github.com/user/ogame-bot/internal/ogamex"
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

	// 3. Setup structured logging
	level := parseLogLevel(cfg.LogLevel)
	handler := charmlog.New(os.Stdout)
	handler.SetLevel(charmLogLevel(level))
	handler.SetTimeFormat("15:04:05")
	handler.SetReportTimestamp(true)
	log = slog.New(handler)

	log.Info("Starting OGame Bot")

	startTime := time.Now()

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var client ogamex.ClientInterface
	ogamexCl := ogamex.NewClient(cfg.OGameX.URL, cfg.OGameX.Email, cfg.OGameX.Password, log)
	log.Info("Using OGameX client", "url", cfg.OGameX.URL)
	if err := ogamexCl.Login(ctx); err != nil {
		log.Error("OGameX login failed", "error", err)
		os.Exit(1)
	}
	log.Info("Connected to OGameX")
	client = ogamexCl

	// 8. Start game state manager per D-09
	stateMgr := state.NewManager(db, client, log)
	go stateMgr.Run(ctx)

	// 8.4. Create dashboard server early to get broadcaster for workers
	var broadcaster dashboard.Broadcaster
	var b *builder.Builder
	if cfg.Features.AutoBuild.Enabled {
		b = builder.NewBuilder(client, stateMgr, db, cfg.Features.AutoBuild, log)
		b.SetBroadcaster(broadcaster)
	}
	if cfg.Dashboard.Enabled {
		var planReader dashboard.PlanReader
		if b != nil {
			planReader = b
		}
		dashSrv := dashboard.NewServer(stateMgr, planReader, db, cfg.Dashboard, log, cfg.Features, startTime)
		broadcaster = dashSrv.GetBroadcaster()
		go dashSrv.Start(ctx)
		log.Info("Dashboard started", "port", cfg.Dashboard.Port)
	}
	if b != nil {
		b.SetBroadcaster(broadcaster)
		go b.Run(ctx)
		log.Info("Builder started", "pollInterval", time.Duration(cfg.Features.AutoBuild.PollIntervalMs)*time.Millisecond)
	}

	// 8.5. Start defender if enabled
	if cfg.Features.Defender.Enabled {
		def := defender.NewDefender(client, stateMgr, db, cfg.Features.Defender, log)
		def.SetBroadcaster(broadcaster)
		go def.Run(ctx)
		log.Info("Defender started", "pollInterval", time.Duration(cfg.Features.Defender.PollIntervalMs)*time.Millisecond)
	}

	// 8.7. Start farmer if enabled
	if cfg.Features.AutoFarm.Enabled {
		f := farmer.NewFarmer(client, stateMgr, db, cfg.Features.AutoFarm, log)
		f.SetBroadcaster(broadcaster)
		go f.Run(ctx)
		log.Info("Farmer started", "pollInterval", time.Duration(cfg.Features.AutoFarm.PollIntervalMs)*time.Millisecond)
	}

	if cfg.Features.Colonizer.Enabled {
		col := colonizer.NewColonizer(client, stateMgr, db, cfg.Features.Colonizer, log)
		col.SetBroadcaster(broadcaster)
		go col.Run(ctx)
		log.Info("Colonizer started", "pollInterval", time.Duration(cfg.Features.Colonizer.PollIntervalMs)*time.Millisecond, "targetPlanets", cfg.Features.Colonizer.TargetPlanetCount)
	}

	log.Info("Bot started successfully")

	// 9. Wait for shutdown signal per RESEARCH.md Pitfall 4
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Info("Shutting down", "signal", sig)
	cancel()
}

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

func charmLogLevel(level slog.Level) charmlog.Level {
	switch {
	case level <= slog.LevelDebug:
		return charmlog.DebugLevel
	case level <= slog.LevelInfo:
		return charmlog.InfoLevel
	case level <= slog.LevelWarn:
		return charmlog.WarnLevel
	default:
		return charmlog.ErrorLevel
	}
}
