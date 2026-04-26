-- Farm targets: discovered inactive planets from galaxy scanning
CREATE TABLE IF NOT EXISTS farm_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    galaxy INTEGER NOT NULL,
    system INTEGER NOT NULL,
    position INTEGER NOT NULL,
    player_id INTEGER NOT NULL DEFAULT 0,
    player_name TEXT NOT NULL DEFAULT '',
    is_inactive BOOLEAN NOT NULL DEFAULT FALSE,
    is_long_inactive BOOLEAN NOT NULL DEFAULT FALSE,
    last_scanned_at DATETIME,
    last_espionage_at DATETIME,
    last_attack_at DATETIME,
    metal_loot INTEGER NOT NULL DEFAULT 0,
    crystal_loot INTEGER NOT NULL DEFAULT 0,
    deuterium_loot INTEGER NOT NULL DEFAULT 0,
    has_defense BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(galaxy, system, position)
);

-- Farm attacks: attack history for audit and cooldown tracking
CREATE TABLE IF NOT EXISTS farm_attacks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fleet_id INTEGER NOT NULL,
    planet_id INTEGER NOT NULL,
    target_galaxy INTEGER NOT NULL,
    target_system INTEGER NOT NULL,
    target_position INTEGER NOT NULL,
    ships_sent INTEGER NOT NULL DEFAULT 0,
    metal_looted INTEGER NOT NULL DEFAULT 0,
    crystal_looted INTEGER NOT NULL DEFAULT 0,
    deuterium_looted INTEGER NOT NULL DEFAULT 0,
    sent_at DATETIME NOT NULL DEFAULT (datetime('now')),
    returned_at DATETIME,
    UNIQUE(fleet_id)
);
