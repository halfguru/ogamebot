package dashboard

import (
	"database/sql"
	"log/slog"

	"github.com/user/ogame-bot/internal/config"
)

// Server is the HTTP server for the web dashboard API.
type Server struct {
	handlers *Handlers
	hub      *Hub
	cfg      config.DashboardConfig
	log      *slog.Logger
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
