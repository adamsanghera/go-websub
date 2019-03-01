package sql

// Config is the configuration for the storage object
type Config struct {
	DSN string // the 'data source name', which the sqlite3 client uses to connect

	PersistActive     bool   // determines whether SQLite3 should periodically write to disk
	DataSource        string // path to SQLite3 .db file
	PersistOnShutdown bool   // determines whether SQLite3 should write to disk on shutdown (or wipe on shutdown)
}

// NewConfig returns the default Config (foreign keys on, database in-memory only)
func NewConfig() *Config {
	return &Config{
		DSN: ":memory:?_fk=yes",

		PersistActive:     false,
		PersistOnShutdown: false,
	}
}
