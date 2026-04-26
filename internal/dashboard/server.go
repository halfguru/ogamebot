package dashboard

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/user/ogame-bot/internal/config"
)

//go:embed static/*
var staticFS embed.FS

// Server is the HTTP server for the web dashboard API.
type Server struct {
	handlers *Handlers
	hub      *Hub
	cfg      config.DashboardConfig
	log      *slog.Logger
}

// GetBroadcaster returns the hub as a Broadcaster interface for passing to workers.
func (s *Server) GetBroadcaster() Broadcaster {
	return s.hub
}

// NewServer creates a new dashboard server.
func NewServer(stateMgr StateReader, db *sql.DB, cfg config.DashboardConfig, log *slog.Logger) *Server {
	hub := NewHub(log)
	handlers := NewHandlers(stateMgr, db, hub, log)

	return &Server{
		handlers: handlers,
		hub:      hub,
		cfg:      cfg,
		log:      log.With("component", "dashboard-server"),
	}
}

// Start launches the HTTP server with all routes and WebSocket support.
// Blocks until context is cancelled (graceful shutdown).
func (s *Server) Start(ctx context.Context) {
	// Start WebSocket hub
	go s.hub.Run(ctx)

	// Register routes using Go 1.22+ method routing
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/planets", s.handlers.handlePlanets)
	mux.HandleFunc("GET /api/fleets", s.handlers.handleFleets)
	mux.HandleFunc("GET /api/research", s.handlers.handleResearch)
	mux.HandleFunc("GET /api/events/builds", s.handlers.handleBuildEvents)
	mux.HandleFunc("GET /api/events/fleet-saves", s.handlers.handleFleetSaveEvents)
	mux.HandleFunc("GET /api/events/farm-attacks", s.handlers.handleFarmAttacks)
	mux.HandleFunc("GET /ws", s.handleWebSocket)

	// Serve embedded static files for the SolidJS dashboard
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		s.log.Error("Failed to create static file system", "error", err)
	} else {
		fileServer := http.FileServer(http.FS(staticSub))
		mux.Handle("GET /assets/", fileServer)
		// SPA catch-all: serve index.html for any non-API, non-WS path
		mux.HandleFunc("GET /", s.serveSPA(fileServer))
	}

	// Wrap with CORS middleware
	handler := s.corsMiddleware(mux)

	addr := fmt.Sprintf(":%d", s.cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		s.log.Info("Dashboard HTTP server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("Dashboard server error", "error", err)
		}
	}()

	// Wait for context cancellation, then graceful shutdown
	<-ctx.Done()
	s.log.Info("Shutting down dashboard server")
	srv.Shutdown(context.Background())
}

// serveSPA returns a handler that serves static files and falls back to index.html
// for SPA client-side routing (any non-API, non-WS GET request).
func (s *Server) serveSPA(fileServer http.Handler) http.HandlerFunc {
	indexHTML, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		s.log.Error("Failed to read embedded index.html", "error", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Try to serve the exact file first
		if path != "" {
			f, err := staticFS.Open("static/" + path)
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fallback to index.html for SPA routing
		if len(indexHTML) > 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexHTML)
			return
		}

		http.NotFound(w, r)
	}
}

// handleWebSocket upgrades an HTTP connection to WebSocket and registers with the hub.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.hub.serveWS(w, r)
}

// corsMiddleware wraps an HTTP handler with CORS headers based on config.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	allowAll := false
	for _, origin := range s.cfg.CorsOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if allowAll && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			for _, allowed := range s.cfg.CorsOrigins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
