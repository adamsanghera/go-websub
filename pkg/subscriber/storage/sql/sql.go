package sql

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3" // Implementation of sqlite3 driver
)

// SQL is a sqlite3 implementation of the subscriber's Storage interface
type SQL struct {
	db *sql.DB

	/* These do not do anything yet!! */
	persistActive     bool   // determines whether SQLite3 should periodically write to disk
	dataSource        string // path to SQLite3 .db file
	persistOnShutdown bool   // determines whether SQLite3 should write to disk on shutdown (or wipe on shutdown)
}

// New creates a new sqlite3 storage object, and returns it
func New(cfg *Config) (*SQL, error) {
	db, err := sql.Open("sqlite3", cfg.DSN)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	// Create tables, views
	if _, err = tx.Exec(subscriptionTable); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(offeredSubscriptionsTable); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(activeView); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(inactiveView); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &SQL{
		db:                db,
		persistActive:     cfg.PersistActive,
		persistOnShutdown: cfg.PersistOnShutdown,
		dataSource:        cfg.DataSource,
	}, nil
}

// Shutdown closes the database
func (sql *SQL) Shutdown() error {
	return sql.db.Close()
}
