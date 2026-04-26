CREATE TABLE IF NOT EXISTS fleet_save_events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    planet_id       INTEGER NOT NULL REFERENCES planets(id),
    fleet_id        INTEGER NOT NULL,          -- fleet ID from SendFleet response
    dest_planet_id  INTEGER NOT NULL,          -- destination planet ID
    attack_id       INTEGER NOT NULL DEFAULT 0, -- attack event that triggered save
    sent_at         DATETIME NOT NULL DEFAULT (datetime('now')),
    recall_at       DATETIME,                  -- when to recall (nil = no recall)
    completed       BOOLEAN NOT NULL DEFAULT FALSE,
    recalled        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_fleet_save_planet_active ON fleet_save_events(planet_id, completed) WHERE completed = FALSE;
