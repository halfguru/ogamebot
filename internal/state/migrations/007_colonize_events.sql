CREATE TABLE IF NOT EXISTS colonize_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    origin_planet_id INTEGER NOT NULL,
    target_galaxy INTEGER NOT NULL,
    target_system INTEGER NOT NULL,
    target_position INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'sent'
);
