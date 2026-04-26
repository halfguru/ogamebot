CREATE TABLE IF NOT EXISTS build_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    planet_id     INTEGER NOT NULL REFERENCES planets(id),
    building_id   INTEGER NOT NULL,
    building_name TEXT NOT NULL,
    from_level    INTEGER NOT NULL,
    to_level      INTEGER NOT NULL,
    cost_metal    INTEGER NOT NULL DEFAULT 0,
    cost_crystal  INTEGER NOT NULL DEFAULT 0,
    cost_deut     INTEGER NOT NULL DEFAULT 0,
    roi_score     REAL NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_build_events_planet ON build_events(planet_id, created_at DESC);
