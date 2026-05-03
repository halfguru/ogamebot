// Package state manages SQLite database connections, migrations,
// and game state caching from OGameX.
package state

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// OpenDB opens a SQLite database, runs migrations, and returns *sql.DB.
// Per D-08 (modernc.org/sqlite) and D-10 (embedded migrations).
func OpenDB(dbPath string, log *slog.Logger) (*sql.DB, error) {
	// Open with WAL mode and foreign keys enabled
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Per RESEARCH.md Pitfall 1: single writer for SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	if err := runMigrations(db, log); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	log.Info("Database initialized", "path", dbPath)
	return db, nil
}

// runMigrations reads embedded SQL migration files and executes them in order.
// Uses a simple tracking table to ensure idempotency (re-runs are no-ops).
// This replaces golang-migrate to avoid the mattn/go-sqlite3 CGo dependency
// and the m.Close() issue that closes the underlying *sql.DB.
func runMigrations(db *sql.DB, log *slog.Logger) error {
	// Create migration tracking table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	// Read and sort migration files
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := strings.TrimSuffix(entry.Name(), ".sql")

		// Check if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
		if err != nil {
			return fmt.Errorf("checking migration %s: %w", version, err)
		}
		if count > 0 {
			continue // Already applied
		}

		// Read and execute migration
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", version, err)
		}

		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("executing migration %s: %w", version, err)
		}

		// Record as applied
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("recording migration %s: %w", version, err)
		}

		log.Info("Applied migration", "version", version)
	}

	return nil
}
