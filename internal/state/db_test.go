package state

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestOpenDB_CreatesTables(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	db, err := OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Query table names from sqlite_master
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		t.Fatalf("querying sqlite_master: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scanning table name: %v", err)
		}
		tables = append(tables, name)
	}

	expected := []string{"buildings", "facilities", "fleets", "planets", "research", "resources"}
	sort.Strings(tables)
	sort.Strings(expected)

	// Check that all expected tables exist (schema_migrations is also created)
	for _, exp := range expected {
		found := false
		for _, tbl := range tables {
			if tbl == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected table %q not found in %v", exp, tables)
		}
	}
	// Also verify schema_migrations exists
	found := false
	for _, tbl := range tables {
		if tbl == "schema_migrations" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected table schema_migrations not found in %v", tables)
	}
}

func TestOpenDB_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// First open — creates tables
	db1, err := OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("first OpenDB() error = %v", err)
	}
	db1.Close()

	// Second open — migrations should be no-op
	db2, err := OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("second OpenDB() error = %v", err)
	}
	defer db2.Close()

	// Verify tables still exist
	var count int
	err = db2.QueryRow("SELECT COUNT(*) FROM planets").Err()
	if err != nil {
		t.Errorf("querying planets after second open: %v", err)
	}
	if count < 0 {
		t.Errorf("unexpected count: %d", count)
	}
}

func TestOpenDB_WALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	db, err := OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("querying journal_mode: %v", err)
	}

	if mode != "wal" {
		t.Errorf("expected journal_mode 'wal', got %q", mode)
	}
}

func TestPlanetCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	db, err := OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Insert a planet
	_, err = db.Exec(`INSERT INTO planets (id, name, galaxy, system, position, is_moon, diameter, fields_used, fields_total, temperature_min, temperature_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		12345, "Homeworld", 1, 2, 3, false, 12800, 80, 163, 15, 65)
	if err != nil {
		t.Fatalf("inserting planet: %v", err)
	}

	// Query it back
	var (
		id             int
		name           string
		galaxy, system, position int
		isMoon         bool
		diameter       int
		fieldsUsed     int
		fieldsTotal    int
		tempMin        int
		tempMax        int
	)
	err = db.QueryRow(`SELECT id, name, galaxy, system, position, is_moon, diameter, fields_used, fields_total, temperature_min, temperature_max
		FROM planets WHERE id = ?`, 12345).Scan(
		&id, &name, &galaxy, &system, &position, &isMoon, &diameter, &fieldsUsed, &fieldsTotal, &tempMin, &tempMax)
	if err != nil {
		t.Fatalf("querying planet: %v", err)
	}

	if id != 12345 {
		t.Errorf("id: expected 12345, got %d", id)
	}
	if name != "Homeworld" {
		t.Errorf("name: expected 'Homeworld', got %q", name)
	}
	if galaxy != 1 || system != 2 || position != 3 {
		t.Errorf("coordinate: expected (1,2,3), got (%d,%d,%d)", galaxy, system, position)
	}
	if isMoon {
		t.Errorf("is_moon: expected false, got true")
	}
	if diameter != 12800 {
		t.Errorf("diameter: expected 12800, got %d", diameter)
	}
	if fieldsUsed != 80 || fieldsTotal != 163 {
		t.Errorf("fields: expected 80/163, got %d/%d", fieldsUsed, fieldsTotal)
	}
	if tempMin != 15 || tempMax != 65 {
		t.Errorf("temperature: expected 15/65, got %d/%d", tempMin, tempMax)
	}
}

func TestOpenDB_MaxOpenConns(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	db, err := OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("expected MaxOpenConnections=1, got %d", stats.MaxOpenConnections)
	}
}

func TestOpenDB_ForeignKeysEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	db, err := OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("querying foreign_keys pragma: %v", err)
	}

	if fkEnabled != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fkEnabled)
	}
}

// Ensure database/sql import is used
var _ sql.NullString
