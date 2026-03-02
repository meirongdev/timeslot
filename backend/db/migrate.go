package db

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS calendars (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    name                TEXT    NOT NULL,
    type                TEXT    NOT NULL DEFAULT 'ical_url',
    url                 TEXT    NOT NULL,
    credentials         TEXT    NOT NULL DEFAULT '',
    sync_interval_min   INTEGER NOT NULL DEFAULT 30,
    last_synced_at      DATETIME,
    enabled             INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS busy_blocks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    calendar_id INTEGER NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    start_at    DATETIME NOT NULL,
    end_at      DATETIME NOT NULL,
    title       TEXT     NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_busy_blocks_time ON busy_blocks(start_at, end_at);

CREATE TABLE IF NOT EXISTS availability_rules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    weekday     INTEGER NOT NULL,   -- 0=Sunday … 6=Saturday
    start_time  TEXT    NOT NULL,   -- "HH:MM"
    end_time    TEXT    NOT NULL,   -- "HH:MM"
    valid_from  DATE,               -- NULL = no lower bound
    valid_until DATE                -- NULL = no upper bound
);
`

func migrate(d *sql.DB) error {
	_, err := d.Exec(schema)
	return err
}
