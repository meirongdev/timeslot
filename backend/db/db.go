package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens (or creates) the SQLite database and runs migrations.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", path)
	d, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := d.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(d); err != nil {
		return nil, err
	}
	return d, nil
}
