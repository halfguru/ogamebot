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

	"github.com/user/ogame-bot/internal/builder"
	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/dashboard"
	"github.com/user/ogame-bot/internal/defender"
	"github.com/user/ogame-bot/internal/farmer"
	"github.com/user/ogame-bot/internal/ogamed"
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

	// 6b. Create OGameX client (new target)
	{
		_ = cfg.OGameX
		if cfg.OGameX.URL != "" {
			ogamexCl := ogamex.NewClient(cfg.OGameX.URL, cfg.OGameX.Email, cfg.OGameX.Password, log)
			log.Info("OGameX client created", "url", cfg.OGameX.URL)
			_ = ogamexCl
		}
	}

	// 7. Login to ogamed per INFRA-01
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := client.Login(ctx); err != nil {
		log.Warn("Login failed, attempting captcha solve", "error", err)
		const maxRetries = 5
		loggedIn := false
		for attempt := 1; attempt <= maxRetries; attempt++ {
			time.Sleep(2 * time.Second)
			challenge, cerr := client.GetCaptchaChallenge(ctx)
			if cerr != nil {
				log.Warn("Failed to get captcha challenge, retrying", "attempt", attempt, "error", cerr)
				if lerr := client.Login(ctx); lerr != nil {
					log.Warn("Login attempt failed", "attempt", attempt, "error", lerr)
				} else {
					loggedIn = true
					break
				}
				continue
			}
			answer := ogamed.SolveCaptcha(challenge.Icons, challenge.Question)
			log.Info("Captcha solved", "attempt", attempt, "answer", answer, "challengeID", challenge.ID)
			if serr := client.SolveCaptchaChallenge(ctx, challenge.ID, answer); serr != nil {
				log.Error("Failed to submit captcha answer", "attempt", attempt, "error", serr)
				continue
			}
			log.Info("Captcha answer submitted, retrying login")
			time.Sleep(2 * time.Second)
			if lerr := client.Login(ctx); lerr != nil {
				log.Warn("Login still failing after captcha", "attempt", attempt, "error", lerr)
				continue
			}
			loggedIn = true
			break
		}
		if !loggedIn {
			log.Error("Failed to login after captcha attempts")
			os.Exit(1)
		}
	}
	log.Info("Connected to ogamed")

	// 8. Start game state manager per D-09
	stateMgr := state.NewManager(db, client, log)
	go stateMgr.Run(ctx)

	// 8.4. Create dashboard server early to get broadcaster for workers
	var broadcaster dashboard.Broadcaster
	if cfg.Dashboard.Enabled {
		dashSrv := dashboard.NewServer(stateMgr, db, cfg.Dashboard, log)
		broadcaster = dashSrv.GetBroadcaster()
		go dashSrv.Start(ctx)
		log.Info("Dashboard started", "port", cfg.Dashboard.Port)
	}

	// 8.5. Start defender if enabled
	if cfg.Features.Defender.Enabled {
		def := defender.NewDefender(client, stateMgr, db, cfg.Features.Defender, log)
		def.SetBroadcaster(broadcaster)
		go def.Run(ctx)
		log.Info("Defender started", "pollInterval", time.Duration(cfg.Features.Defender.PollIntervalMs)*time.Millisecond)
	}

	// 8.6. Start builder if enabled
	if cfg.Features.AutoBuild.Enabled {
		b := builder.NewBuilder(client, stateMgr, db, cfg.Features.AutoBuild, log)
		b.SetBroadcaster(broadcaster)
		go b.Run(ctx)
		log.Info("Builder started", "pollInterval", time.Duration(cfg.Features.AutoBuild.PollIntervalMs)*time.Millisecond)
	}

	// 8.7. Start farmer if enabled
	if cfg.Features.AutoFarm.Enabled {
		f := farmer.NewFarmer(client, stateMgr, db, cfg.Features.AutoFarm, log)
		f.SetBroadcaster(broadcaster)
		go f.Run(ctx)
		log.Info("Farmer started", "pollInterval", time.Duration(cfg.Features.AutoFarm.PollIntervalMs)*time.Millisecond)
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
