CREATE TABLE IF NOT EXISTS planets (
    id              INTEGER PRIMARY KEY,
    name            TEXT NOT NULL,
    galaxy          INTEGER NOT NULL,
    system          INTEGER NOT NULL,
    position        INTEGER NOT NULL,
    is_moon         BOOLEAN NOT NULL DEFAULT FALSE,
    diameter        INTEGER NOT NULL DEFAULT 0,
    fields_used     INTEGER NOT NULL DEFAULT 0,
    fields_total    INTEGER NOT NULL DEFAULT 0,
    temperature_min INTEGER NOT NULL DEFAULT 0,
    temperature_max INTEGER NOT NULL DEFAULT 0,
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS resources (
    planet_id   INTEGER PRIMARY KEY REFERENCES planets(id),
    metal       INTEGER NOT NULL DEFAULT 0,
    crystal     INTEGER NOT NULL DEFAULT 0,
    deuterium   INTEGER NOT NULL DEFAULT 0,
    energy      INTEGER NOT NULL DEFAULT 0,
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS buildings (
    planet_id               INTEGER PRIMARY KEY REFERENCES planets(id),
    metal_mine              INTEGER NOT NULL DEFAULT 0,
    crystal_mine            INTEGER NOT NULL DEFAULT 0,
    deuterium_synthesizer   INTEGER NOT NULL DEFAULT 0,
    solar_plant             INTEGER NOT NULL DEFAULT 0,
    fusion_reactor          INTEGER NOT NULL DEFAULT 0,
    metal_storage           INTEGER NOT NULL DEFAULT 0,
    crystal_storage         INTEGER NOT NULL DEFAULT 0,
    deuterium_tank          INTEGER NOT NULL DEFAULT 0,
    updated_at              DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS facilities (
    planet_id           INTEGER PRIMARY KEY REFERENCES planets(id),
    robotics_factory    INTEGER NOT NULL DEFAULT 0,
    shipyard            INTEGER NOT NULL DEFAULT 0,
    research_lab        INTEGER NOT NULL DEFAULT 0,
    alliance_depot      INTEGER NOT NULL DEFAULT 0,
    missile_silo        INTEGER NOT NULL DEFAULT 0,
    nanite_factory      INTEGER NOT NULL DEFAULT 0,
    terraformer         INTEGER NOT NULL DEFAULT 0,
    space_dock          INTEGER NOT NULL DEFAULT 0,
    updated_at          DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS research (
    id                          INTEGER PRIMARY KEY CHECK (id = 1),
    energy_technology           INTEGER NOT NULL DEFAULT 0,
    laser_technology            INTEGER NOT NULL DEFAULT 0,
    ion_technology              INTEGER NOT NULL DEFAULT 0,
    hyperspace_technology       INTEGER NOT NULL DEFAULT 0,
    plasma_technology           INTEGER NOT NULL DEFAULT 0,
    combustion_drive            INTEGER NOT NULL DEFAULT 0,
    impulse_drive               INTEGER NOT NULL DEFAULT 0,
    hyperspace_drive            INTEGER NOT NULL DEFAULT 0,
    espionage_technology        INTEGER NOT NULL DEFAULT 0,
    computer_technology         INTEGER NOT NULL DEFAULT 0,
    astrophysics                INTEGER NOT NULL DEFAULT 0,
    intergalactic_research_network INTEGER NOT NULL DEFAULT 0,
    graviton_technology         INTEGER NOT NULL DEFAULT 0,
    weapon_technology           INTEGER NOT NULL DEFAULT 0,
    shielding_technology        INTEGER NOT NULL DEFAULT 0,
    armour_technology           INTEGER NOT NULL DEFAULT 0,
    updated_at                  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS fleets (
    id              INTEGER PRIMARY KEY,
    mission         INTEGER NOT NULL,
    return_flight   BOOLEAN NOT NULL DEFAULT FALSE,
    origin_galaxy   INTEGER NOT NULL,
    origin_system   INTEGER NOT NULL,
    origin_position INTEGER NOT NULL,
    dest_galaxy     INTEGER NOT NULL,
    dest_system     INTEGER NOT NULL,
    dest_position   INTEGER NOT NULL,
    metal           INTEGER NOT NULL DEFAULT 0,
    crystal         INTEGER NOT NULL DEFAULT 0,
    deuterium       INTEGER NOT NULL DEFAULT 0,
    arrival_time    INTEGER NOT NULL DEFAULT 0,
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);
